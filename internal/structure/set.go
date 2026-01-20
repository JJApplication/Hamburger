/*
Project: Sandwich structure/set.go
Created: 2025/01/11 by Qoder
*/

package structure

import (
	"sync"
)

// Set 泛型集合，支持任意可比较类型，内部元素不重复
type Set[T comparable] struct {
	items map[T]struct{} // 使用空结构体节省内存
	mu    sync.RWMutex   // 读写锁保证并发安全
}

// NewSet 创建新的Set集合
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		items: make(map[T]struct{}),
	}
}

// NewSetWithItems 使用给定元素创建Set集合
func NewSetWithItems[T comparable](items ...T) *Set[T] {
	s := NewSet[T]()
	for _, item := range items {
		s.Set(item)
	}
	return s
}

// Len 返回集合中元素的数量
func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Get 检查元素是否存在于集合中
func (s *Set[T]) Get(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.items[item]
	return exists
}

// Set 向集合中添加元素（如果元素已存在，不会重复添加）
func (s *Set[T]) Set(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

// List 返回集合中所有元素的切片（顺序不保证）
func (s *Set[T]) List() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]T, 0, len(s.items))
	for item := range s.items {
		result = append(result, item)
	}
	return result
}

// Find 查找满足条件的元素
func (s *Set[T]) Find(predicate func(T) bool) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var zero T
	for item := range s.items {
		if predicate(item) {
			return item, true
		}
	}
	return zero, false
}

// Remove 从集合中移除元素
func (s *Set[T]) Remove(item T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.items[item]; exists {
		delete(s.items, item)
		return true
	}
	return false
}

// Clear 清空集合
func (s *Set[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[T]struct{})
}

// Contains 检查元素是否存在（Get的别名，提供更语义化的方法名）
func (s *Set[T]) Contains(item T) bool {
	return s.Get(item)
}

// Add 添加元素（Set的别名，提供更语义化的方法名）
func (s *Set[T]) Add(item T) {
	s.Set(item)
}

// Union 返回两个集合的并集
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	
	s.mu.RLock()
	for item := range s.items {
		result.Set(item)
	}
	s.mu.RUnlock()
	
	other.mu.RLock()
	for item := range other.items {
		result.Set(item)
	}
	other.mu.RUnlock()
	
	return result
}

// Intersection 返回两个集合的交集
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()
	
	// 遍历较小的集合以提高效率
	if len(s.items) <= len(other.items) {
		for item := range s.items {
			if _, exists := other.items[item]; exists {
				result.Set(item)
			}
		}
	} else {
		for item := range other.items {
			if _, exists := s.items[item]; exists {
				result.Set(item)
			}
		}
	}
	
	return result
}

// Difference 返回两个集合的差集（存在于当前集合但不存在于other集合的元素）
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()
	
	for item := range s.items {
		if _, exists := other.items[item]; !exists {
			result.Set(item)
		}
	}
	
	return result
}

// IsEmpty 检查集合是否为空
func (s *Set[T]) IsEmpty() bool {
	return s.Len() == 0
}