package initialize

import (
	applua "Hamburger/internal/lua"
)

// InitLuaVM 初始化全局Lua虚拟机。
func (i *Initializer) InitLuaVM() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			return applua.Init(i.cfg.Lua, i.logger)
		},
	}
}
