package lua

import lua "github.com/yuin/gopher-lua"

const requestHandleFuncName = "request_handle"

type middleware struct {
	scriptPath string
	function   *lua.LFunction
}

type requestResult struct {
	Allow     *bool
	Error     string
	Host      string
	Path      string
	RawQuery  string
	HeaderSet map[string]string
	HeaderAdd map[string]string
	HeaderDel []string
}
