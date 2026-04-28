package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func loadScripts(state *lua.LState, root string) ([]middleware, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("lua scripts root is not a directory: %s", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".lua") {
			continue
		}
		paths = append(paths, filepath.Join(root, entry.Name()))
	}
	sort.Strings(paths)
	middlewares := make([]middleware, 0, len(paths))
	for _, path := range paths {
		if err := state.DoFile(path); err != nil {
			return nil, fmt.Errorf("load lua script %s failed: %w", path, err)
		}
		if fn, ok := state.GetGlobal(requestHandleFuncName).(*lua.LFunction); ok {
			middlewares = append(middlewares, middleware{
				scriptPath: path,
				function:   fn,
			})
			state.SetGlobal(requestHandleFuncName, lua.LNil)
		}
	}
	return middlewares, nil
}
