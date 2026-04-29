package lua

import (
	"Hamburger/internal/buildinfo"
	"Hamburger/internal/constant"
	appenv "Hamburger/internal/env"
	"Hamburger/internal/logger"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func registerGlobals(state *lua.LState) {
	state.SetGlobal("log", buildLogTable(state))
	state.SetGlobal("hamburger", buildHamburgerTable(state))
	state.SetGlobal("constant", toLuaTable(state, constant.LuaExport()))
	state.SetGlobal("env", toLuaTable(state, stringMapToAny(appenv.Snapshot())))
}

func buildLogTable(state *lua.LState) *lua.LTable {
	logTable := state.NewTable()
	for _, item := range []struct {
		name string
		fn   func(string)
	}{
		{name: "Info", fn: logInfo},
		{name: "Debug", fn: logDebug},
		{name: "Warn", fn: logWarn},
		{name: "Error", fn: logError},
		{name: "info", fn: logInfo},
		{name: "debug", fn: logDebug},
		{name: "warn", fn: logWarn},
		{name: "error", fn: logError},
	} {
		current := item
		state.SetField(logTable, current.name, state.NewFunction(func(l *lua.LState) int {
			current.fn(l.CheckString(1))
			return 0
		}))
	}
	return logTable
}

func buildHamburgerTable(state *lua.LState) *lua.LTable {
	return toLuaTable(state, map[string]interface{}{
		"app_name":    buildinfo.AppName,
		"description": buildinfo.Description,
		"version":     buildinfo.Version,
		"build_hash":  buildinfo.BuildHash,
	})
}

func logInfo(message string) {
	logger.L().Info().Msg(prefixLog(message))
}

func logDebug(message string) {
	logger.L().Debug().Msg(prefixLog(message))
}

func logWarn(message string) {
	logger.L().Warn().Msg(prefixLog(message))
}

func logError(message string) {
	logger.L().Error().Msg(prefixLog(message))
}

func prefixLog(message string) string {
	return fmt.Sprintf("[hamburger lua] %s", message)
}

func stringMapToAny(values map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func toLuaTable(state *lua.LState, values map[string]interface{}) *lua.LTable {
	table := state.NewTable()
	for key, value := range values {
		state.SetField(table, key, toLuaValue(state, value))
	}
	return table
}

func toLuaValue(state *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case nil:
		return lua.LNil
	case string:
		return lua.LString(v)
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(v)
	case int8:
		return lua.LNumber(v)
	case int16:
		return lua.LNumber(v)
	case int32:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case uint:
		return lua.LNumber(v)
	case uint8:
		return lua.LNumber(v)
	case uint16:
		return lua.LNumber(v)
	case uint32:
		return lua.LNumber(v)
	case uint64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case map[string]string:
		return toLuaTable(state, stringMapToAny(v))
	case map[string]interface{}:
		return toLuaTable(state, v)
	case []string:
		items := state.NewTable()
		for _, item := range v {
			items.Append(lua.LString(item))
		}
		return items
	case []interface{}:
		items := state.NewTable()
		for _, item := range v {
			items.Append(toLuaValue(state, item))
		}
		return items
	default:
		return lua.LString(fmt.Sprint(v))
	}
}
