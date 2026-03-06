package trojan

import (
	"Hamburger/exp/trojan/core/config"
)

type Config struct {
	LocalHost        string `json:"local_addr" yaml:"local-addr"`
	LocalPort        int    `json:"local_port" yaml:"local-port"`
	RemoteHost       string `json:"remote_addr" yaml:"remote-addr"`
	RemotePort       int    `json:"remote_port" yaml:"remote-port"`
	DisableHTTPCheck bool   `json:"disable_http_check" yaml:"disable-http-check"`
}

func init() {
	config.RegisterConfigCreator(Name, func() interface{} {
		return &Config{}
	})
}
