# Sandwich Structure Package

这个包提供了多个高级数据结构，用于快速高效地处理数据：

## 1. Set 集合

Set 是一个泛型集合，支持任意可比较类型，集合中的元素不会重复。

### 特性
- **泛型支持**: 支持任意可比较类型 (`comparable`)
- **并发安全**: 使用读写锁保证线程安全
- **高效操作**: 基于哈希表实现，查找和插入都是 O(1) 时间复杂度
- **集合运算**: 支持并集、交集、差集操作

### 方法

| 方法 | 描述 | 时间复杂度 |
|------|------|------------|
| `Len()` | 返回集合中元素的数量 | O(1) |
| `Get(item)` | 检查元素是否存在于集合中 | O(1) |
| `Set(item)` | 向集合中添加元素 | O(1) |
| `List()` | 返回集合中所有元素的切片 | O(n) |
| `Find(predicate)` | 查找满足条件的元素 | O(n) |
| `Remove(item)` | 从集合中移除元素 | O(1) |
| `Clear()` | 清空集合 | O(1) |
| `Union(other)` | 返回两个集合的并集 | O(n+m) |
| `Intersection(other)` | 返回两个集合的交集 | O(min(n,m)) |
| `Difference(other)` | 返回两个集合的差集 | O(n) |

### 使用示例

```go
// 创建字符串类型的Set
fruits := NewSet[string]()

// 添加元素
fruits.Add("apple")
fruits.Add("banana")
fruits.Add("orange")

// 检查元素
fmt.Printf("包含apple: %v\n", fruits.Contains("apple")) // true

// 集合运算
citrus := NewSetWithItems("orange", "lemon", "lime")
union := fruits.Union(citrus)
fmt.Printf("并集: %v\n", union.List())
```

## 2. Map 泛型并发安全字典

Map 是一个使用Go泛型实现的并发安全字典，支持string键和任意类型的值。

### 特性
- **泛型支持**: 支持任意类型的值 (`T any`)
- **并发安全**: 使用读写锁保证线程安全
- **类型安全**: 编译时类型检查，避免运行时类型错误
- **高性能**: 基于Go原生map实现，性能优异
- **完整API**: 提供常用的Map操作方法

### 方法

| 方法 | 描述 | 时间复杂度 |
|------|------|------------|
| `Put(key, value)` | 存储键值对 | O(1) |
| `Get(key)` | 获取值，返回值和存在标志 | O(1) |
| `MustGet(key)` | 获取值，不存在时返回零值 | O(1) |
| `Exist(key)` | 检查键是否存在 | O(1) |
| `Delete(key)` | 删除指定键 | O(1) |
| `Size()` | 返回元素数量 | O(1) |
| `Clear()` | 清空所有元素 | O(1) |
| `Keys()` | 返回所有键的切片 | O(n) |
| `Values()` | 返回所有值的切片 | O(n) |
| `Range(fn)` | 遍历所有键值对 | O(n) |
| `Find(fn)` | 查找第一个满足条件的键值对 | O(n) |
| `FindAll(fn)` | 查找所有满足条件的键值对 | O(n) |

### 使用示例

```go
// 创建存储int类型的Map
intMap := NewMap[int]()
intMap.Put("count", 100)
intMap.Put("total", 500)

// 获取数据
if value, ok := intMap.Get("count"); ok {
    fmt.Printf("count: %d\n", value)
}

// 遍历所有元素
intMap.Range(func(key string, value int) bool {
    fmt.Printf("%s: %d\n", key, value)
    return true // 继续遍历
})

// 存储结构体
type User struct {
    ID   int
    Name string
}

userMap := NewMap[User]()
userMap.Put("alice", User{ID: 1, Name: "Alice"})

// 存储指针
userPtrMap := NewMap[*User]()
userPtrMap.Put("bob", &User{ID: 2, Name: "Bob"})
```

#### MustGet方法
MustGet方法用于获取值，当键不存在时返回该类型的零值，而不是像Get方法那样返回布尔标志：

```go
// 创建Map并添加数据
intMap := NewMap[int]()
intMap.Put("count", 100)

// 获取存在的键
value := intMap.MustGet("count")  // 返回 100

// 获取不存在的键，返回零值
missing := intMap.MustGet("missing")  // 返回 0 (int的零值)

// 不同类型的零值示例
stringMap := NewMap[string]()
emptyStr := stringMap.MustGet("missing")  // 返回 "" (string的零值)

boolMap := NewMap[bool]()
falseBool := boolMap.MustGet("missing")  // 返回 false (bool的零值)

ptrMap := NewMap[*User]()
nilPtr := ptrMap.MustGet("missing")  // 返回 nil (指针的零值)

structMap := NewMap[User]()
zeroUser := structMap.MustGet("missing")  // 返回 User{} (结构体的零值)
```

### Range方法详解

Range方法提供了一种安全高效的遍历方式：

```go
scoreMap := NewMap[int]()
scoreMap.Put("math", 95)
scoreMap.Put("english", 88)
scoreMap.Put("physics", 92)

// 完整遍历
scoreMap.Range(func(subject string, score int) bool {
    fmt.Printf("%s: %d分\n", subject, score)
    return true // 继续遍历
})

// 条件遍历（查找第一个90分以上的科目）
scoreMap.Range(func(subject string, score int) bool {
    if score >= 90 {
        fmt.Printf("找到: %s (%d分)\n", subject, score)
        return false // 停止遍历
    }
    return true // 继续查找
})

// 统计计算
totalScore := 0
count := 0
scoreMap.Range(func(subject string, score int) bool {
    totalScore += score
    count++
    return true
})
average := float64(totalScore) / float64(count)
```

### Find方法详解

Find方法用于查找第一个满足条件的键值对，并返回匹配的项：

```go
// 查找第一个值大于50的元素
key, value, found := scoreMap.Find(func(k string, v int) bool {
    return v > 90
})

if found {
    fmt.Printf("找到: %s = %d分\n", key, value)
} else {
    fmt.Println("没有找到满足条件的元素")
}

// 查找特定的key
key, value, found = scoreMap.Find(func(k string, v int) bool {
    return k == "math"
})
```

### FindAll方法详解

FindAll方法用于查找所有满足条件的键值对：

```go
// 查找所有值大于90的元素
results := scoreMap.FindAll(func(k string, v int) bool {
    return v > 90
})

fmt.Printf("找到%d个匹配的元素:\n", len(results))
for _, kv := range results {
    fmt.Printf("  %s: %d分\n", kv.Key, kv.Value)
}

// 查找所有值在某个范围内的元素
results = scoreMap.FindAll(func(k string, v int) bool {
    return v >= 85 && v <= 95
})
```

## 3. OrderedMap 有序字典

OrderedMap 是一个有序字典，内部元素的顺序是固定的（按照插入顺序）。

### 特性
- **泛型支持**: 键必须是可比较类型，值可以是任意类型
- **插入顺序**: 保持元素的插入顺序
- **并发安全**: 使用读写锁保证线程安全
- **高效操作**: 结合哈希表和双向链表，查找是 O(1)，遍历是 O(n)

### 方法

| 方法 | 描述 | 时间复杂度 |
|------|------|------------|
| `Len()` | 返回字典中元素的数量 | O(1) |
| `Get(key)` | 根据键获取值 | O(1) |
| `Set(key, value)` | 设置键值对 | O(1) |
| `List()` | 返回所有键值对的有序列表 | O(n) |
| `Find(predicate)` | 查找满足条件的键值对 | O(n) |
| `Delete(key)` | 删除指定键的键值对 | O(1) |
| `Keys()` | 返回所有键的有序列表 | O(n) |
| `Values()` | 返回所有值的有序列表 | O(n) |
| `Front()` | 返回第一个键值对 | O(1) |
| `Back()` | 返回最后一个键值对 | O(1) |
| `ForEach(fn)` | 遍历所有键值对 | O(n) |

### 使用示例

```go
// 创建字符串到整数的有序字典
scores := NewOrderedMap[string, int]()

// 添加键值对
scores.Set("Alice", 95)
scores.Set("Bob", 87)
scores.Set("Charlie", 92)

// 获取值
if score, exists := scores.Get("Alice"); exists {
    fmt.Printf("Alice的分数: %d\n", score)
}

// 按插入顺序遍历
scores.ForEach(func(name string, score int) {
    fmt.Printf("%s: %d分\n", name, score)
})
```

## 使用场景

### Set 集合适用场景
- **去重操作**: 快速去除重复元素
- **权限检查**: 检查用户是否拥有特定权限
- **标签管理**: 管理不重复的标签集合
- **集合运算**: 需要进行并集、交集、差集操作

### Map 泛型字典适用场景
- **缓存系统**: 高性能的内存缓存实现
- **配置管理**: 存储各种类型的配置项
- **计数器**: 统计各种事件的发生次数
- **索引映射**: 建立ID到对象的快速映射
- **会话管理**: 存储用户会话信息
- **数据聚合**: 按键分组聚合数据

### OrderedMap 有序字典适用场景
- **LRU缓存**: 实现最近最少使用缓存
- **配置管理**: 保持配置项的插入顺序
- **历史记录**: 按时间顺序记录操作历史
- **有序数据**: 需要保持键值对插入顺序的场景

## 性能特点

- **内存效率**: Set使用空结构体节省内存，OrderedMap使用双向链表维护顺序
- **并发安全**: 两个数据结构都使用读写锁，支持多goroutine并发访问
- **类型安全**: 使用Go泛型，编译时类型检查，避免运行时类型错误

## 测试

运行测试：
```bash
go test ./structure -v
```

运行性能测试：
```bash
go test ./structure -bench=.
```

## 注意事项

1. Set的`List()`方法返回的元素顺序不保证，因为内部使用哈希表存储
2. OrderedMap的键必须是可比较类型（支持`==`和`!=`操作）
3. 两个数据结构都是并发安全的，但如果在高并发场景下使用，建议进行性能测试
4. 删除操作会立即释放内存，适合长期运行的应用