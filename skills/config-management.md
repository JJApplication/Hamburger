# Skill: 配置管理

## 概述
Hamburger 支持两种配置格式：
1. **标准 JSON** - `config.json` 系列文件
2. **Hamburger DSL** - `config.hamburger` 系列文件（推荐，支持变量和表达式）

## DSL 配置语法

### 变量系统
```hamburger
# 内置变量
host: @ZeroHost          # 0.0.0.0
port: @HTTPPort          # 80
host: @Localhost         # 127.0.0.1
port: @HTTPSPort         # 443

# 自定义变量引用
custom_header: {
  Proxy-Copyright: @Copyright
  Proxy-Server: @AppName
}
```

### 表达式支持
```hamburger
max_request_body: 50 * 1<<20    # 50MB
max_header_bytes: 50 * 1<<20    # 50MB
max_upload_buffer: 16 * 1<<20   # 16MB
```

### 注释
```hamburger
# 这是单行注释
{
  # 配置项
  enabled: true
}
```

## 配置文件结构

### 主配置入口
```
config/config.hamburger
├── pxy_backend_file    → 后端代理配置
├── pxy_frontend_file   → 前端代理配置
└── domain_map          → 域名映射配置
```

### 核心配置块

#### servers - 网关监听
```hamburger
servers: [
  {
    name: "http-server",
    host: @ZeroHost,
    port: @HTTPPort,
    protocol: "http",        # http | https
    enabled: true,
    use_http2: true,
    max_request_body: 50 * 1<<20,
    domains: [
      {
        domains: ["renj.io", "*.renj.io"],
        auto_redirect: true
      }
    ]
  }
]
```

HTTPS 服务器额外配置：
```hamburger
{
  protocol: "https",
  tls: {
    min_version: "TLS13",
    cert_map: {
      "renj.io": {
        domains: ["renj.io", "*.renj.io"],
        cert_file: "config/ssl/fullchain.cer",
        key_file: "config/ssl/ssl.key"
      }
    },
    auto_tls: false    # 自动证书管理
  },
  http2: {
    max_concurrent_streams: 128,
    max_handlers: 100,
    idle_timeout: 60,
    read_idle_timeout: 60
  }
}
```

#### proxy - 代理引擎
```hamburger
proxy: {
  flush_interval: 0,
  buf_size: 65536,
  transport: "fasthttp",     # fasthttp | standard
  proxy_mode: "fasthttp",
  net_io: "nbio",            # nbio | standard
  max_conns_per_host: 100,
  idle_conn_timeout: 60
}
```

#### features - 功能模块
```hamburger
features: {
  flow_control: {
    enabled: true,
    global_limit: {
      requests: 10000,
      window: "60s"
    },
    rules: [
      {
        name: "api-limit",
        enabled: true,
        priority: 1,
        match_type: "host",
        match_value: "renj.io",
        limits: [
          { requests: 1000, window: "10s" },
          { requests: 5000, window: "10min" }
        ],
        action: "block"    # block | allow
      }
    ]
  },
  websocket: {
    enabled: true
  },
  proxy_cache: {
    enabled: true,
    cache_size: 2048,
    cache_ttl: 3600
  }
}
```

#### middleware - 中间件
```hamburger
middleware: {
  cors: {
    enabled: true,
    header: ["Content-Type", "Origin", "Authorization"]
  },
  trace: {
    enabled: true,
    trace_id: "X-Gateway-Trace-Id"
  },
  secure_header: true,
  gzip: {
    enabled: true,
    level: 6,
    types: ["text/html", "application/json", ...],
    threshold: 2048
  }
}
```

#### database - 数据库
```hamburger
database: {
  mongo: {
    url: "mongodb://localhost:27017",
    database: "ApolloMongo",
    timeout: 10
  },
  influx: {
    enabled: false,
    url: "http://localhost:8086",
    token: "",
    org: "sandwich",
    bucket: "sandwich"
  }
}
```

#### stat - 统计
```hamburger
stat: {
  db_file: "data/stat.db",
  use_db: true,
  enabled: true,
  enable_stat: true,
  sync_duration: 720,      # 秒
  save_duration: 3600,     # 秒
  save_file: "data/stat.json",
  geo_file: "data/geo.json",
  domain_file: "data/domain.json"
}
```

#### api_server_config - 管理 API
```hamburger
api_server_config: {
  enabled: true,
  host: @Localhost,
  port: 8888,
  http2: {
    enabled: true,
    insecure: true
  },
  bblot: {
    enabled: true,
    file: "data/hamburger-api.db"
  }
}
```

## 子配置文件

### domains.hamburger - 域名映射
```hamburger
{
  "renj.io": {
    frontend: "Homeland"
  },
  "blog.renj.io": {
    frontend: "BlogFront",
    backend: "Blog"
  },
  "service.renj.io": {
    backend: "Apollo"
  }
}
```

### frontend.hamburger - 前端服务
```hamburger
{
  host: @Localhost,
  port: 7777,
  balancer: "http://127.0.0.1:80",
  cache: {
    enable: true,
    dir: "/renj.io/cache",
    matcher: ["*.html", "*.css", "*.js", "*.svg"]
  },
  servers: [
    {
      type: "WebServer",
      name: "Homeland",
      root: "/renj.io/app/Homeland",
      index: "index.html",
      try_file: "index.html",    # SPA fallback
      compress: false
    },
    {
      name: "BlogFront",
      root: "/renj.io/app/BlogFront",
      alias: {
        "/images": "/renj.io/app/Blog/images"
      },
      backends: [
        {
          api: "/api",
          service: "Blog",
          use_rewrite: false
        }
      ]
    }
  ]
}
```

## 配置验证

```bash
# 测试配置文件
./hamburger test -c config/config.hamburger
```

## 配置生成

```bash
# 交互式生成配置
./hamburger generate
```

## 最佳实践

1. **使用 DSL 格式** - 支持变量、表达式和注释，更易维护
2. **拆分配置文件** - 将 backend/frontend/domains 分离到独立文件
3. **环境变量** - 使用 `@Variable` 语法注入运行时值
4. **配置验证** - 部署前始终运行 `hamburger test`
5. **版本控制** - 将 `.hamburger` 文件纳入版本控制，`.json` 文件可生成
