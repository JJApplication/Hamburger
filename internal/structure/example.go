/*
Project: Sandwich structure/example.go
Created: 2025/01/11 by Qoder
*/

package structure

import (
	"fmt"
)

// ExampleSetUsage 展示Set集合的使用方法
func ExampleSetUsage() {
	fmt.Println("=== Set usage example ===")

	// 创建字符串类型的Set
	fruits := NewSet[string]()

	// 添加元素
	fruits.Add("apple")
	fruits.Add("banana")
	fruits.Add("orange")
	fruits.Add("apple") // 重复添加，不会增加元素

	fmt.Printf("set size: %d\n", fruits.Len())
	fmt.Printf("contains apple: %v\n", fruits.Contains("apple"))
	fmt.Printf("contains grape: %v\n", fruits.Contains("grape"))

	// 列出所有元素
	fmt.Printf("all items: %v\n", fruits.List())

	// 查找满足条件的元素
	found, exists := fruits.Find(func(item string) bool {
		return len(item) > 5
	})
	if exists {
		fmt.Printf("found fruit with length > 5: %s\n", found)
	}

	// 创建另一个集合进行集合操作
	citrus := NewSetWithItems("orange", "lemon", "lime")

	// 并集
	union := fruits.Union(citrus)
	fmt.Printf("union: %v\n", union.List())

	// 交集
	intersection := fruits.Intersection(citrus)
	fmt.Printf("intersection: %v\n", intersection.List())

	// 差集
	difference := fruits.Difference(citrus)
	fmt.Printf("difference: %v\n", difference.List())

	fmt.Println()
}

// ExampleOrderedMapUsage 展示有序字典的使用方法
func ExampleOrderedMapUsage() {
	fmt.Println("=== OrderedMap usage example ===")

	// 创建字符串到整数的有序字典
	scores := NewOrderedMap[string, int]()

	// 添加键值对
	scores.Set("Alice", 95)
	scores.Set("Bob", 87)
	scores.Set("Charlie", 92)
	scores.Set("David", 88)

	fmt.Printf("map size: %d\n", scores.Len())

	// 获取值
	if score, exists := scores.Get("Alice"); exists {
		fmt.Printf("Alice's score: %d\n", score)
	}

	// 更新值
	scores.Set("Bob", 90)
	fmt.Printf("Bob's score after update: %d\n", func() int { v, _ := scores.Get("Bob"); return v }())

	// 按插入顺序列出所有键
	fmt.Printf("keys in insertion order: %v\n", scores.Keys())

	// 按插入顺序列出所有值
	fmt.Printf("values in insertion order: %v\n", scores.Values())

	// 列出所有键值对
	fmt.Println("all entries:")
	entries := scores.List()
	for _, entry := range entries {
		fmt.Printf("  %s: %d\n", entry.Key, entry.Value)
	}

	// 查找满足条件的键值对
	name, score, found := scores.Find(func(name string, score int) bool {
		return score > 90
	})
	if found {
		fmt.Printf("found student with score > 90: %s (%d)\n", name, score)
	}

	// 获取第一个和最后一个元素
	if firstName, firstScore, exists := scores.Front(); exists {
		fmt.Printf("first element: %s=%d\n", firstName, firstScore)
	}

	if lastName, lastScore, exists := scores.Back(); exists {
		fmt.Printf("last element: %s=%d\n", lastName, lastScore)
	}

	// 遍历所有元素
	fmt.Println("iterate using ForEach:")
	scores.ForEach(func(name string, score int) {
		fmt.Printf("  %s: %d\n", name, score)
	})

	// 删除元素
	if scores.Delete("Charlie") {
		fmt.Println("successfully deleted Charlie")
	}

	fmt.Printf("keys after deletion: %v\n", scores.Keys())

	fmt.Println()
}

// ExampleAdvancedUsage 展示高级使用场景
func ExampleAdvancedUsage() {
	fmt.Println("=== Advanced usage scenarios ===")

	// 场景1: 使用Set去重用户ID
	userIDs := []int{1, 2, 3, 2, 4, 1, 5, 3}
	uniqueIDs := NewSet[int]()

	for _, id := range userIDs {
		uniqueIDs.Add(id)
	}

	fmt.Printf("Original user IDs: %v\n", userIDs)
	fmt.Printf("User IDs after deduplication: %v\n", uniqueIDs.List())

	// 场景2: 使用有序字典实现LRU缓存的基础结构
	cache := NewOrderedMap[string, string]()
	maxSize := 3

	// 模拟缓存访问
	cache.Set("page1", "content1")
	cache.Set("page2", "content2")
	cache.Set("page3", "content3")

	fmt.Printf("Cache content: %v\n", cache.Keys())

	// 当缓存满时，删除最旧的元素
	if cache.Len() >= maxSize {
		if oldestKey, _, exists := cache.Front(); exists {
			cache.Delete(oldestKey)
			fmt.Printf("Deleted oldest cache item: %s\n", oldestKey)
		}
	}

	cache.Set("page4", "content4")
	fmt.Printf("Cache after adding new page: %v\n", cache.Keys())

	// 场景3: 使用Set进行权限检查
	adminPermissions := NewSetWithItems("read", "write", "delete", "admin")
	userPermissions := NewSetWithItems("read", "write")

	// 检查用户是否有特定权限
	if userPermissions.Contains("delete") {
		fmt.Println("User has delete permission")
	} else {
		fmt.Println("User does not have delete permission")
	}

	// 获取用户缺少的权限
	missingPermissions := adminPermissions.Difference(userPermissions)
	fmt.Printf("User missing permissions: %v\n", missingPermissions.List())

	fmt.Println()
}
