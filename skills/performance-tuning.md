# Skill: 性能优化

## 概述
Hamburger 基于高性能网络库构建，默认配置已优化。本文档提供进一步调优指南。

## 网络层优化

### 传输模式选择

```hamburger
proxy: {
  transport: "fasthttp",    # fasthttp > standard
  proxy_mode: "fasthttp",
  net_io: "nbio",           # nbio > standard
}
```

**推荐配置**
- `transport: "fasthttp"` - 比标准库快 10 倍
- `net_io: "nbio"` - 非阻塞 I/O，高并发场景优势明显

### 连接池调优

```hamburger
proxy: {
  max_conns_per_host: 100,    # 每主机最大连接数
  idle_conn_timeout: 60       # 空闲连接超时 (秒)
}
```

**调优建议**

| 场景 | max_conns_per_host | idle_conn_timeout |
|------|-------------------|-------------------|
| 低流量 (<100 req/s) | 20 | 30 |
| 中流量 (100-1000 req/s) | 50 | 60 |
| 高流量 (>1000 req/s) | 100-200 | 120 |

### 缓冲区配置

```hamburger
proxy: {
  buf_size: 65536,           # 64KB 读写缓冲区
  flush_interval: 0          # 0 = 立即刷新
}
```

**调优建议**
- `buf_size` 增大可提升大文件传输性能
- `flush_interval: 0` 确保低延迟

## HTTP/2 调优

### 网关 HTTP/2

```hamburger
servers: [
  {
    use_http2: true,
    http2: {
      max_concurrent_streams: 128,    # 每连接最大流数
      max_handlers: 100,              # 最大处理程序数
      idle_timeout: 60,               # 空闲超时
      read_idle_timeout: 60           # 读空闲超时
    }
  }
]
```

**调优建议**

| 参数 | 低并发 | 高并发 |
|------|--------|--------|
| max_concurrent_streams | 64 | 256+ |
| max_handlers | 50 | 200+ |
| idle_timeout | 30 | 120 |

### 前端代理 HTTP/2

```hamburger
exp_fast_connect: {
  enabled: true,
  http2: {
    read_timeout: 30,
    write_timeout: 30,
    idle_timeout: 60,
    read_header_timeout: 10,
    max_header_bytes: 50 * 1<<20,    # 50MB
    keep_alive: 30,
    max_handlers: 256,
    max_concurrent_streams: 512,
    max_upload_buffer_per_connection: 16 * 1<<20,
    max_upload_buffer_per_stream: 16 * 1<<20
  }
}
```

**调优建议**
- `max_concurrent_streams: 512` - 支持更多并发流
- `max_upload_buffer` - 大文件上传时增大

## HTTP/3 (QUIC)

```hamburger
exp_fast_connect: {
  http3: {
    enabled: false,           # 实验性
    max_connections: 100,
    idle_timeout: 60,
    keep_alive: 30,
    insecure_skip_verify: true
  }
}
```

**注意**：HTTP/3 为实验性功能，生产环境谨慎使用。

## 缓存优化

### 代理缓存

```hamburger
features: {
  proxy_cache: {
    enabled: true,
    cache_size: 2048,      # 缓存条目数
    cache_ttl: 3600        # 缓存时间 (秒)
  }
}
```

**调优建议**
- `cache_size` 根据可用内存调整
- `cache_ttl` 根据内容更新频率调整

### 前端静态缓存

```hamburger
cache: {
  enable: true,
  dir: "/renj.io/cache",
  expire: 0,               # 0 = 永不过期
  matcher: [
    "*.html", "*.css", "*.js",
    "*.svg", "*.ico", "*.woff", "*.woff2",
    "*.ttf", "*.otf", "*.eot", "*.map"
  ]
}
```

**调优建议**
- 为静态资源配置长缓存
- HTML 文件设置较短缓存或禁用

## Gzip 压缩

```hamburger
middleware: {
  gzip: {
    enabled: true,
    level: 6,              # 1-9
    threshold: 2048,       # 最小压缩大小
    types: [...]
  }
}
```

**压缩级别性能对比**

| 级别 | CPU 使用 | 压缩率 | 推荐场景 |
|------|---------|--------|---------|
| 1-3 | 低 | 60-70% | CPU 受限 |
| 4-6 | 中 | 70-80% | 通用推荐 |
| 7-9 | 高 | 80-85% | 带宽受限 |

**threshold 建议**
- 2048 (2KB) - 通用推荐
- 小文件压缩收益低且消耗 CPU

## 流量控制

### 限流保护

```hamburger
flow_control: {
  enabled: true,
  global_limit: {
    requests: 10000,
    window: "60s"
  },
  rules: [
    {
      name: "api-limit",
      match_type: "host",
      match_value: "renj.io",
      limits: [
        { requests: 1000, window: "10s" },
        { requests: 5000, window: "10min" }
      ],
      action: "block"
    }
  ]
}
```

**调优建议**
- 多层限流：全局 + 域名 + 路径
- 使用滑动窗口或令牌桶算法更平滑
- 记录被拦截请求用于分析攻击模式

## 资源限制

### CPU 核心

```hamburger
max_cores: 4    # 限制使用的 CPU 核心数
```

**建议**
- 设置为物理核心的 50-75%
- 为其他服务保留资源

### 请求体限制

```hamburger
servers: [
  {
    max_request_body: 50 * 1<<20    # 50MB
  }
]
```

**建议**
- API 服务：1-10MB
- 文件上传：50-100MB
- 防止恶意大请求

## 日志优化

```hamburger
log: {
  log_level: "error",    # 生产环境使用 error
  log_file: "",          # stdout 性能更好
  color: false           # 生产环境禁用彩色
}
```

## 统计优化

```hamburger
stat: {
  enabled: true,
  use_db: true,
  sync_duration: 720,      # 同步间隔 (秒)
  save_duration: 3600,     # 保存间隔 (秒)
  enable_stat: true
}
```

**调优建议**
- 增大 `sync_duration` 减少数据库写入
- 高流量场景使用 `use_db: false` 仅内存统计

## pprof 性能分析

```hamburger
pprof: {
  enable: true,
  port: 8889
}
```

**常用分析命令**
```bash
# CPU Profile (30秒)
go tool pprof http://localhost:8889/debug/pprof/profile?seconds=30

# 内存 Profile
go tool pprof http://localhost:8889/debug/pprof/heap

# Goroutine
go tool pprof http://localhost:8889/debug/pprof/goroutine

# 阻塞 Profile
go tool pprof http://localhost:8889/debug/pprof/block
```

## 监控指标

### 关键指标
- **QPS** - 每秒请求数
- **延迟** - P50/P95/P99 响应时间
- **错误率** - 4xx/5xx 比例
- **连接数** - 活跃连接数
- **CPU/内存** - 资源使用率

### 获取方式
```bash
# 连接统计
curl http://127.0.0.1:8888/api/conn

# 流量统计
curl http://127.0.0.1:8888/api/stat

# 系统资源
top -p $(cat hamburger.pid)
```

## 调优检查清单

### 网络层
- [ ] 使用 `fasthttp` + `nbio`
- [ ] 调整连接池参数
- [ ] 配置合适的缓冲区

### HTTP/2
- [ ] 启用 HTTP/2
- [ ] 调整并发流数
- [ ] 配置超时参数

### 缓存
- [ ] 启用代理缓存
- [ ] 配置静态文件缓存
- [ ] 设置合理的 TTL

### 压缩
- [ ] 启用 Gzip
- [ ] 选择合适的压缩级别
- [ ] 设置压缩阈值

### 安全
- [ ] 配置限流规则
- [ ] 设置请求体限制
- [ ] 启用安全头

### 资源
- [ ] 限制 CPU 核心
- [ ] 优化日志级别
- [ ] 调整统计频率
