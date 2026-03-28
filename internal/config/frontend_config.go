package config

import (
	"Hamburger/internal/dsl_conf"
	"Hamburger/internal/json"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// FrontCacheConfig 缓存配置
type FrontCacheConfig struct {
	Enable  bool     `json:"enable" toml:"enable"`
	Dir     string   `json:"dir" toml:"dir"`
	Expire  int      `json:"expire" toml:"expire"`
	Matcher []string `json:"matcher" toml:"matcher"`
}

type FrontHttp2Config struct {
	ReadTimeout                  int64 `json:"read_timeout" toml:"read_timeout"`
	WriteTimeout                 int64 `json:"write_timeout" toml:"write_timeout"`
	IdleTimeout                  int64 `json:"idle_timeout" toml:"idle_timeout"`
	ReadHeaderTimeout            int64 `json:"read_header_timeout" toml:"read_header_timeout"`
	MaxHeaderBytes               int64 `json:"max_header_bytes" toml:"max_header_bytes"`
	KeepAlive                    int64 `json:"keep_alive" toml:"keep_alive"`
	MaxHandlers                  int   `yaml:"max_handlers" json:"max_handlers"`                                         // 最大处理协程
	MaxConcurrentStreams         int64 `yaml:"max_concurrent_streams" json:"max_concurrent_streams"`                     // 最大并发流
	MaxReadFrameSize             int64 `yaml:"max_read_frame_size" json:"max_read_frame_size"`                           // 最大帧大小
	MaxDecoderHeaderTableSize    int64 `yaml:"max_decoder_header_table_size" json:"max_decoder_header_table_size"`       // 解码端头表大小
	MaxEncoderHeaderTableSize    int64 `yaml:"max_encoder_header_table_size" json:"max_encoder_header_table_size"`       // 编码端头表大小
	MaxUploadBufferPerConnection int64 `yaml:"max_upload_buffer_per_connection" json:"max_upload_buffer_per_connection"` // 连接级上传缓冲
	MaxUploadBufferPerStream     int64 `yaml:"max_upload_buffer_per_stream" json:"max_upload_buffer_per_stream"`         // 流级上传缓冲
}

// FrontHttp3Config 前端HTTP/3转发配置
type FrontHttp3Config struct {
	Enabled            bool   `json:"enabled" toml:"enabled"`                           // 是否启用
	Host               string `json:"host" toml:"host"`                                 // 目标主机
	Port               int    `json:"port" toml:"port"`                                 // 目标端口
	CertFile           string `json:"cert_file" toml:"cert_file"`                       // 证书文件路径
	KeyFile            string `json:"key_file" toml:"key_file"`                         // 私钥文件路径
	MaxConnections     int    `json:"max_connections" toml:"max_connections"`           // 最大连接数
	IdleTimeout        int64  `json:"idle_timeout" toml:"idle_timeout"`                 // 空闲超时(秒)
	KeepAlive          int64  `json:"keep_alive" toml:"keep_alive"`                     // 保活间隔(秒)
	InsecureSkipVerify bool   `json:"insecure_skip_verify" toml:"insecure_skip_verify"` // 跳过证书校验
}

type FrontFastConnectConfig struct {
	Enabled bool             `json:"enabled" toml:"enabled"`
	Http2   FrontHttp2Config `json:"http2" toml:"http2"`
	Http3   FrontHttp3Config `json:"http3" toml:"http3"` // HTTP/3配置
}

// BackendConfig 后端配置
type BackendConfig struct {
	API          string       `json:"api" toml:"api"`
	Service      string       `json:"service" toml:"service"`
	UseRewrite   bool         `json:"use_rewrite" toml:"use_rewrite"`
	Rewrite      string       `json:"rewrite" toml:"rewrite"`
	IsStatic     bool         `json:"is_static" toml:"is_static"` // 是否为静态目录
	StaticDirect StaticDirect `json:"static_direct" toml:"static_direct"`
	// 通用转发配置
	ProxyDirect ProxyDirect `json:"proxy_direct" toml:"proxy_direct"` // 后端代理转发
}

type ProxyDirect struct {
	ProxyHost string `json:"proxy_host" toml:"proxy_host"`
	ProxyPort int    `json:"proxy_port" toml:"proxy_port"`
}

// StaticDirect 静态代理直连
//
// 对于需要权限控制的静态资源会绕过原本的权限控制
// 拼接方式/API/xxx -> StaticRoot/xxx
type StaticDirect struct {
	DirectAccess bool     `json:"direct_access" toml:"direct_access"` // 直连静态目录
	StaticRoot   string   `json:"static_root" toml:"static_root"`     // 静态根目录
	AllowExt     []string `json:"allow_ext" toml:"allow_ext"`         // 安全配置允许的后缀名
}

// FrontServerConfig 服务器配置
type FrontServerConfig struct {
	Type     string            `json:"type" toml:"type"`
	Name     string            `json:"name" toml:"name"`
	Root     string            `json:"root" toml:"root"`
	Index    string            `json:"index" toml:"index"`
	TryFile  string            `json:"try_file" toml:"try_file"`
	Access   bool              `json:"access" toml:"access"`
	Compress bool              `json:"compress" toml:"compress"`
	Alias    map[string]string `json:"alias" toml:"alias"`
	Backends []BackendConfig   `json:"backends" toml:"backends"`
}

// ErrorConfig 错误页面配置
type ErrorConfig struct {
	NotFound            string `json:"not_found" toml:"not_found"`
	InternalServerError string `json:"internal_server_error" toml:"internal_server_error"`
}

// CustomHeaderConfig 自定义头配置
type CustomHeaderConfig struct {
	Name  string `json:"name" toml:"name"`
	Value string `json:"value" toml:"value"`
}

// PxyFrontConfig 前端服务器配置
type PxyFrontConfig struct {
	Host                string                 `json:"host" toml:"host"`
	Port                int                    `json:"port" toml:"port"`
	Balancer            string                 `json:"balancer" toml:"balancer"`
	ExpFastConnect      FrontFastConnectConfig `json:"exp_fast_connect" toml:"exp_fast_connect"`
	Cache               FrontCacheConfig       `json:"cache" toml:"cache"`
	InternalFlag        string                 `json:"internal_flag" toml:"internal_flag"`
	InternalLocalFlag   string                 `json:"internal_local_flag" toml:"internal_local_flag"`
	InternalBackendFlag string                 `json:"internal_backend_flag" toml:"internal_backend_flag"`
	CacheHeader         string                 `json:"cache_header" toml:"cache_header"`
	Servers             []FrontServerConfig    `json:"servers" toml:"servers"`
	Error               ErrorConfig            `json:"error" toml:"error"`
	CustomHeaders       []CustomHeaderConfig   `json:"custom_headers" toml:"custom_headers"`
}

func LoadFrontConfig(file string) (PxyFrontConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return PxyFrontConfig{}, err
	}
	var cf PxyFrontConfig
	ext := filepath.Ext(file)
	switch ext {
	case ".json":
		err = json.Unmarshal(data, &cf)
		return cf, err
	case ".toml":
		err = toml.Unmarshal(data, &cf)
		return cf, err
	case ".hamburger":
		err = dsl_conf.Unmarshal(data, &cf)
		return cf, err
	default:
		err = json.Unmarshal(data, &cf)
		return cf, err
	}
}
