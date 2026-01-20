# Modifier 模块

Modifier 模块提供了统一的响应修改器接口，用于在HTTP响应返回给客户端之前对响应进行各种处理。

## 架构设计

### 核心接口

```go
type Modifier interface {
    ModifyResponse(response *http.Response) error  // 修改HTTP响应
    IsEnabled() bool                               // 检查是否启用
    UpdateConfig()                                 // 更新配置
    GetName() string                              // 获取修改器名称
}
```

### 主要组件

1. **Modifier**: 统一的修改器接口
2. **ModifierChain**: 修改器链，按顺序执行多个修改器
3. **ModifierManager**: 修改器管理器，负责管理和协调所有修改器
4. **GzipModifier**: Gzip压缩修改器
5. **CustomHeaderModifier**: 自定义响应头修改器

## 内置修改器

### 1. GzipModifier (Gzip压缩修改器)

**功能**: 对响应进行gzip压缩以减少传输数据量

**特性**:
- 使用 `sync.Pool` 优化性能，重用压缩器对象
- 支持配置压缩级别和可压缩的MIME类型
- 自动检查客户端是否支持gzip
- 小于1KB的响应不进行压缩
- 支持配置热更新

**配置示例**:
```json
{
  "features": {
    "gzip": {
      "enabled": true,
      "level": 6,
      "types": ["text/html", "text/css", "application/json"]
    }
  }
}
```

### 2. CustomHeaderModifier (自定义响应头修改器)

**功能**: 为响应添加自定义头部

**特性**:
- 支持动态添加/移除头部
- 线程安全的头部管理
- 不会覆盖已存在的头部
- 支持批量设置头部
- 支持配置热更新

**配置示例**:
```json
{
  "custom_header": {
    "X-Powered-By": "Sandwich",
    "X-Server-Version": "v1.0.0",
    "X-Copyright": "Renj"
  }
}
```

## 使用方法

### 基本使用

```go
// 创建修改器管理器
modifierManager := modifier.NewModifierManager()

// 在处理HTTP响应时应用修改器
err := modifierManager.ModifyResponse(response)
if err != nil {
    log.ErrorF("修改响应失败: %v", err)
}
```

### 动态管理自定义头

```go
// 获取自定义头修改器
customHeaderModifier := modifierManager.GetCustomHeaderModifier()
if customHeaderModifier != nil {
    // 添加头部
    customHeaderModifier.AddHeader("X-Custom-Info", "value")
    
    // 移除头部
    customHeaderModifier.RemoveHeader("X-Old-Header")
    
    // 批量设置头部
    headers := map[string]string{
        "X-API-Version": "v2.0",
        "X-Request-ID": "12345",
    }
    customHeaderModifier.SetHeaders(headers)
}
```

### 创建自定义修改器

```go
type MyCustomModifier struct {
    enabled bool
}

func (m *MyCustomModifier) ModifyResponse(response *http.Response) error {
    if !m.enabled {
        return nil
    }
    
    // 自定义处理逻辑
    response.Header.Set("X-Custom-Processing", "done")
    return nil
}

func (m *MyCustomModifier) IsEnabled() bool {
    return m.enabled
}

func (m *MyCustomModifier) UpdateConfig() {
    // 更新配置逻辑
}

func (m *MyCustomModifier) GetName() string {
    return "my_custom_modifier"
}

// 添加到管理器
customModifier := &MyCustomModifier{enabled: true}
modifierManager.AddCustomModifier(customModifier)
```

## 执行顺序

修改器按照添加到链中的顺序执行。默认顺序为：

1. **GzipModifier** - 压缩响应体
2. **CustomHeaderModifier** - 添加自定义头部

可以通过创建自定义的 `ModifierChain` 来控制执行顺序：

```go
chain := modifier.NewModifierChain()
chain.AddModifier(customHeaderModifier)  // 先添加头部
chain.AddModifier(gzipModifier)          // 再压缩
```

## 性能优化

### Gzip压缩优化
- 使用 `sync.Pool` 重用 `gzip.Writer` 和 `bytes.Buffer` 对象
- 避免频繁的内存分配和垃圾回收
- 在高并发场景下显著提升性能

### 自定义头优化
- 使用读写锁保护配置，支持并发读取
- 惰性启用机制，只有配置了头部才启用修改器
- 批量操作减少锁竞争

## 监控和调试

### 获取修改器状态
```go
status := modifierManager.GetStatus()
log.InfoF("总修改器: %d, 启用: %d", 
    status["total_modifiers"], 
    status["enabled_modifiers"])
```

### 调试日志
修改器会输出详细的调试日志，包括：
- 修改器的启用/禁用状态
- 配置更新情况
- 压缩效果统计
- 头部添加情况

启用调试日志：
```json
{
  "log": {
    "log_level": "debug"
  }
}
```

## 最佳实践

1. **合理设置压缩类型**: 只对文本类型内容启用gzip压缩
2. **控制自定义头数量**: 避免添加过多不必要的响应头
3. **注意执行顺序**: 确保修改器按正确顺序执行
4. **使用配置热更新**: 支持运行时动态更新配置
5. **监控性能影响**: 定期检查修改器对性能的影响

## 扩展性

该模块设计为高度可扩展：

- 实现 `Modifier` 接口即可创建新的修改器
- 支持插件式架构，可动态添加修改器
- 配置驱动，支持运行时调整行为
- 完整的生命周期管理（启用/禁用/更新）

## 示例代码

详见 `example.go` 文件，包含：
- 基本使用示例
- 自定义修改器创建示例
- 修改器链执行顺序示例