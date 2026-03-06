package trojan

import (
	"Hamburger/exp/trojan/log"
	"encoding/json"
	"os"

	"Hamburger/exp/trojan/common"
)

type liteConfig struct {
	Log      logConfig       `json:"log"`
	Server   serverConfig    `json:"server"`
	Fallback fallbackConfig  `json:"fallback"`
	Auth     authConfig      `json:"auth"`
	TLS      tlsConfig       `json:"tls"`
	Mux      muxConfig       `json:"mux"`
	Router   routerConfig    `json:"router"`
	Plugin   transportPlugin `json:"transport_plugin"`
}

type logConfig struct {
	LogLevel       int    `json:"log_level"`
	LogFile        string `json:"log_file"`
	DisableConsole bool   `json:"disable_console"`
}

type serverConfig struct {
	ListenHost       string `json:"listen_host"`
	ListenPort       int    `json:"listen_port"`
	DisableHTTPCheck bool   `json:"disable_http_check"`
}

type fallbackConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type authConfig struct {
	Passwords []string `json:"passwords"`
}

type tlsConfig struct {
	CertPath           string   `json:"cert"`
	KeyPath            string   `json:"key"`
	KeyPassword        string   `json:"key_password"`
	SNI                string   `json:"sni"`
	ALPN               []string `json:"alpn"`
	Cipher             string   `json:"cipher"`
	PreferServerCipher bool     `json:"prefer_server_cipher"`
	PlainHTTPResponse  string   `json:"plain_http_response"`
	FallbackHost       string   `json:"fallback_addr"`
	FallbackPort       int      `json:"fallback_port"`
	Curves             string   `json:"curves"`
	Fingerprint        string   `json:"fingerprint"`
	KeyLogPath         string   `json:"key_log"`
	CertCheckRate      int      `json:"cert_check_rate"`
}

type muxConfig struct {
	Enabled     bool `json:"enabled"`
	IdleTimeout int  `json:"idle_timeout"`
	Concurrency int  `json:"concurrency"`
}

type routerConfig struct {
	Enabled        bool     `json:"enabled"`
	Bypass         []string `json:"bypass"`
	Proxy          []string `json:"proxy"`
	Block          []string `json:"block"`
	DomainStrategy string   `json:"domain_strategy"`
	DefaultPolicy  string   `json:"default_policy"`
	GeoIP          string   `json:"geoip"`
	GeoSite        string   `json:"geosite"`
}

type transportPlugin struct {
	Enabled bool     `json:"enabled"`
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Option  string   `json:"option"`
	Arg     []string `json:"arg"`
	Env     []string `json:"env"`
}

func loadConfig(path string) (*liteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &liteConfig{
		Log: logConfig{
			LogLevel:       int(log.InfoLevel),
			LogFile:        "hamburger-trojan.log",
			DisableConsole: false,
		},
		Mux: muxConfig{
			Enabled:     true,
			IdleTimeout: 30,
			Concurrency: 8,
		},
		Router: routerConfig{
			Enabled:        false,
			DomainStrategy: "as_is",
			DefaultPolicy:  "proxy",
			GeoIP:          common.GetAssetLocation("geoip.dat"),
			GeoSite:        common.GetAssetLocation("geosite.dat"),
		},
	}
	if err = json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if len(cfg.TLS.ALPN) == 0 {
		cfg.TLS.ALPN = []string{"http/1.1"}
	}
	if cfg.Router.DomainStrategy == "" {
		cfg.Router.DomainStrategy = "as_is"
	}
	if cfg.Router.DefaultPolicy == "" {
		cfg.Router.DefaultPolicy = "proxy"
	}
	if cfg.Router.GeoIP == "" {
		cfg.Router.GeoIP = common.GetAssetLocation("geoip.dat")
	}
	if cfg.Router.GeoSite == "" {
		cfg.Router.GeoSite = common.GetAssetLocation("geosite.dat")
	}
	return cfg, nil
}
