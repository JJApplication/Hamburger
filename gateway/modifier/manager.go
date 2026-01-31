/*
Create: 2025/1/15
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package modifier

import (
	"net/http"
	"sync"

	"Hamburger/internal/logger"
)

// ModifierManager 修改器管理器
// 负责管理和协调所有响应修改器
type ModifierManager struct {
	chain     *ModifierChain
	lock      sync.RWMutex
	modifiers []Modifier
}

var (
	m    *ModifierManager
	once sync.Once
)

func GetManager() *ModifierManager {
	once.Do(func() {
		if m == nil {
			m = NewModifierManager()
		}
	})
	return m
}

// NewModifierManager 创建新的修改器管理器
func NewModifierManager() *ModifierManager {
	manager := &ModifierManager{
		lock:      sync.RWMutex{},
		chain:     NewModifierChain(),
		modifiers: make([]Modifier, 0),
	}

	return manager
}

// InitModifiers 注册全部中间件 根据配置激活
func InitModifiers() {
	mm := GetManager()
	// add trace
	mm.RegisterModifier(NewTraceModifier())
	// add secure header
	mm.RegisterModifier(NewSecureHeaderModifier())
	// no cache
	mm.RegisterModifier(NewNoCache())
	// custom header
	mm.RegisterModifier(NewCustomHeaderModifier())
	// 应用gzip压缩中间件
	mm.RegisterModifier(NewGzipModifier())
	// 应用cors
	mm.RegisterModifier(NewCorsHeaderModifier())
}

func (mm *ModifierManager) RegisterModifier(modifier Modifier) {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	mm.chain.AddModifier(modifier)
}

// ModifyResponse 对响应应用所有启用的修改器
func (mm *ModifierManager) ModifyResponse(response *http.Response) error {
	return mm.chain.ModifyResponse(response)
}

// UpdateConfig 更新所有修改器的配置
func (mm *ModifierManager) UpdateConfig() {
	mm.chain.UpdateConfig()
	logger.GetLogger().Debug().Msg("all modifier configurations updated")
}

// GetEnabledModifiers 获取所有启用的修改器
func (mm *ModifierManager) GetEnabledModifiers() []Modifier {
	return mm.chain.GetEnabledModifiers()
}

// GetModifierByName 根据名称获取修改器
func (mm *ModifierManager) GetModifierByName(name string) Modifier {
	return mm.chain.GetModifierByName(name)
}

// AddCustomModifier 添加自定义修改器
func (mm *ModifierManager) AddCustomModifier(modifier Modifier) {
	if modifier != nil {
		mm.chain.AddModifier(modifier)
		logger.GetLogger().Debug().Str("name", modifier.GetName()).Msg("custom modifier added")
	}
}

// GetGzipModifier 获取gzip修改器实例（类型安全的访问方式）
func (mm *ModifierManager) GetGzipModifier() *GzipModifier {
	modifier := mm.GetModifierByName("gzip")
	if gzipModifier, ok := modifier.(*GzipModifier); ok {
		return gzipModifier
	}
	return nil
}

// GetCustomHeaderModifier 获取自定义头修改器实例（类型安全的访问方式）
func (mm *ModifierManager) GetCustomHeaderModifier() *CustomHeaderModifier {
	modifier := mm.GetModifierByName("custom_header")
	if customHeaderModifier, ok := modifier.(*CustomHeaderModifier); ok {
		return customHeaderModifier
	}
	return nil
}

// GetStatus 获取修改器管理器状态信息
func (mm *ModifierManager) GetStatus() map[string]interface{} {
	enabledModifiers := mm.GetEnabledModifiers()

	status := map[string]interface{}{
		"total_modifiers":   len(mm.chain.modifiers),
		"enabled_modifiers": len(enabledModifiers),
		"modifiers":         make([]map[string]interface{}, 0),
	}

	for _, modifier := range mm.chain.modifiers {
		modifierInfo := map[string]interface{}{
			"name":    modifier.GetName(),
			"enabled": modifier.IsEnabled(),
		}

		// 添加特定修改器的详细信息
		switch m := modifier.(type) {
		case *GzipModifier:
			modifierInfo["level"] = m.GetLevel()
			modifierInfo["types"] = m.GetTypes()
		case *CustomHeaderModifier:
			modifierInfo["headers_count"] = len(m.GetHeaders())
		}

		status["modifiers"] = append(status["modifiers"].([]map[string]interface{}), modifierInfo)
	}

	return status
}

func (mm *ModifierManager) GetModifiers() []Modifier {
	mm.lock.RLock()
	defer mm.lock.RUnlock()
	return mm.chain.GetEnabledModifiers()
}
