package structure

import "sync"

// 并发安全的带类型map 不使用泛型实现

type MapStruct struct {
	m  map[string]struct{}
	mu sync.RWMutex
}

// NewMapStruct 预分配内存
func NewMapStruct(size int) *MapStruct {
	return &MapStruct{
		m:  make(map[string]struct{}, size),
		mu: sync.RWMutex{},
	}
}

func (m *MapStruct) Put(key string) {
	m.mu.Lock()
	m.m[key] = struct{}{}
	m.mu.Unlock()
}

func (m *MapStruct) Get(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.m[key]
	return ok
}

func (m *MapStruct) Exist(key string) bool {
	return m.Get(key)
}

func (m *MapStruct) Delete(key string) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}
