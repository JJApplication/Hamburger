package lua

import (
	"net/http"
	"net/url"

	lua "github.com/yuin/gopher-lua"
)

func buildRequestTable(state *lua.LState, request *http.Request) *lua.LTable {
	table := state.NewTable()
	state.SetField(table, "method", lua.LString(request.Method))
	state.SetField(table, "host", lua.LString(request.Host))
	state.SetField(table, "path", lua.LString(request.URL.Path))
	state.SetField(table, "raw_query", lua.LString(request.URL.RawQuery))
	state.SetField(table, "url", lua.LString(request.URL.String()))
	state.SetField(table, "remote_addr", lua.LString(request.RemoteAddr))
	state.SetField(table, "header", buildHeaderTable(state, request.Header))
	return table
}

func buildHeaderTable(state *lua.LState, header http.Header) *lua.LTable {
	table := state.NewTable()
	for key, values := range header {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			state.SetField(table, key, lua.LString(values[0]))
			continue
		}
		items := state.NewTable()
		for _, value := range values {
			items.Append(lua.LString(value))
		}
		state.SetField(table, key, items)
	}
	return table
}

func parseRequestResult(value lua.LValue) (*requestResult, error) {
	if value == lua.LNil {
		return nil, nil
	}
	table, ok := value.(*lua.LTable)
	if !ok {
		return nil, newTypeError("request_handle 返回值必须为 table 或 nil")
	}
	result := &requestResult{
		Error:     luaTableString(table, "error"),
		Host:      luaTableString(table, "host"),
		Path:      luaTableString(table, "path"),
		RawQuery:  luaTableString(table, "raw_query"),
		HeaderSet: luaTableStringMap(table, "header_set"),
		HeaderAdd: luaTableStringMap(table, "header_add"),
		HeaderDel: luaTableStringSlice(table, "header_del"),
	}
	if allow := table.RawGetString("allow"); allow != lua.LNil {
		allowed, ok := allow.(lua.LBool)
		if !ok {
			return nil, newTypeError("request_handle.allow 必须为 boolean")
		}
		value := bool(allowed)
		result.Allow = &value
	}
	return result, nil
}

func applyRequestResult(request *http.Request, result *requestResult) {
	if result == nil {
		return
	}
	for key, value := range result.HeaderSet {
		request.Header.Set(key, value)
	}
	for key, value := range result.HeaderAdd {
		request.Header.Add(key, value)
	}
	for _, key := range result.HeaderDel {
		request.Header.Del(key)
	}
	if result.Host != "" {
		request.Host = result.Host
	}
	if result.Path != "" {
		request.URL.Path = result.Path
		request.URL.RawPath = result.Path
	}
	if result.RawQuery != "" {
		request.URL.RawQuery = result.RawQuery
	}
	if result.Path != "" || result.RawQuery != "" {
		request.RequestURI = request.URL.RequestURI()
	}
	if request.URL.Host == "" && request.Host != "" {
		request.URL.Host = request.Host
	}
	if request.URL.Scheme == "" {
		request.URL.Scheme = "http"
	}
}

func luaTableString(table *lua.LTable, key string) string {
	value := table.RawGetString(key)
	if value == lua.LNil {
		return ""
	}
	if str, ok := value.(lua.LString); ok {
		return string(str)
	}
	return value.String()
}

func luaTableStringMap(table *lua.LTable, key string) map[string]string {
	value := table.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	m, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	out := map[string]string{}
	m.ForEach(func(k lua.LValue, v lua.LValue) {
		out[k.String()] = v.String()
	})
	return out
}

func luaTableStringSlice(table *lua.LTable, key string) []string {
	value := table.RawGetString(key)
	if value == lua.LNil {
		return nil
	}
	items, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	out := make([]string, 0, items.Len())
	items.ForEach(func(_ lua.LValue, v lua.LValue) {
		out = append(out, v.String())
	})
	return out
}

func ensureURL(request *http.Request) {
	if request.URL != nil {
		return
	}
	request.URL = &url.URL{}
}
