package structure

import "sync"

// Map 并发安全的泛型Map，支持string键和任意类型值
type Map[T any] struct {
	m  map[string]T
	mu sync.RWMutex
}

// NewMap 创建一个新的泛型Map实例
func NewMap[T any](size ...int) *Map[T] {
	if len(size) > 0 {
		preAlloc := size[0]
		return &Map[T]{
			m:  make(map[string]T, preAlloc),
			mu: sync.RWMutex{},
		}
	}
	return &Map[T]{
		m:  make(map[string]T),
		mu: sync.RWMutex{},
	}
}

func NewSizeMap[T any](size int) *Map[T] {
	return &Map[T]{
		m:  make(map[string]T, size),
		mu: sync.RWMutex{},
	}
}

// Put 存储键值对
func (m *Map[T]) Put(key string, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

// Get 获取指定键的值，返回值和是否存在的标志
func (m *Map[T]) Get(key string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.m[key]
	return value, ok
}

func (m *Map[T]) MustGet(key string) T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.m[key]
	if !ok {
		var zero T
		return zero
	}

	return value
}

// Exist 检查指定键是否存在
func (m *Map[T]) Exist(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.m[key]
	return ok
}

// Delete 删除指定键的元素
func (m *Map[T]) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

// Size 返回Map中元素的数量
func (m *Map[T]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// Keys 返回所有键的切片
func (m *Map[T]) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.m))
	for k := range m.m {
		keys = append(keys, k)
	}
	return keys
}

// Values 返回所有值的切片
func (m *Map[T]) Values() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]T, 0, len(m.m))
	for _, v := range m.m {
		values = append(values, v)
	}
	return values
}

// Clear 清空Map中的所有元素
func (m *Map[T]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m = make(map[string]T)
}

// Range 遍历Map中的所有键值对
// 参数fn是一个函数，接收key和value作为参数，返回bool值
// 如果fn返回false，遍历将停止
// 遍历过程中Map是只读的，保证数据一致性
func (m *Map[T]) Range(fn func(key string, value T) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.m {
		if !fn(k, v) {
			break
		}
	}
}

// KeyValue 表示键值对
type KeyValue[T any] struct {
	Key   string
	Value T
}

// Find 查找第一个满足条件的键值对
// predicate函数返回true表示匹配
// 返回值：key, value, found
func (m *Map[T]) Find(predicate func(key string, value T) bool) (string, T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.m {
		if predicate(k, v) {
			return k, v, true
		}
	}

	var zero T
	return "", zero, false
}

// FindAll 查找所有满足条件的键值对
// predicate函数返回true表示匹配
// 返回值：[]KeyValue 包含所有匹配的键值对
func (m *Map[T]) FindAll(predicate func(key string, value T) bool) []KeyValue[T] {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []KeyValue[T]
	for k, v := range m.m {
		if predicate(k, v) {
			results = append(results, KeyValue[T]{Key: k, Value: v})
		}
	}

	return results
}
