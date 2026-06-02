package precheck

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"Hamburger/gateway/precheck_cache"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
)

const (
	defaultPathPrefix = "/__precheck"
	defaultTTLSeconds = int64(3600)
	defaultCacheMaxMB = 64
)

// ShouldBypassHTTPRedirect 判断请求是否应跳过 HTTP→HTTPS 自动跳转。
// 已开启 pre_check 的域名须在 HTTP 监听上先完成 challenge，不能被外层 301 抢先转到 HTTPS（否则内层 precheck 不会执行）。
func ShouldBypassHTTPRedirect(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	path := req.URL.Path
	if strings.HasPrefix(path, defaultPathPrefix) {
		return true
	}
	host := runtime.NormalizeRequestHost(req.Host)
	if svc, ok := runtime.GetDomain2Service(host); ok && svc.PreCheck.Enabled {
		return true
	}
	if svc, ok := runtime.FindPrecheckEnabledService(); ok {
		pc := normalizePreCheckConfig(svc.PreCheck)
		if strings.HasPrefix(path, pc.PathPrefix) {
			return true
		}
	}
	return false
}

type pageData struct {
	ReturnURL  string
	VerifyPath string
	RequestID  string
}

// Router 在反代 handler 外包一层前置检查（与 core.GlobalStaticAlias 相同模式）。
type Router struct {
	next     http.Handler
	cacheMgr *precheck_cache.Manager
	tpl      *template.Template
}

func NewRouter(next http.Handler) *Router {
	tpl, err := template.ParseFS(assetsFS, "templates/precheck.tmpl")
	if err != nil {
		tpl = template.New("precheck")
	}
	return &Router{
		next:     next,
		cacheMgr: precheck_cache.NewManager(),
		tpl:      tpl,
	}
}

func (r *Router) Handler() http.Handler {
	return http.HandlerFunc(r.serveHTTP)
}

func (r *Router) serveHTTP(w http.ResponseWriter, req *http.Request) {
	svc, pc, ok := resolveServiceForRequest(req)
	if !ok || !pc.Enabled {
		r.next.ServeHTTP(w, req)
		return
	}

	if strings.HasPrefix(req.URL.Path, pc.PathPrefix) {
		r.handlePrecheck(w, req, svc, pc)
		return
	}

	if shouldSkipPrecheck(req.URL.Path, pc) {
		r.next.ServeHTTP(w, req)
		return
	}

	remoteKey := normalizedRemoteKey(req.RemoteAddr)
	requestID := makeRequestID(svc.ServiceName, remoteKey)

	cache, cacheErr := r.cacheMgr.GetOrCreate(
		svc.ServiceName,
		precheck_cache.DefaultCacheConfig(time.Duration(pc.TTLSeconds)*time.Second, pc.CacheMaxMB),
	)
	if cacheErr != nil {
		r.next.ServeHTTP(w, req)
		return
	}
	if cache.IsValid(requestID, time.Now()) {
		r.next.ServeHTTP(w, req)
		return
	}

	returnURL := req.URL.RequestURI()
	redirectTo := pc.PathPrefix + "?u=" + url.QueryEscape(returnURL)
	http.Redirect(w, req, redirectTo, http.StatusFound)
}

func (r *Router) handlePrecheck(w http.ResponseWriter, req *http.Request, svc config.Service, pc config.PreCheckConfig) {
	path := strings.TrimPrefix(req.URL.Path, pc.PathPrefix)
	switch {
	case path == "" || path == "/":
		r.handlePrecheckPage(w, req, svc, pc)
	case path == "/verify":
		r.handleVerify(w, req, svc, pc)
	default:
		http.NotFound(w, req)
	}
}

func (r *Router) handlePrecheckPage(w http.ResponseWriter, req *http.Request, svc config.Service, pc config.PreCheckConfig) {
	returnURL := sanitizeReturnURL(req.URL.Query().Get("u"))

	remoteKey := normalizedRemoteKey(req.RemoteAddr)
	requestID := makeRequestID(svc.ServiceName, remoteKey)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.tpl == nil {
		http.Error(w, "template not loaded", http.StatusInternalServerError)
		return
	}
	_ = r.tpl.ExecuteTemplate(w, "precheck", pageData{
		ReturnURL:  returnURL,
		VerifyPath: pc.PathPrefix + "/verify",
		RequestID:  requestID,
	})
}

func (r *Router) handleVerify(w http.ResponseWriter, req *http.Request, svc config.Service, pc config.PreCheckConfig) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := req.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	returnURL := sanitizeReturnURL(strings.TrimSpace(req.FormValue("u")))

	remoteKey := normalizedRemoteKey(req.RemoteAddr)
	requestID := makeRequestID(svc.ServiceName, remoteKey)

	cache, err := r.cacheMgr.GetOrCreate(
		svc.ServiceName,
		precheck_cache.DefaultCacheConfig(time.Duration(pc.TTLSeconds)*time.Second, pc.CacheMaxMB),
	)
	if err == nil && cache != nil {
		_ = cache.Set(requestID, time.Now().Add(time.Duration(pc.TTLSeconds)*time.Second))
	}

	http.Redirect(w, req, returnURL, http.StatusFound)
}

func sanitizeReturnURL(returnURL string) string {
	returnURL = strings.TrimSpace(returnURL)
	returnURL = strings.Trim(returnURL, `"`)
	if returnURL == "" {
		return "/"
	}
	if strings.HasPrefix(returnURL, "http://") || strings.HasPrefix(returnURL, "https://") {
		return "/"
	}
	if unescaped, err := url.PathUnescape(returnURL); err == nil {
		returnURL = unescaped
	}
	returnURL = strings.ReplaceAll(returnURL, `"`, "")
	returnURL = strings.TrimSpace(returnURL)
	if returnURL == "" {
		return "/"
	}
	if strings.Contains(returnURL, "..") {
		return "/"
	}
	if !strings.HasPrefix(returnURL, "/") {
		returnURL = "/" + strings.TrimLeft(returnURL, "/")
	}
	returnURL = path.Clean(returnURL)
	if returnURL == "." || returnURL == "" {
		return "/"
	}
	if !strings.HasPrefix(returnURL, "/") {
		returnURL = "/" + returnURL
	}
	return returnURL
}

// resolveServiceForRequest 按 Host 解析服务；若直接访问 challenge 路径且 Host 为 IP 等未映射域名，
// 则回退到第一个开启 pre_check 的服务（便于本地调试）。
func resolveServiceForRequest(req *http.Request) (config.Service, config.PreCheckConfig, bool) {
	host := runtime.NormalizeRequestHost(req.Host)
	if svc, ok := runtime.GetDomain2Service(host); ok {
		pc := normalizePreCheckConfig(svc.PreCheck)
		return svc, pc, true
	}
	if svc, ok := runtime.FindPrecheckEnabledService(); ok {
		pc := normalizePreCheckConfig(svc.PreCheck)
		if strings.HasPrefix(req.URL.Path, pc.PathPrefix) {
			return svc, pc, true
		}
	}
	return config.Service{}, config.PreCheckConfig{}, false
}

func normalizePreCheckConfig(pc config.PreCheckConfig) config.PreCheckConfig {
	if strings.TrimSpace(pc.PathPrefix) == "" {
		pc.PathPrefix = defaultPathPrefix
	}
	if !strings.HasPrefix(pc.PathPrefix, "/") {
		pc.PathPrefix = "/" + pc.PathPrefix
	}
	if len(pc.PathPrefix) > 1 {
		pc.PathPrefix = strings.TrimRight(pc.PathPrefix, "/")
	}
	if pc.TTLSeconds <= 0 {
		pc.TTLSeconds = defaultTTLSeconds
	}
	if pc.CacheMaxMB <= 0 {
		pc.CacheMaxMB = defaultCacheMaxMB
	}
	if pc.ExcludePaths == nil {
		pc.ExcludePaths = []string{}
	}
	if pc.ExcludeExtensions == nil {
		pc.ExcludeExtensions = []string{}
	}
	pc.ExcludeExtensions = normalizeExcludeExtensions(pc.ExcludeExtensions)
	return pc
}

func shouldSkipPrecheck(requestPath string, pc config.PreCheckConfig) bool {
	return hasExcludedPrefix(requestPath, pc.ExcludePaths) ||
		hasExcludedExtension(requestPath, pc.ExcludeExtensions)
}

func hasExcludedPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func normalizeExcludeExtensions(exts []string) []string {
	out := make([]string, 0, len(exts))
	seen := make(map[string]struct{}, len(exts))
	for _, ext := range exts {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if _, ok := seen[ext]; ok {
			continue
		}
		seen[ext] = struct{}{}
		out = append(out, ext)
	}
	return out
}

func hasExcludedExtension(requestPath string, exts []string) bool {
	if len(exts) == 0 {
		return false
	}
	ext := strings.ToLower(path.Ext(path.Base(requestPath)))
	if ext == "" {
		return false
	}
	for _, allowed := range exts {
		if ext == allowed {
			return true
		}
	}
	return false
}

func normalizedRemoteKey(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	host, port, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" && port != "" {
		return host + ":" + port
	}
	return remoteAddr
}

func makeRequestID(serviceName, remoteKey string) string {
	sum := sha256.Sum256([]byte(serviceName + "|" + remoteKey))
	return hex.EncodeToString(sum[:16])
}
