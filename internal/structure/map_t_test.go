package structure

import (
	"fmt"
	"sync"
	"testing"
)

// TestMapBasicOperations 测试基本操作
func TestMapBasicOperations(t *testing.T) {
	// 测试int类型
	intMap := NewMap[int]()
	
	// 测试Put和Get
	intMap.Put("key1", 100)
	intMap.Put("key2", 200)
	
	if value, ok := intMap.Get("key1"); !ok || value != 100 {
		t.Errorf("Expected 100, got %d", value)
	}
	
	// 测试Exist
	if !intMap.Exist("key1") {
		t.Error("key1 should exist")
	}
	
	if intMap.Exist("nonexistent") {
		t.Error("nonexistent key should not exist")
	}
	
	// 测试Size
	if size := intMap.Size(); size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}
	
	// 测试Delete
	intMap.Delete("key1")
	if intMap.Exist("key1") {
		t.Error("key1 should be deleted")
	}
	
	if size := intMap.Size(); size != 1 {
		t.Errorf("Expected size 1 after delete, got %d", size)
	}
	
	// 测试Clear
	intMap.Clear()
	if size := intMap.Size(); size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
	
	// 测试MustGet - 存在的键
	intMap.Put("test", 42)
	if value := intMap.MustGet("test"); value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	
	// 测试MustGet - 不存在的键应该返回零值
	if value := intMap.MustGet("nonexistent"); value != 0 {
		t.Errorf("Expected zero value 0 for nonexistent key, got %d", value)
	}
}

// TestMapMustGetZeroValues 测试MustGet方法返回不同类型的零值
func TestMapMustGetZeroValues(t *testing.T) {
	// 测试int类型零值
	intMap := NewMap[int]()
	if value := intMap.MustGet("nonexistent"); value != 0 {
		t.Errorf("Expected int zero value 0, got %d", value)
	}
	
	// 测试string类型零值
	stringMap := NewMap[string]()
	if value := stringMap.MustGet("nonexistent"); value != "" {
		t.Errorf("Expected string zero value '', got '%s'", value)
	}
	
	// 测试bool类型零值
	boolMap := NewMap[bool]()
	if value := boolMap.MustGet("nonexistent"); value != false {
		t.Errorf("Expected bool zero value false, got %t", value)
	}
	
	// 测试指针类型零值
	ptrMap := NewMap[*int]()
	if value := ptrMap.MustGet("nonexistent"); value != nil {
		t.Errorf("Expected pointer zero value nil, got %v", value)
	}
	
	// 测试slice类型零值
	sliceMap := NewMap[[]int]()
	if value := sliceMap.MustGet("nonexistent"); value != nil {
		t.Errorf("Expected slice zero value nil, got %v", value)
	}
	
	// 测试结构体类型零值
	type TestStruct struct {
		Name string
		Age  int
	}
	structMap := NewMap[TestStruct]()
	expectedZero := TestStruct{}
	if value := structMap.MustGet("nonexistent"); value != expectedZero {
		t.Errorf("Expected struct zero value %+v, got %+v", expectedZero, value)
	}
}

// TestMapWithStruct 测试结构体类型
func TestMapWithStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	
	personMap := NewMap[Person]()
	
	person1 := Person{Name: "Alice", Age: 25}
	person2 := Person{Name: "Bob", Age: 30}
	
	personMap.Put("alice", person1)
	personMap.Put("bob", person2)
	
	if retrieved, ok := personMap.Get("alice"); !ok || retrieved.Name != "Alice" || retrieved.Age != 25 {
		t.Errorf("Expected Alice, 25, got %s, %d", retrieved.Name, retrieved.Age)
	}
	
	// 测试Keys和Values
	keys := personMap.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
	
	values := personMap.Values()
	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(values))
	}
}

// TestMapWithPointer 测试指针类型
func TestMapWithPointer(t *testing.T) {
	type Data struct {
		Value int
	}
	
	ptrMap := NewMap[*Data]()
	
	data1 := &Data{Value: 100}
	data2 := &Data{Value: 200}
	
	ptrMap.Put("data1", data1)
	ptrMap.Put("data2", data2)
	
	if retrieved, ok := ptrMap.Get("data1"); !ok || retrieved.Value != 100 {
		t.Errorf("Expected 100, got %d", retrieved.Value)
	}
	
	// 修改原始数据，验证指针引用
	data1.Value = 150
	if retrieved, ok := ptrMap.Get("data1"); !ok || retrieved.Value != 150 {
		t.Errorf("Expected 150 after modification, got %d", retrieved.Value)
	}
}

// TestMapConcurrency 测试并发安全性
func TestMapConcurrency(t *testing.T) {
	intMap := NewMap[int]()
	
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000
	
	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				intMap.Put(key, id*numOperations+j)
			}
		}(i)
	}
	
	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				intMap.Get(key)
				intMap.Exist(key)
			}
		}(i)
	}
	
	wg.Wait()
	
	// 验证最终大小
	expectedSize := numGoroutines * numOperations
	if size := intMap.Size(); size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

// BenchmarkMapPut 性能测试 - Put操作
func BenchmarkMapPut(b *testing.B) {
	intMap := NewMap[int]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		intMap.Put(key, i)
	}
}

// TestMapRange 测试Range方法
func TestMapRange(t *testing.T) {
	intMap := NewMap[int]()
	
	// 添加测试数据
	testData := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
		"four":  4,
		"five":  5,
	}
	
	for k, v := range testData {
		intMap.Put(k, v)
	}
	
	// 测试完整遍历
	visited := make(map[string]int)
	intMap.Range(func(key string, value int) bool {
		visited[key] = value
		return true // 继续遍历
	})
	
	// 验证所有元素都被访问
	if len(visited) != len(testData) {
		t.Errorf("Expected %d items, got %d", len(testData), len(visited))
	}
	
	for k, v := range testData {
		if visitedValue, ok := visited[k]; !ok || visitedValue != v {
			t.Errorf("Key %s: expected %d, got %d", k, v, visitedValue)
		}
	}
	
	// 测试提前停止遍历
	count := 0
	intMap.Range(func(key string, value int) bool {
		count++
		return count < 3 // 只遍历前3个元素
	})
	
	if count != 3 {
		t.Errorf("Expected to visit 3 items, got %d", count)
	}
}

// TestMapRangeWithStruct 测试Range方法处理结构体
func TestMapRangeWithStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	
	personMap := NewMap[Person]()
	
	// 添加测试数据
	personMap.Put("alice", Person{Name: "Alice", Age: 25})
	personMap.Put("bob", Person{Name: "Bob", Age: 30})
	personMap.Put("charlie", Person{Name: "Charlie", Age: 35})
	
	// 遍历并验证
	totalAge := 0
	nameCount := 0
	
	personMap.Range(func(key string, person Person) bool {
		totalAge += person.Age
		nameCount++
		
		// 验证key和person.Name的一致性（假设key是小写的name）
		if key != "alice" && key != "bob" && key != "charlie" {
			t.Errorf("Unexpected key: %s", key)
		}
		
		return true
	})
	
	expectedTotalAge := 25 + 30 + 35
	if totalAge != expectedTotalAge {
		t.Errorf("Expected total age %d, got %d", expectedTotalAge, totalAge)
	}
	
	if nameCount != 3 {
		t.Errorf("Expected 3 people, got %d", nameCount)
	}
}

// TestMapRangeEmpty 测试空Map的Range操作
func TestMapRangeEmpty(t *testing.T) {
	intMap := NewMap[int]()
	
	called := false
	intMap.Range(func(key string, value int) bool {
		called = true
		return true
	})
	
	if called {
		t.Error("Range function should not be called on empty map")
	}
}

// BenchmarkMapGet 性能测试 - Get操作
func BenchmarkMapGet(b *testing.B) {
	intMap := NewMap[int]()
	
	// 预填充数据
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		intMap.Put(key, i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%10000)
		intMap.Get(key)
	}
}

// BenchmarkMapRange 性能测试 - Range操作
func BenchmarkMapRange(b *testing.B) {
	intMap := NewMap[int]()
	
	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		intMap.Put(key, i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		intMap.Range(func(key string, value int) bool {
			return true // 遍历所有元素
		})
	}
}

// TestMapFind 测试Find方法
func TestMapFind(t *testing.T) {
	m := NewMap[int]()
	m.Put("apple", 5)
	m.Put("banana", 3)
	m.Put("cherry", 8)
	m.Put("date", 2)
	
	// 测试查找存在的元素
	key, value, found := m.Find(func(k string, v int) bool {
		return v > 5
	})
	if !found {
		t.Error("should find element with value greater than 5")
	}
	if value <= 5 {
		t.Errorf("found value should be greater than 5, actual value: %d", value)
	}
	t.Logf("found: %s = %d", key, value)
	
	// 测试查找不存在的元素
	_, _, found = m.Find(func(k string, v int) bool {
		return v > 10
	})
	if found {
		t.Error("should not find element with value greater than 10")
	}
	
	// 测试根据key查找
	key, value, found = m.Find(func(k string, v int) bool {
		return k == "banana"
	})
	if !found {
		t.Error("should find element with key banana")
	}
	if key != "banana" || value != 3 {
		t.Errorf("expected to find banana=3, actually found %s=%d", key, value)
	}
}

// TestMapFindAll 测试FindAll方法
func TestMapFindAll(t *testing.T) {
	m := NewMap[int]()
	m.Put("apple", 5)
	m.Put("banana", 3)
	m.Put("cherry", 8)
	m.Put("date", 2)
	m.Put("elderberry", 6)
	
	// 测试查找所有值大于等于5的元素
	results := m.FindAll(func(k string, v int) bool {
		return v >= 5
	})
	
	if len(results) != 3 {
		t.Errorf("expected to find 3 elements, actually found %d", len(results))
	}
	
	// 验证结果
	expectedKeys := map[string]int{"apple": 5, "cherry": 8, "elderberry": 6}
	for _, kv := range results {
		expectedValue, exists := expectedKeys[kv.Key]
		if !exists {
			t.Errorf("unexpected key: %s", kv.Key)
		}
		if kv.Value != expectedValue {
			t.Errorf("value for key %s does not match, expected %d, actual %d", kv.Key, expectedValue, kv.Value)
		}
	}
	
	// 测试查找不存在的元素
	results = m.FindAll(func(k string, v int) bool {
		return v > 10
	})
	if len(results) != 0 {
		t.Errorf("expected to find 0 elements, actually found %d", len(results))
	}
	
	// 测试查找所有元素
	results = m.FindAll(func(k string, v int) bool {
		return true
	})
	if len(results) != 5 {
		t.Errorf("expected to find 5 elements, actually found %d", len(results))
	}
}

// TestMapFindWithStruct 测试Find方法处理结构体
func TestMapFindWithStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	
	m := NewMap[Person]()
	m.Put("p1", Person{Name: "Alice", Age: 25})
	m.Put("p2", Person{Name: "Bob", Age: 30})
	m.Put("p3", Person{Name: "Charlie", Age: 35})
	
	// 查找年龄大于等于30的第一个人
	key, person, found := m.Find(func(k string, p Person) bool {
		return p.Age >= 30
	})
	if !found {
		t.Error("should find person with age greater than or equal to 30")
	}
	if person.Age < 30 {
		t.Errorf("found person's age should be greater than or equal to 30, actual age: %d", person.Age)
	}
	t.Logf("found: %s = %+v", key, person)
	
	// 查找所有年龄大于等于30的人
	results := m.FindAll(func(k string, p Person) bool {
		return p.Age >= 30
	})
	if len(results) != 2 {
		t.Errorf("expected to find 2 people, actually found %d", len(results))
	}
}

// TestMapFindEmpty 测试空Map的Find操作
func TestMapFindEmpty(t *testing.T) {
	m := NewMap[int]()
	
	// 测试空Map的Find
	_, _, found := m.Find(func(k string, v int) bool {
		return true
	})
	if found {
		t.Error("empty map should not find any elements")
	}
	
	// 测试空Map的FindAll
	results := m.FindAll(func(k string, v int) bool {
		return true
	})
	if len(results) != 0 {
		t.Errorf("empty map should return empty results, actually returned %d elements", len(results))
	}
}

// BenchmarkMapFind 性能测试Find方法
func BenchmarkMapFind(b *testing.B) {
	m := NewMap[int]()
	
	// 准备测试数据
	for i := 0; i < 1000; i++ {
		m.Put(fmt.Sprintf("key%d", i), i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Find(func(key string, value int) bool {
			return value == 500
		})
	}
}

// BenchmarkMapFindAll 性能测试FindAll方法
func BenchmarkMapFindAll(b *testing.B) {
	m := NewMap[int]()
	
	// 准备测试数据
	for i := 0; i < 1000; i++ {
		m.Put(fmt.Sprintf("key%d", i), i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.FindAll(func(key string, value int) bool {
			return value%10 == 0
		})
	}
}