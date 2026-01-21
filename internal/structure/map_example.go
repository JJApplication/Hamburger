package structure

import "fmt"

// User 示例结构体
type User struct {
	ID   int
	Name string
	Age  int
}

// ExampleUsage 演示泛型Map的使用方法
func ExampleUsage() {
	// 示例1: 存储int类型
	intMap := NewMap[int]()
	intMap.Put("count", 100)
	intMap.Put("total", 500)

	if value, ok := intMap.Get("count"); ok {
		fmt.Printf("count: %d\n", value)
	}

	fmt.Printf("total exists: %t\n", intMap.Exist("total"))
	fmt.Printf("size: %d\n", intMap.Size())

	// 示例2: 存储结构体
	userMap := NewMap[User]()
	userMap.Put("user1", User{ID: 1, Name: "Alice", Age: 25})
	userMap.Put("user2", User{ID: 2, Name: "Bob", Age: 30})

	if user, ok := userMap.Get("user1"); ok {
		fmt.Printf("User: %+v\n", user)
	}

	// 示例3: 存储结构体指针
	userPtrMap := NewMap[*User]()
	userPtrMap.Put("ptr1", &User{ID: 3, Name: "Charlie", Age: 35})
	userPtrMap.Put("ptr2", &User{ID: 4, Name: "Diana", Age: 28})

	if userPtr, ok := userPtrMap.Get("ptr1"); ok {
		fmt.Printf("User Pointer: %+v\n", *userPtr)
	}

	// 示例4: 存储字符串切片
	sliceMap := NewMap[[]string]()
	sliceMap.Put("tags", []string{"go", "generics", "concurrent"})

	if tags, ok := sliceMap.Get("tags"); ok {
		fmt.Printf("Tags: %v\n", tags)
	}

	// 示例5: 使用MustGet方法
	fmt.Println("\n=== MustGet method demo ===")

	// MustGet存在的键
	count := intMap.MustGet("count")
	fmt.Printf("count (exists): %d\n", count)

	// MustGet不存在的键，返回零值
	missing := intMap.MustGet("missing")
	fmt.Printf("missing (not exists): %d (zero value)\n", missing)

	// 不同类型的零值示例
	stringMap := NewMap[string]()
	stringMap.Put("hello", "world")

	existing := stringMap.MustGet("hello")
	fmt.Printf("existing string: '%s'\n", existing)

	emptyString := stringMap.MustGet("nonexistent")
	fmt.Printf("nonexistent string: '%s' (zero value)\n", emptyString)

	// 结构体零值示例
	missingUser := userMap.MustGet("nonexistent")
	fmt.Printf("nonexistent user: %+v (zero value)\n", missingUser)

	// 指针零值示例
	missingPtr := userPtrMap.MustGet("nonexistent")
	fmt.Printf("nonexistent pointer: %v (zero value)\n", missingPtr)

	// 获取所有键
	keys := intMap.Keys()
	fmt.Printf("Int map keys: %v\n", keys)

	// 获取所有值
	values := intMap.Values()
	fmt.Printf("Int map values: %v\n", values)

	// 删除元素
	intMap.Delete("count")
	fmt.Printf("After delete, size: %d\n", intMap.Size())

	// 清空Map
	intMap.Clear()
	fmt.Printf("After clear, size: %d\n", intMap.Size())

	// 示例5: 使用Range方法遍历
	fmt.Println("\n=== Range method demo ===")

	// 重新添加一些数据用于演示
	scoreMap := NewMap[int]()
	scoreMap.Put("math", 95)
	scoreMap.Put("english", 88)
	scoreMap.Put("physics", 92)
	scoreMap.Put("chemistry", 90)

	// 完整遍历所有元素
	fmt.Println("All subject scores:")
	scoreMap.Range(func(subject string, score int) bool {
		fmt.Printf("  %s: %d\n", subject, score)
		return true // 继续遍历
	})

	// 计算总分和平均分
	totalScore := 0
	subjectCount := 0
	scoreMap.Range(func(subject string, score int) bool {
		totalScore += score
		subjectCount++
		return true
	})

	if subjectCount > 0 {
		average := float64(totalScore) / float64(subjectCount)
		fmt.Printf("Total: %d, Average: %.2f\n", totalScore, average)
	}

	// 查找第一个90分以上的科目
	fmt.Println("First subject with score >= 90:")
	scoreMap.Range(func(subject string, score int) bool {
		if score >= 90 {
			fmt.Printf("  Found: %s (%d)\n", subject, score)
			return false // 找到后停止遍历
		}
		return true // 继续查找
	})

	// 使用Range方法过滤数据
	fmt.Println("Subjects with score >= 90:")
	scoreMap.Range(func(subject string, score int) bool {
		if score >= 90 {
			fmt.Printf("  %s: %d\n", subject, score)
		}
		return true // 继续遍历所有元素
	})

	fmt.Println("\n=== Find method demo ===")

	// 使用Find查找第一个90分以上的学生
	fmt.Println("Find first student with score >= 90:")
	name, score, found := scoreMap.Find(func(n string, s int) bool {
		return s >= 90
	})
	if found {
		fmt.Printf("  Found: %s (%d)\n", name, score)
	} else {
		fmt.Println("  No student found with score >= 90")
	}

	// 查找特定学生
	fmt.Println("\nFind score for student 'Alice':")
	name, score, found = scoreMap.Find(func(n string, s int) bool {
		return n == "Alice"
	})
	if found {
		fmt.Printf("  %s: %d\n", name, score)
	} else {
		fmt.Println("  Student Alice not found")
	}

	// 查找最高分学生
	fmt.Println("\nFind student with highest score:")
	maxScore := 0
	scoreMap.Range(func(n string, s int) bool {
		if s > maxScore {
			maxScore = s
		}
		return true
	})

	name, score, found = scoreMap.Find(func(n string, s int) bool {
		return s == maxScore
	})
	if found {
		fmt.Printf("  Highest score: %s (%d)\n", name, score)
	}

	fmt.Println("\n=== FindAll method demo ===")

	// 查找所有90分以上的学生
	fmt.Println("Find all students with score >= 90:")
	highScoreStudents := scoreMap.FindAll(func(n string, s int) bool {
		return s >= 90
	})

	if len(highScoreStudents) > 0 {
		fmt.Printf("  Found %d high score students:\n", len(highScoreStudents))
		for _, kv := range highScoreStudents {
			fmt.Printf("    %s: %d\n", kv.Key, kv.Value)
		}
	} else {
		fmt.Println("  No student found with score >= 90")
	}

	// 查找所有80-89分的学生
	fmt.Println("\nFind all students with score 80-89:")
	goodStudents := scoreMap.FindAll(func(n string, s int) bool {
		return s >= 80 && s < 90
	})

	if len(goodStudents) > 0 {
		fmt.Printf("  Found %d good score students:\n", len(goodStudents))
		for _, kv := range goodStudents {
			fmt.Printf("    %s: %d\n", kv.Key, kv.Value)
		}
	} else {
		fmt.Println("  No student found with score 80-89")
	}

	// 统计不同分数段的学生数量
	fmt.Println("\nCount students in different score ranges:")
	excellent := scoreMap.FindAll(func(n string, s int) bool { return s >= 90 })
	good := scoreMap.FindAll(func(n string, s int) bool { return s >= 80 && s < 90 })
	average := scoreMap.FindAll(func(n string, s int) bool { return s >= 70 && s < 80 })
	poor := scoreMap.FindAll(func(n string, s int) bool { return s < 70 })

	fmt.Printf("  Excellent (90+): %d\n", len(excellent))
	fmt.Printf("  Good (80-89): %d\n", len(good))
	fmt.Printf("  Average (70-79): %d\n", len(average))
	fmt.Printf("  Need improvement (<70): %d\n", len(poor))
}
