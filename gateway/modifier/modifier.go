/*
Create: 2025/1/15
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package modifier

import (
	"net/http"
)

// Modifier 响应修改器接口
// 所有响应修改器都需要实现此接口
type Modifier interface {
	Use(response *http.Response)
	// ModifyResponse 修改HTTP响应
	// 在响应返回给客户端之前对响应进行处理
	ModifyResponse(response *http.Response) error

	// IsEnabled 检查修改器是否启用
	IsEnabled() bool

	// UpdateConfig 更新配置（支持热更新）
	UpdateConfig()

	// GetName 获取修改器名称
	GetName() string
}

// ModifierChain 修改器链
// 用于按顺序执行多个修改器
type ModifierChain struct {
	modifiers []Modifier
}

// NewModifierChain 创建新的修改器链
func NewModifierChain() *ModifierChain {
	return &ModifierChain{
		modifiers: make([]Modifier, 0),
	}
}

// AddModifier 添加修改器到链中
func (mc *ModifierChain) AddModifier(modifier Modifier) {
	if modifier != nil {
		mc.modifiers = append(mc.modifiers, modifier)
	}
}

// ModifyResponse 执行所有启用的修改器
func (mc *ModifierChain) ModifyResponse(response *http.Response) error {
	for _, modifier := range mc.modifiers {
		if modifier.IsEnabled() {
			if err := modifier.ModifyResponse(response); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateConfig 更新所有修改器的配置
func (mc *ModifierChain) UpdateConfig() {
	for _, modifier := range mc.modifiers {
		modifier.UpdateConfig()
	}
}

// GetEnabledModifiers 获取所有启用的修改器
func (mc *ModifierChain) GetEnabledModifiers() []Modifier {
	enabled := make([]Modifier, 0)
	for _, modifier := range mc.modifiers {
		if modifier.IsEnabled() {
			enabled = append(enabled, modifier)
		}
	}
	return enabled
}

// GetModifierByName 根据名称获取修改器
func (mc *ModifierChain) GetModifierByName(name string) Modifier {
	for _, modifier := range mc.modifiers {
		if modifier.GetName() == name {
			return modifier
		}
	}
	return nil
}
