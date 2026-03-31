# Skill: 中间件与安全

## 概述
Hamburger 内置多种中间件，在请求处理管道中提供安全、性能和可观测性能力。

## CORS 跨域

### 配置
```hamburger
middleware: {
  cors: {
    enabled: true,
    header: [
      "Content-Type",
      "Origin",
      "Authorization",
      "access-token",
      "Token"
    ]
  }
}
```

### 使用场景
- 前后端分离项目
- 多域名 API 访问
- 第三方集成

## Gzip 压缩

### 配置
```hamburger
middleware: {
  gzip: {
    enabled: true,
    level: 6,              # 1-9, 6 是性能与压缩率的最佳平衡
    threshold: 2048,       # 最小压缩大小 (bytes)
    types: [
      "text/html",
      "text/css",
      "text/javascript",
      "text/plain",
      "text/xml",
      "application/javascript",
      "application/xml",
      "application/json",
      "font/ttf",
      "font/woff",
      "font/woff2",
      "font/otf"
    ]
  }
}
```

### 压缩级别建议
| 级别 | 场景 |
|------|------|
| 1-3 | CPU 受限环境 |
| 4-6 | 通用推荐 |
| 7-9 | 带宽受限，CPU 充足 |

## 流量控制 (限流)

### 全局限流
```hamburger
features: {
  flow_control: {
    enabled: true,
    global_limit: {
      requests: 10000,
      window: "60s",
      unit: "s"
    }
  }
}
```

### 规则限流
```hamburger
rules: [
  {
    name: "api-limit",
    enabled: true,
    priority: 1,           # 数字越小优先级越高
    match_type: "host",    # host | path | header
    match_value: "renj.io",
    header_key: "",        # match_type=header 时使用
    limits: [
      { requests: 1000, window: "10s", unit: "s" },
      { requests: 5000, window: "10min", unit: "min" }
    ],
    action: "block",       # block | allow
    description: "API服务限流"
  }
]
```

### 限流算法
Hamburger 实现多种限流算法（`gateway/flow_control/`）：
- **Fixed Window** - 固定窗口
- **Sliding Window** - 滑动窗口
- **Token Bucket** - 令牌桶
- **Leaky Bucket** - 漏桶

### 限流记录
```hamburger
recording: {
  enabled: false,
  record_blocked: true,     # 记录被拦截请求
  record_allowed: false,    # 记录通过请求
  storage_type: "file",
  retention_period: "30d"
}
```

## Trace 追踪

### 配置
```hamburger
middleware: {
  trace: {
    enabled: true,
    trace_id: "X-Gateway-Trace-Id"
  }
}
```

### 工作原理
- 每个请求生成唯一 Trace ID
- Trace ID 传递给后端服务
- 可在日志和响应头中追踪

## 安全头

### 配置
```hamburger
middleware: {
  secure_header: true
}
```

自动添加的安全头包括：
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security`

## IP 黑白名单

### 配置
```hamburger
security: {
  strict_mode: true,
  allow_ips: ["192.168.1.0/24", "10.0.0.1"],
  deny_ips: ["1.2.3.4"],
  rate_limit: 1000
}
```

### 规则
- `allow_ips` 非空时，仅允许列表中的 IP
- `deny_ips` 中的 IP 始终被拒绝
- `strict_mode` 启用严格模式

## 自定义响应头

### 配置
```hamburger
custom_header: {
  "Proxy-Copyright": "renj.io",
  "Proxy-Server": "Hamburger"
}
```

## 错误页面

### 配置
```hamburger
error_config: {
  error_mode: "html",
  enable_page_cache: true,
  error_page: {
    "403": "static/403.html",
    "404": "static/404.html",
    "429": "static/429.html",
    "500": "static/500.html",
    "502": "static/502.html"
  }
}
```

## 代理缓存

### 配置
```hamburger
features: {
  proxy_cache: {
    enabled: true,
    cache_size: 2048,      # 缓存条目数
    cache_ttl: 3600        # 缓存时间 (秒)
  }
}
```

## 最佳实践

1. **始终启用 Trace** - 便于问题排查和请求追踪
2. **合理设置限流** - 全局 + 规则组合，防止滥用
3. **Gzip 阈值** - 小文件不压缩（2KB 起）
4. **安全头** - 生产环境始终启用 `secure_header`
5. **错误页面** - 自定义错误页提升用户体验
6. **IP 白名单** - 管理 API 使用白名单保护
