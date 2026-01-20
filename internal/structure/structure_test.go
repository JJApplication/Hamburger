/*
Project: Sandwich structure/structure_test.go
Created: 2025/01/11 by Qoder
*/

package structure

import (
	"strconv"
	"testing"
)

// TestSet 测试Set集合功能
func TestSet(t *testing.T) {
	// 创建字符串类型的Set
	set := NewSet[string]()
	
	// 测试初始状态
	if set.Len() != 0 {
		t.Errorf("Expected length 0, got %d", set.Len())
	}
	
	if !set.IsEmpty() {
		t.Error("Expected set to be empty")
	}
	
	// 测试添加元素
	set.Set("apple")
	set.Set("banana")
	set.Set("orange")
	
	if set.Len() != 3 {
		t.Errorf("Expected length 3, got %d", set.Len())
	}
	
	// 测试重复添加
	set.Set("apple")
	if set.Len() != 3 {
		t.Errorf("Expected length 3 after duplicate add, got %d", set.Len())
	}
	
	// 测试Get方法
	if !set.Get("apple") {
		t.Error("Expected 'apple' to exist in set")
	}
	
	if set.Get("grape") {
		t.Error("Expected 'grape' not to exist in set")
	}
	
	// 测试List方法
	items := set.List()
	if len(items) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(items))
	}
	
	// 测试Find方法
	found, exists := set.Find(func(item string) bool {
		return item == "banana"
	})
	if !exists || found != "banana" {
		t.Error("Expected to find 'banana'")
	}
	
	// 测试不存在的查找
	_, exists = set.Find(func(item string) bool {
		return item == "grape"
	})
	if exists {
		t.Error("Expected not to find 'grape'")
	}
	
	// 测试Remove方法
	if !set.Remove("apple") {
		t.Error("Expected to successfully remove 'apple'")
	}
	
	if set.Len() != 2 {
		t.Errorf("Expected length 2 after removal, got %d", set.Len())
	}
	
	// 测试集合操作
	set2 := NewSetWithItems("banana", "grape", "kiwi")
	
	// 并集
	union := set.Union(set2)
	if union.Len() != 4 { // banana, orange, grape, kiwi
		t.Errorf("Expected union length 4, got %d", union.Len())
	}
	
	// 交集
	intersection := set.Intersection(set2)
	if intersection.Len() != 1 { // banana
		t.Errorf("Expected intersection length 1, got %d", intersection.Len())
	}
	
	// 差集
	difference := set.Difference(set2)
	if difference.Len() != 1 { // orange
		t.Errorf("Expected difference length 1, got %d", difference.Len())
	}
}

// TestOrderedMap 测试有序字典功能
func TestOrderedMap(t *testing.T) {
	// 创建字符串到整数的有序字典
	om := NewOrderedMap[string, int]()
	
	// 测试初始状态
	if om.Len() != 0 {
		t.Errorf("Expected length 0, got %d", om.Len())
	}
	
	if !om.IsEmpty() {
		t.Error("Expected ordered map to be empty")
	}
	
	// 测试添加元素
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)
	
	if om.Len() != 3 {
		t.Errorf("Expected length 3, got %d", om.Len())
	}
	
	// 测试Get方法
	value, exists := om.Get("second")
	if !exists || value != 2 {
		t.Errorf("Expected to get value 2 for 'second', got %d, exists: %v", value, exists)
	}
	
	// 测试不存在的键
	_, exists = om.Get("fourth")
	if exists {
		t.Error("Expected 'fourth' not to exist")
	}
	
	// 测试更新现有键
	om.Set("second", 22)
	value, exists = om.Get("second")
	if !exists || value != 22 {
		t.Errorf("Expected updated value 22 for 'second', got %d", value)
	}
	
	// 长度应该保持不变
	if om.Len() != 3 {
		t.Errorf("Expected length to remain 3 after update, got %d", om.Len())
	}
	
	// 测试List方法（检查顺序）
	entries := om.List()
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries in list, got %d", len(entries))
	}
	
	expectedOrder := []string{"first", "second", "third"}
	for i, entry := range entries {
		if entry.Key != expectedOrder[i] {
			t.Errorf("Expected key %s at position %d, got %s", expectedOrder[i], i, entry.Key)
		}
	}
	
	// 测试Keys和Values方法
	keys := om.Keys()
	values := om.Values()
	
	if len(keys) != 3 || len(values) != 3 {
		t.Errorf("Expected 3 keys and values, got %d keys, %d values", len(keys), len(values))
	}
	
	for i, key := range keys {
		if key != expectedOrder[i] {
			t.Errorf("Expected key %s at position %d, got %s", expectedOrder[i], i, key)
		}
	}
	
	// 测试Find方法
	key, value, found := om.Find(func(k string, v int) bool {
		return v > 20
	})
	if !found || key != "second" || value != 22 {
		t.Errorf("Expected to find 'second' with value 22, got %s=%d, found=%v", key, value, found)
	}
	
	// 测试Delete方法
	if !om.Delete("second") {
		t.Error("Expected to successfully delete 'second'")
	}
	
	if om.Len() != 2 {
		t.Errorf("Expected length 2 after deletion, got %d", om.Len())
	}
	
	// 验证删除后的顺序
	keys = om.Keys()
	expectedAfterDelete := []string{"first", "third"}
	for i, key := range keys {
		if key != expectedAfterDelete[i] {
			t.Errorf("Expected key %s at position %d after deletion, got %s", expectedAfterDelete[i], i, key)
		}
	}
	
	// 测试Front和Back方法
	frontKey, frontValue, exists := om.Front()
	if !exists || frontKey != "first" || frontValue != 1 {
		t.Errorf("Expected front to be 'first'=1, got %s=%d, exists=%v", frontKey, frontValue, exists)
	}
	
	backKey, backValue, exists := om.Back()
	if !exists || backKey != "third" || backValue != 3 {
		t.Errorf("Expected back to be 'third'=3, got %s=%d, exists=%v", backKey, backValue, exists)
	}
	
	// 测试ForEach方法
	var count int
	om.ForEach(func(k string, v int) {
		count++
	})
	if count != 2 {
		t.Errorf("Expected ForEach to iterate over 2 items, got %d", count)
	}
}

// BenchmarkSet 性能测试Set
func BenchmarkSet(b *testing.B) {
	set := NewSet[int]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Set(i)
	}
}

// BenchmarkOrderedMap 性能测试OrderedMap
func BenchmarkOrderedMap(b *testing.B) {
	om := NewOrderedMap[string, int]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		om.Set(strconv.Itoa(i), i)
	}
}