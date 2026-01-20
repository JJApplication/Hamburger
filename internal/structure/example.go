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
	fmt.Println("=== Set集合使用示例 ===")
	
	// 创建字符串类型的Set
	fruits := NewSet[string]()
	
	// 添加元素
	fruits.Add("apple")
	fruits.Add("banana")
	fruits.Add("orange")
	fruits.Add("apple") // 重复添加，不会增加元素
	
	fmt.Printf("集合大小: %d\n", fruits.Len())
	fmt.Printf("包含apple: %v\n", fruits.Contains("apple"))
	fmt.Printf("包含grape: %v\n", fruits.Contains("grape"))
	
	// 列出所有元素
	fmt.Printf("所有元素: %v\n", fruits.List())
	
	// 查找满足条件的元素
	found, exists := fruits.Find(func(item string) bool {
		return len(item) > 5
	})
	if exists {
		fmt.Printf("找到长度大于5的水果: %s\n", found)
	}
	
	// 创建另一个集合进行集合操作
	citrus := NewSetWithItems("orange", "lemon", "lime")
	
	// 并集
	union := fruits.Union(citrus)
	fmt.Printf("并集: %v\n", union.List())
	
	// 交集
	intersection := fruits.Intersection(citrus)
	fmt.Printf("交集: %v\n", intersection.List())
	
	// 差集
	difference := fruits.Difference(citrus)
	fmt.Printf("差集: %v\n", difference.List())
	
	fmt.Println()
}

// ExampleOrderedMapUsage 展示有序字典的使用方法
func ExampleOrderedMapUsage() {
	fmt.Println("=== 有序字典使用示例 ===")
	
	// 创建字符串到整数的有序字典
	scores := NewOrderedMap[string, int]()
	
	// 添加键值对
	scores.Set("Alice", 95)
	scores.Set("Bob", 87)
	scores.Set("Charlie", 92)
	scores.Set("David", 88)
	
	fmt.Printf("字典大小: %d\n", scores.Len())
	
	// 获取值
	if score, exists := scores.Get("Alice"); exists {
		fmt.Printf("Alice的分数: %d\n", score)
	}
	
	// 更新值
	scores.Set("Bob", 90)
	fmt.Printf("更新后Bob的分数: %d\n", func() int { v, _ := scores.Get("Bob"); return v }())
	
	// 按插入顺序列出所有键
	fmt.Printf("按插入顺序的键: %v\n", scores.Keys())
	
	// 按插入顺序列出所有值
	fmt.Printf("按插入顺序的值: %v\n", scores.Values())
	
	// 列出所有键值对
	fmt.Println("所有键值对:")
	entries := scores.List()
	for _, entry := range entries {
		fmt.Printf("  %s: %d\n", entry.Key, entry.Value)
	}
	
	// 查找满足条件的键值对
	name, score, found := scores.Find(func(name string, score int) bool {
		return score > 90
	})
	if found {
		fmt.Printf("找到分数大于90的学生: %s (%d分)\n", name, score)
	}
	
	// 获取第一个和最后一个元素
	if firstName, firstScore, exists := scores.Front(); exists {
		fmt.Printf("第一个元素: %s=%d\n", firstName, firstScore)
	}
	
	if lastName, lastScore, exists := scores.Back(); exists {
		fmt.Printf("最后一个元素: %s=%d\n", lastName, lastScore)
	}
	
	// 遍历所有元素
	fmt.Println("使用ForEach遍历:")
	scores.ForEach(func(name string, score int) {
		fmt.Printf("  %s: %d分\n", name, score)
	})
	
	// 删除元素
	if scores.Delete("Charlie") {
		fmt.Println("成功删除Charlie")
	}
	
	fmt.Printf("删除后的键: %v\n", scores.Keys())
	
	fmt.Println()
}

// ExampleAdvancedUsage 展示高级使用场景
func ExampleAdvancedUsage() {
	fmt.Println("=== 高级使用场景 ===")
	
	// 场景1: 使用Set去重用户ID
	userIDs := []int{1, 2, 3, 2, 4, 1, 5, 3}
	uniqueIDs := NewSet[int]()
	
	for _, id := range userIDs {
		uniqueIDs.Add(id)
	}
	
	fmt.Printf("原始用户ID: %v\n", userIDs)
	fmt.Printf("去重后的用户ID: %v\n", uniqueIDs.List())
	
	// 场景2: 使用有序字典实现LRU缓存的基础结构
	cache := NewOrderedMap[string, string]()
	maxSize := 3
	
	// 模拟缓存访问
	cache.Set("page1", "content1")
	cache.Set("page2", "content2")
	cache.Set("page3", "content3")
	
	fmt.Printf("缓存内容: %v\n", cache.Keys())
	
	// 当缓存满时，删除最旧的元素
	if cache.Len() >= maxSize {
		if oldestKey, _, exists := cache.Front(); exists {
			cache.Delete(oldestKey)
			fmt.Printf("删除最旧的缓存项: %s\n", oldestKey)
		}
	}
	
	cache.Set("page4", "content4")
	fmt.Printf("添加新页面后的缓存: %v\n", cache.Keys())
	
	// 场景3: 使用Set进行权限检查
	adminPermissions := NewSetWithItems("read", "write", "delete", "admin")
	userPermissions := NewSetWithItems("read", "write")
	
	// 检查用户是否有特定权限
	if userPermissions.Contains("delete") {
		fmt.Println("用户有删除权限")
	} else {
		fmt.Println("用户没有删除权限")
	}
	
	// 获取用户缺少的权限
	missingPermissions := adminPermissions.Difference(userPermissions)
	fmt.Printf("用户缺少的权限: %v\n", missingPermissions.List())
	
	fmt.Println()
}