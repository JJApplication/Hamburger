# Skill: 路由与代理

## 概述
Hamburger 实现了三层代理架构：
1. **Gateway** - 核心网关，处理域名路由和反向代理
2. **Frontend Proxy** - 静态文件服务（Helios）
3. **Backend Proxy** - 后端 API 代理

## 请求流程

```
Client → Gateway (80/443)
         ├── 域名匹配 → Frontend Proxy (7777) → 静态文件
         │                                      └── API 路径 → Backend Proxy
         └── 直接代理 → Backend Service
```

## 域名路由

### 域名映射配置
在 `domains.hamburger` 中定义域名到服务的映射：

```hamburger
{
  # 仅前端
  "renj.io": {
    frontend: "Homeland"
  },
  
  # 前后端都有
  "blog.renj.io": {
    frontend: "BlogFront",
    backend: "Blog"
  },
  
  # 仅后端
  "service.renj.io": {
    backend: "Apollo"
  }
}
```

### 域名匹配规则
- 精确匹配优先
- 支持通配符域名（如 `*.renj.io`）
- 域名在 `servers[].domains` 中声明才会被监听

## 前端代理 (Helios)

### 基础配置
```hamburger
{
  host: "127.0.0.1",
  port: 7777,
  balancer: "http://127.0.0.1:80",   # 回退到网关
  cache: {
    enable: true,
    dir: "/renj.io/cache",
    expire: 0,    # 0 = 永不过期
    matcher: [
      "*.html", "*.css", "*.js",
      "*.svg", "*.ico", "*.woff", "*.woff2"
    ]
  }
}
```

### 静态站点配置
```hamburger
servers: [
  {
    type: "WebServer",
    name: "Homeland",           # 与 domains.hamburger 中的 frontend 值对应
    root: "/renj.io/app/Homeland",
    index: "index.html",
    try_file: "index.html",     # SPA 路由回退
    access: false,
    compress: false
  }
]
```

### 前端 + 后端组合
```hamburger
{
  name: "BlogFront",
  root: "/renj.io/app/BlogFront",
  index: "index.html",
  try_file: "index.html",
  
  # 路径别名
  alias: {
    "/images": "/renj.io/app/Blog/images"
  },
  
  # API 代理
  backends: [
    {
      api: "/api",              # 匹配路径前缀
      service: "Blog",          # 后端服务名
      use_rewrite: false        # 是否重写路径
    },
    {
      api: "/static",
      service: "Blog",
      use_rewrite: false
    }
  ]
}
```

## 后端代理

### 服务注册
后端服务在 `backend.hamburger` 中定义：

```hamburger
{
  services: [
    {
      name: "Blog",
      host: "127.0.0.1",
      port: 8080,
      protocol: "http",
      health_check: {
        enabled: true,
        path: "/health",
        interval: 30
      }
    }
  ]
}
```

### 代理模式
```hamburger
proxy: {
  transport: "fasthttp",    # fasthttp (高性能) | standard
  proxy_mode: "fasthttp",
  net_io: "nbio",           # nbio (非阻塞) | standard
  max_conns_per_host: 100,
  idle_conn_timeout: 60
}
```

## 内部头传递

Hamburger 使用特殊头在组件间传递上下文：

| 头 | 用途 |
|---|---|
| `X-Proxy-Internal-Host` | 原始请求 Host |
| `X-Proxy-Internal-Local` | 本地代理标记 |
| `X-Proxy-Backend` | 后端服务标识 |
| `X-Proxy-Internal-Front` | 前端代理标记 |
| `X-Forward-Host` | 转发 Host |
| `X-Gateway-Trace-Id` | 请求追踪 ID |

## 负载均衡

### 配置方式
```hamburger
{
  balancer: "http://127.0.0.1:80",    # 单后端
  # 或
  backends: [
    {
      service: "Blog",
      hosts: ["127.0.0.1:8080", "127.0.0.1:8081"],
      strategy: "round_robin"
    }
  ]
}
```

## WebSocket 支持

```hamburger
features: {
  websocket: {
    enabled: true
  }
}
```

WebSocket 连接自动升级，无需额外配置。

## gRPC 代理

Hamburger 支持 gRPC 和 gRPC-Web 代理：

```hamburger
# 在 backend 配置中指定 gRPC 服务
{
  name: "GrpcService",
  protocol: "grpc",
  host: "127.0.0.1",
  port: 50051
}
```

## 最佳实践

1. **SPA 应用** - 始终配置 `try_file: "index.html"` 支持前端路由
2. **API 路径分离** - 使用 `backends` 将 `/api` 等路径代理到后端
3. **静态缓存** - 为 CSS/JS/字体配置缓存，提升加载速度
4. **健康检查** - 为后端服务配置健康检查实现自动故障转移
5. **Trace ID** - 启用 trace 中间件便于请求追踪
