package prehandler

import "sync"

type PreHandlerManager struct {
	lock      sync.RWMutex
	modifiers []PreHandler
}

var (
	m    *PreHandlerManager
	once sync.Once
)

func GetManager() *PreHandlerManager {
	once.Do(func() {
		if m == nil {
			m = NewModifierManager()
		}
	})
	return m
}

// NewModifierManager 创建新的修改器管理器
func NewModifierManager() *PreHandlerManager {
	manager := &PreHandlerManager{
		lock:      sync.RWMutex{},
		modifiers: make([]PreHandler, 0),
	}

	return manager
}

func InitPreHandlerManager() {
	pm := GetManager()
	pm.Add(NewHeaderSanitizer())
	pm.Add(NewPreCheckDomains())
	pm.Add(NewRateLimiter())
	pm.Add(NewImageProtectModifier())
}

func (m *PreHandlerManager) Add(ph PreHandler) {
	m.lock.Lock()
	m.modifiers = append(m.modifiers, ph)
	m.lock.Unlock()
}

func (m *PreHandlerManager) GetPreHandlers() []PreHandler {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.modifiers
}
