package wasm_plugin

import "github.com/tetratelabs/wazero/api"

type plugin struct {
	name       string
	enabled    bool
	params     map[string]interface{}
	module     api.Module
	requestFn  api.Function
	responseFn api.Function
	allocFn    api.Function
	freeFn     api.Function
	memory     api.Memory
}

type requestContext struct {
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Host       string              `json:"host"`
	RemoteAddr string              `json:"remote_addr"`
	Header     map[string][]string `json:"header"`
}

type responseContext struct {
	StatusCode int                 `json:"status_code"`
	Header     map[string][]string `json:"header"`
}

type requestResult struct {
	Allow     *bool             `json:"allow"`
	Error     string            `json:"error"`
	Status    int               `json:"status"`
	HeaderSet map[string]string `json:"header_set"`
	HeaderAdd map[string]string `json:"header_add"`
	HeaderDel []string          `json:"header_del"`
}

type responseResult struct {
	Status    int               `json:"status"`
	HeaderSet map[string]string `json:"header_set"`
	HeaderAdd map[string]string `json:"header_add"`
	HeaderDel []string          `json:"header_del"`
}
