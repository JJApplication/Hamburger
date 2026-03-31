# Skill: WASM 插件开发

## 概述
Hamburger 支持基于 WASM 的插件系统，使用 `wazero` 作为 WASI 运行时，可在请求/响应处理管道中注入自定义逻辑。

## 插件架构

```
Request → WASM Plugin (request_handle) → Backend → WASM Plugin (response_handle) → Client
```

## 启用插件系统

### 配置
```hamburger
plugin: {
  enabled: true,
  root: "plugins",         # WASM 文件目录
  plugins: [
    {
      name: "my-plugin",
      enabled: true,
      params: {
        key: "value"
      }
    }
  ]
}
```

### 目录结构
```
plugins/
├── my-plugin.wasm         # 编译后的 WASM 文件
└── other-plugin.wasm
```

## WASM 插件接口

### 导出函数
插件必须导出以下函数（至少一个）：

| 函数 | 用途 | 参数 | 返回值 |
|------|------|------|--------|
| `request_handle` | 请求处理 | (ptr, len) | (ptr, len) |
| `response_handle` | 响应处理 | (ptr, len) | (ptr, len) |
| `alloc` / `malloc` | 内存分配 | (size) | ptr |
| `dealloc` / `free` | 内存释放 | (ptr, len) | void |

### 请求上下文 (输入)
```json
{
  "method": "GET",
  "url": "/api/data",
  "host": "renj.io",
  "remote_addr": "1.2.3.4:12345",
  "header": {
    "Content-Type": ["application/json"],
    "Authorization": ["Bearer xxx"]
  }
}
```

### 请求结果 (输出)
```json
{
  "allow": true,
  "status": 200,
  "error": "",
  "header_set": {"X-Custom": "value"},
  "header_add": {"X-Extra": "data"},
  "header_del": ["X-Unwanted"]
}
```

### 响应上下文 (输入)
```json
{
  "status_code": 200,
  "header": {
    "Content-Type": ["application/json"]
  }
}
```

### 响应结果 (输出)
```json
{
  "status": 200,
  "header_set": {"X-Custom": "value"},
  "header_add": {"X-Extra": "data"},
  "header_del": ["X-Unwanted"]
}
```

## 开发示例 (Rust)

### Cargo.toml
```toml
[package]
name = "my-plugin"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
```

### src/lib.rs
```rust
use serde::{Deserialize, Serialize};
use std::alloc::{alloc, dealloc, Layout};

#[derive(Deserialize)]
struct RequestContext {
    method: String,
    url: String,
    host: String,
    header: std::collections::HashMap<String, Vec<String>>,
}

#[derive(Serialize)]
struct RequestResult {
    allow: bool,
    status: u16,
    error: String,
    header_set: std::collections::HashMap<String, String>,
    header_add: std::collections::HashMap<String, String>,
    header_del: Vec<String>,
}

#[no_mangle]
pub extern "C" fn request_handle(ptr: *mut u8, len: usize) -> *mut u8 {
    let slice = unsafe { std::slice::from_raw_parts(ptr, len) };
    let ctx: RequestContext = serde_json::from_slice(slice).unwrap();
    
    let mut result = RequestResult {
        allow: true,
        status: 0,
        error: String::new(),
        header_set: std::collections::HashMap::new(),
        header_add: std::collections::HashMap::new(),
        header_del: Vec::new(),
    };
    
    // 示例：添加自定义头
    result.header_set.insert(
        "X-Plugin-Processed".to_string(),
        "true".to_string()
    );
    
    // 示例：基于路径拒绝请求
    if ctx.url.starts_with("/blocked") {
        result.allow = false;
        result.error = "Blocked by plugin".to_string();
    }
    
    let json = serde_json::to_vec(&result).unwrap();
    let out_ptr = alloc(Layout::from_size_align(json.len(), 8).unwrap());
    unsafe {
        std::ptr::copy_nonoverlapping(json.as_ptr(), out_ptr, json.len());
    }
    
    // 编码为 ptr+len (64位)
    ((json.len() as u64) << 32) | (out_ptr as u64) as *mut u8
}

#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut u8 {
    let layout = Layout::from_size_align(size, 8).unwrap();
    unsafe { alloc(layout) }
}

#[no_mangle]
pub extern "C" fn dealloc(ptr: *mut u8, size: usize) {
    unsafe {
        dealloc(ptr, Layout::from_size_align(size, 8).unwrap());
    }
}
```

### 编译
```bash
# 添加 WASM 目标
rustup target add wasm32-wasi

# 编译
cargo build --target wasm32-wasi --release

# 复制 WASM 文件
cp target/wasm32-wasi/release/my_plugin.wasm /path/to/hamburger/plugins/
```

## 插件生命周期

1. **加载** - 启动时扫描 `plugin.root` 目录，加载所有 `.wasm` 文件
2. **实例化** - 创建 WASI 运行时实例
3. **配置** - 从 `plugin.plugins` 读取插件配置
4. **请求处理** - 按加载顺序调用 `request_handle`
5. **响应处理** - 按加载顺序调用 `response_handle`
6. **卸载** - 关闭时释放 WASM 模块

## 内存管理

### 分配/释放协议
```
Hamburger → alloc(size) → ptr
Hamburger → request_handle(ptr, len) → result_ptr
Hamburger → dealloc(result_ptr, result_len)
```

### 返回值编码
```
result = (len << 32) | ptr
```

## 插件配置传递

插件可通过 `params` 接收配置：
```hamburger
plugins: [
  {
    name: "rate-limiter",
    enabled: true,
    params: {
      max_requests: 100,
      window: "60s"
    }
  }
]
```

## 错误处理

- 插件返回 `allow: false` 会拒绝请求
- 插件返回 `error` 字符串会作为错误信息
- 插件抛出异常会记录日志并继续

## 最佳实践

1. **最小化依赖** - WASM 文件越小加载越快
2. **快速返回** - 插件不应阻塞请求处理
3. **错误容错** - 插件失败不应导致网关崩溃
4. **配置驱动** - 使用 params 传递配置而非硬编码
5. **日志记录** - 在插件中记录关键操作便于调试
