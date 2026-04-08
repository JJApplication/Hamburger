package main

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

func main() {
	configPath := "config/config.json"
	flag.StringVar(&configPath, "c", "config/config.json", "hamburger config file path")
	flag.StringVar(&configPath, "config", "config/config.json", "hamburger config file path")
	flag.Parse()

	resolvedConfigPath, err := resolvePath(configPath, "")
	if err != nil {
		writeErrorAndExit(fmt.Sprintf("解析配置文件路径失败: %v", err))
	}

	appCfg, err := loader.LoadConfig(resolvedConfigPath)
	if err != nil {
		writeErrorAndExit(fmt.Sprintf("加载主配置失败: %v", err))
	}

	configBaseDir := filepath.Dir(resolvedConfigPath)
	appCfg.PxyFrontendFile = normalizeSubConfigPath(configBaseDir, appCfg.PxyFrontendFile)
	appCfg.PxyBackendFile = normalizeSubConfigPath(configBaseDir, appCfg.PxyBackendFile)

	validationErrors := validateSubConfigs(appCfg)
	merged := loader.Merge(appCfg)
	if merged == nil {
		writeErrorAndExit("合并配置失败")
	}

	fmt.Printf("主配置加载成功: %s\n\n", resolvedConfigPath)
	renderFrontServerTable(merged)
	renderBackendServerTable(merged)
	renderGatewayConfigTable(merged)
	renderExpConfigTable(merged)

	if len(validationErrors) > 0 {
		for _, item := range validationErrors {
			_, _ = os.Stderr.WriteString(item + "\n")
		}
		os.Exit(1)
	}

	fmt.Println("配置校验通过")
}

func validateSubConfigs(appCfg *config.AppConfig) []string {
	var errs []string
	if appCfg == nil {
		return []string{"主配置为空"}
	}
	if strings.TrimSpace(appCfg.PxyFrontendFile) != "" {
		if !fileExists(appCfg.PxyFrontendFile) {
			appCfg.PxyFrontendFile = ""
		} else {
			if _, err := config.LoadFrontConfig(appCfg.PxyFrontendFile); err != nil {
				errs = append(errs, fmt.Sprintf("前端配置加载失败: %s (%v)", appCfg.PxyFrontendFile, err))
			}
		}
	}
	if strings.TrimSpace(appCfg.PxyBackendFile) != "" {
		if !fileExists(appCfg.PxyBackendFile) {
			appCfg.PxyBackendFile = ""
		} else {
			if _, err := config.LoadBackendConfig(appCfg.PxyBackendFile); err != nil {
				errs = append(errs, fmt.Sprintf("后端配置加载失败: %s (%v)", appCfg.PxyBackendFile, err))
			}
		}
	}
	return errs
}

func renderFrontServerTable(cfg *config.Config) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Front服务器配置")
	tw.AppendHeader(table.Row{"名称", "类型", "根目录", "后端数量", "压缩", "访问"})
	for _, server := range cfg.PxyFrontend.Servers {
		tw.AppendRow(table.Row{
			server.Name,
			server.Type,
			server.Root,
			len(server.Backends),
			formatBool(server.Compress),
			formatBool(server.Access),
		})
	}
	if len(cfg.PxyFrontend.Servers) == 0 {
		tw.AppendRow(table.Row{"-", "-", "-", 0, "-", "-"})
	}
	tw.Render()
	fmt.Println()
}

func renderBackendServerTable(cfg *config.Config) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Backend服务器配置")
	tw.AppendHeader(table.Row{"服务名", "类型", "监听地址", "静态目录", "TCP目标"})
	for _, server := range cfg.PxyBackend.Servers {
		listen := fmt.Sprintf("%s:%d", server.Host, server.Port)
		staticDir := server.Http.StaticDir
		if strings.TrimSpace(staticDir) == "" {
			staticDir = server.WebDav.Root
		}
		tcpTarget := server.Tcp.Target
		if strings.TrimSpace(tcpTarget) == "" {
			tcpTarget = "-"
		}
		if strings.TrimSpace(staticDir) == "" {
			staticDir = "-"
		}
		tw.AppendRow(table.Row{
			server.ServiceName,
			server.Type,
			listen,
			staticDir,
			tcpTarget,
		})
	}
	if len(cfg.PxyBackend.Servers) == 0 {
		tw.AppendRow(table.Row{"-", "-", "-", "-", "-"})
	}
	tw.Render()
	fmt.Println()
}

func renderGatewayConfigTable(cfg *config.Config) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Gateway配置")
	tw.AppendHeader(table.Row{"配置项", "值"})
	tw.AppendRows([]table.Row{
		{"Transport", cfg.CoreProxy.Transport},
		{"ProxyMode", cfg.CoreProxy.ProxyMode},
		{"NetIO", cfg.CoreProxy.NetIO},
		{"MaxConnsPerHost", cfg.CoreProxy.MaxConnsPerHost},
		{"GatewayServers", len(cfg.Servers)},
		{"Trace", formatBool(cfg.Middleware.Trace.Enabled)},
		{"Gzip", formatBool(cfg.Middleware.Gzip.Enabled)},
		{"CORS", formatBool(cfg.Middleware.CORS.Enabled)},
		{"FlowControl", formatBool(cfg.Features.FlowControl.Enabled)},
		{"WebSocket", formatBool(cfg.Features.WebSocket.Enabled)},
		{"HTTP3", formatBool(cfg.Features.HTTP3.Enabled)},
		{"ProxyCache", formatBool(cfg.Features.ProxyCache.Enabled)},
		{"APIServer", formatBool(cfg.ApiServerConfig.Enabled)},
		{"PreAuth", formatBool(cfg.PreAuthConfig.Enabled)},
		{"Notify", formatBool(cfg.NotifyConfig.Enabled)},
	})
	tw.Render()
	fmt.Println()
}

func renderExpConfigTable(cfg *config.Config) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Exp配置")
	tw.AppendHeader(table.Row{"配置项", "值"})
	tw.AppendRows([]table.Row{
		{"VpnServerEnabled", formatBool(cfg.ExpConfig.VpnServer.Enabled)},
		{"VpnHTTPListen", fmt.Sprintf("%s:%d", cfg.ExpConfig.VpnServer.Host, cfg.ExpConfig.VpnServer.HttpPort)},
		{"VpnSocksListen", fmt.Sprintf("%s:%d", cfg.ExpConfig.VpnServer.Host, cfg.ExpConfig.VpnServer.SocksPort)},
		{"AnyTLSEnabled", formatBool(cfg.ExpConfig.AnyTLSServer.Enabled)},
		{"AnyTLSListen", fmt.Sprintf("%s:%d", cfg.ExpConfig.AnyTLSServer.Host, cfg.ExpConfig.AnyTLSServer.Port)},
		{"TrojanConfig", cfg.ExpConfig.TrojanServer},
	})
	tw.Render()
	fmt.Println()
}

func normalizeSubConfigPath(baseDir string, file string) string {
	trimmed := strings.TrimSpace(file)
	if trimmed == "" {
		return ""
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}

	candidates := []string{filepath.Join(baseDir, trimmed)}
	if filepath.Base(baseDir) == "config" {
		normalized := filepath.ToSlash(trimmed)
		if strings.HasPrefix(normalized, "config/") {
			candidates = append(candidates, filepath.Join(baseDir, strings.TrimPrefix(normalized, "config/")))
		}
	}
	for _, candidate := range candidates {
		absCandidate, err := resolvePath(candidate, "")
		if err == nil && fileExists(absCandidate) {
			return absCandidate
		}
	}

	resolved, err := resolvePath(filepath.Join(baseDir, trimmed), "")
	if err != nil {
		return trimmed
	}
	return resolved
}

func resolvePath(path string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("空路径")
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	if strings.TrimSpace(baseDir) != "" {
		return filepath.Clean(filepath.Join(baseDir, trimmed)), nil
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func formatBool(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

func writeErrorAndExit(msg string) {
	_, _ = os.Stderr.WriteString(msg + "\n")
	os.Exit(1)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
