/*
Project: Sandwich structure/ordered_map.go
Created: 2025/01/11 by Qoder
*/

package structure

import (
	"sync"
)

// OrderedMapEntry 有序字典的条目
type OrderedMapEntry[K comparable, V any] struct {
	Key   K
	Value V
	prev  *OrderedMapEntry[K, V]
	next  *OrderedMapEntry[K, V]
}

// OrderedMap 有序字典，保持插入顺序
type OrderedMap[K comparable, V any] struct {
	items map[K]*OrderedMapEntry[K, V] // 用于快速查找
	head  *OrderedMapEntry[K, V]       // 链表头节点
	tail  *OrderedMapEntry[K, V]       // 链表尾节点
	mu    sync.RWMutex                 // 读写锁保证并发安全
}

// NewOrderedMap 创建新的有序字典
func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		items: make(map[K]*OrderedMapEntry[K, V]),
	}
}

// Len 返回字典中元素的数量
func (om *OrderedMap[K, V]) Len() int {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return len(om.items)
}

// Get 根据键获取值
func (om *OrderedMap[K, V]) Get(key K) (V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	if entry, exists := om.items[key]; exists {
		return entry.Value, true
	}
	
	var zero V
	return zero, false
}

// Set 设置键值对，如果键已存在则更新值，否则添加新的键值对
func (om *OrderedMap[K, V]) Set(key K, value V) {
	om.mu.Lock()
	defer om.mu.Unlock()
	
	if entry, exists := om.items[key]; exists {
		// 键已存在，更新值
		entry.Value = value
	} else {
		// 键不存在，创建新条目并添加到链表尾部
		entry := &OrderedMapEntry[K, V]{
			Key:   key,
			Value: value,
		}
		
		om.items[key] = entry
		om.appendToTail(entry)
	}
}

// List 返回所有键值对的有序列表
func (om *OrderedMap[K, V]) List() []*OrderedMapEntry[K, V] {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	result := make([]*OrderedMapEntry[K, V], 0, len(om.items))
	current := om.head
	for current != nil {
		result = append(result, &OrderedMapEntry[K, V]{
			Key:   current.Key,
			Value: current.Value,
		})
		current = current.next
	}
	return result
}

// Find 查找满足条件的第一个键值对
func (om *OrderedMap[K, V]) Find(predicate func(K, V) bool) (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	var zeroK K
	var zeroV V
	
	current := om.head
	for current != nil {
		if predicate(current.Key, current.Value) {
			return current.Key, current.Value, true
		}
		current = current.next
	}
	return zeroK, zeroV, false
}

// Delete 删除指定键的键值对
func (om *OrderedMap[K, V]) Delete(key K) bool {
	om.mu.Lock()
	defer om.mu.Unlock()
	
	if entry, exists := om.items[key]; exists {
		delete(om.items, key)
		om.removeFromList(entry)
		return true
	}
	return false
}

// Keys 返回所有键的有序列表
func (om *OrderedMap[K, V]) Keys() []K {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	result := make([]K, 0, len(om.items))
	current := om.head
	for current != nil {
		result = append(result, current.Key)
		current = current.next
	}
	return result
}

// Values 返回所有值的有序列表
func (om *OrderedMap[K, V]) Values() []V {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	result := make([]V, 0, len(om.items))
	current := om.head
	for current != nil {
		result = append(result, current.Value)
		current = current.next
	}
	return result
}

// Clear 清空字典
func (om *OrderedMap[K, V]) Clear() {
	om.mu.Lock()
	defer om.mu.Unlock()
	
	om.items = make(map[K]*OrderedMapEntry[K, V])
	om.head = nil
	om.tail = nil
}

// Contains 检查是否包含指定的键
func (om *OrderedMap[K, V]) Contains(key K) bool {
	_, exists := om.Get(key)
	return exists
}

// IsEmpty 检查字典是否为空
func (om *OrderedMap[K, V]) IsEmpty() bool {
	return om.Len() == 0
}

// Front 返回第一个键值对
func (om *OrderedMap[K, V]) Front() (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	var zeroK K
	var zeroV V
	
	if om.head != nil {
		return om.head.Key, om.head.Value, true
	}
	return zeroK, zeroV, false
}

// Back 返回最后一个键值对
func (om *OrderedMap[K, V]) Back() (K, V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	var zeroK K
	var zeroV V
	
	if om.tail != nil {
		return om.tail.Key, om.tail.Value, true
	}
	return zeroK, zeroV, false
}

// ForEach 遍历所有键值对
func (om *OrderedMap[K, V]) ForEach(fn func(K, V)) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	
	current := om.head
	for current != nil {
		fn(current.Key, current.Value)
		current = current.next
	}
}

// appendToTail 将条目添加到链表尾部
func (om *OrderedMap[K, V]) appendToTail(entry *OrderedMapEntry[K, V]) {
	if om.tail == nil {
		// 空链表
		om.head = entry
		om.tail = entry
	} else {
		// 添加到尾部
		om.tail.next = entry
		entry.prev = om.tail
		om.tail = entry
	}
}

// removeFromList 从链表中移除条目
func (om *OrderedMap[K, V]) removeFromList(entry *OrderedMapEntry[K, V]) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		// 移除的是头节点
		om.head = entry.next
	}
	
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		// 移除的是尾节点
		om.tail = entry.prev
	}
	
	entry.prev = nil
	entry.next = nil
}