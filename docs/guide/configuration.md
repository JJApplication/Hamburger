# 配置指南

Hamburger 使用结构化配置文件管理运行参数，推荐以主配置为入口，再按业务拆分子配置文件。

## 配置文件结构

| 文件 | 作用 |
| --- | --- |
| `config/config.hamburger` | 主配置入口，定义服务端口、中间件、数据库、实验模块等 |
| `config/frontend.hamburger` | 前端代理与静态站点相关配置 |
| `config/services.hamburger` | 服务列表与域名映射配置 |
| `config/trojan.hamburger` | Trojan 实验模块配置入口 |

推荐统一使用 `.hamburger` 作为主配置和子配置格式，便于复用环境变量、表达式和字符串插值能力。

## 主配置关键字段

```hamburger
{
  pxy_frontend_file: "config/frontend.hamburger",
  pxy_backend_file: "config/backend.hamburger",
  servers: {
    gateway: {
      enable_http3: true,
      enable_fast_proxy: true
    }
  },
  middleware: {
    gateway: {
      prehandlers: {
        pre_auth: {
          enabled: true,
          mode: "jwt"
        }
      }
    }
  }
}
```

## Stat 统计与历史监控

生产环境可使用以下配置启用分钟级历史统计：

```hamburger
stat: {
  db_file: "data/stat.db",
  use_db: true,
  compatible: true,
  enabled: true,
  enable_stat: true,
  sync_duration: 720,
  save_duration: 3600,
  save_file: "data/stat.json",
  geo_file: "data/geo.json",
  domain_file: "data/domain.json",
  sequence: {
    enabled: true,
    interval: 60,
    retention_days: 30,
    flush_interval: 5,
    cleanup_interval: 3600
  }
}
```

`use_db` 仅控制旧的累计 Stat/Geo/Domain 数据是否写入 SQLite；`sequence.enabled` 开启后，历史数据始终根据 `db_file` 初始化固定表，即使 `use_db` 为 `false`。历史保留最近 30 天，按分钟存储并按小时清理。`GET /api/stat` 默认查询 `1h`，还支持 `5h`、`24h`、`7d` 和 `30d`；统计、GEO、域名和连接接口均不需要 JWT。独立前端位于 `gateway/api/stat_web`，通过 `VITE_API_BASE_URL` 指向 API，不由 Go 服务托管。

## Connect Protocol

Connect 可复用已启用网关监听器的端口，不会创建新的 API 监听器。默认配置如下：

```hamburger
connect_protocol: {
  enabled: false,
  base_route: "/hamburger.service",
  enable_bidi_stream: false
}
```

开启后，内置 API 会以 Protobuf Connect RPC 暴露。例如统计接口使用 `POST /hamburger.service/stat`，请求体为 `{"range":"1h","domain":""}`；现有 `/api/*` REST 路由保持不变。`base_route` 可以改为任意合法的绝对路径，客户端应使用生成代码中的 `NewServiceClientAtBaseRoute`。配置为自定义路径时只挂载该路径，默认的 `/hamburger.service/*` 不会再作为 Connect 别名提供。

`enable_bidi_stream` 开启后会额外提供 `statStream`、`geoStream` 等流式方法；一条流中的每个请求按顺序产生一个响应。双向流必须使用 HTTP/2，普通一元调用支持 HTTP/1.1、HTTP/2 和 HTTP/3。受保护方法仍使用 `api_server_config.jwt` 的令牌规则。

一元 Connect 调用可以直接使用 protobuf-JSON（默认监听器为 `http://127.0.0.1:80`）：

```bash
curl -X POST http://127.0.0.1/hamburger.service/stat \
  -H 'Content-Type: application/json' \
  -d '{"range":"1h","domain":""}'
```

生成的 Go 客户端仍直接使用 protobuf 消息；定制路由通过配置感知的构造器访问：

```go
client, err := connect.NewServiceClientAtBaseRoute(http.DefaultClient,
    "https://gateway.example", "/rpc")
if err != nil { log.Fatal(err) }
stream := client.StatStream(context.Background())
defer stream.CloseRequest()
_ = stream.Send(&connect.StatRequest{Range: "1h"})
response, err := stream.Receive()
```

流式示例要求网关监听器启用 HTTP/2，并将 `enable_bidi_stream` 设为 `true`。

若需重新生成绑定，安装 `protoc`、`protoc-gen-go` 和 `protoc-gen-connect-go` 后执行 `go generate ./app/connect`。

## 配置组织建议

- 按职责拆分：入口配置、前端配置、服务映射、实验配置分离管理
- 按环境管理：为开发、测试、生产维护独立配置副本
- 先校验后启动：上线前使用 `test` 子命令验证配置有效性
- 保持一致性：`services.hamburger` 中的域名声明与服务端口配置同步更新

## 注意事项

- `pxy_backend_file` 指向的文件应确保存在并可解析
- 启用 PreAuth 时应同步配置认证模式和凭证来源
- 开启 HTTP3 时需正确准备证书与监听端口策略

## 高级配置

`.hamburger` 支持面向生产环境的高级语法能力：环境变量引用、常量符号、表达式计算、字符串插值拼接。

### 1) `.env` 文件自动加载

启动时会优先根据主配置文件所在目录，自动扫描并加载该目录下的 `*.env` 文件。

- 例如主配置为 `config/config.hamburger`，会扫描 `config/*.env`
- 会按文件名排序后依次加载
- 已存在于进程环境中的变量不会被 `.env` 覆盖

示例文件 `config/hamburger.env`：

```env
ROOT=/srv/hamburger
API_PORT=18080
FEATURE_ENABLED=true
```

### 2) 环境变量 `$ENV`

可直接使用 `$变量名` 读取环境变量并自动按目标字段类型转换。

```hamburger
{
  api_server_config: {
    enabled: $FEATURE_ENABLED,
    port: $API_PORT
  }
}
```

如果变量不存在会直接报错，避免带着错误配置启动。

### 3) 双引号字符串中的环境变量拼接

当值使用双引号包裹时，会在字符串内执行 `$ENV` 插值拼接，支持前缀、中间、后缀位置。

```hamburger
{
  storage_root: "$ROOT/data",
  upload_dir: "data/$ROOT",
  backup_dir: "backup/$ROOT/archive"
}
```

说明：

- 该拼接能力仅在双引号字符串中生效
- 不使用双引号时，`$ROOT/data` 会被当作表达式而不是字符串拼接

### 4) 内置常量 `@NAME`

支持直接引用内置常量与运行时信息：

- `@AppName`
- `@Copyright`
- `@DATE`
- `@DATETIME`
- `@ARCH`
- `@GOOS`
- `@GOVERSION`
- `@NUMCORE`
- `@KERNEL`

```hamburger
{
  custom_header: {
    Proxy-Server: @AppName
  },
  build_os: @GOOS
}
```

### 5) 表达式计算

支持算术、位运算与括号优先级，适合端口、容量、并发等配置计算。

```hamburger
{
  max_cores: 1 << 2,
  worker_count: (2 + 2) * 4,
  cache_size: 1024 * 1024 * 64
}
```

支持操作符：

- 算术：`+ - * / %`
- 位运算：`<< >> & | ^`
- 一元运算：`+ - ~`

### 6) 高级配置综合示例

```hamburger
{
  pxy_frontend_file: "config/frontend.hamburger",
  pxy_backend_file: "config/backend.hamburger",

  api_server_config: {
    enabled: true,
    host: "0.0.0.0",
    port: $API_PORT
  },

  custom_header: {
    Proxy-Server: @AppName,
    Deploy-Date: @DATE
  },

  log: {
    path: "$ROOT/logs/hamburger.log"
  },

  max_cores: 1 << 3
}
```

## 服务映射配置

`service_map.go` 当前使用的是 v2 版本服务映射模型，配置文件已从 `domains.hamburger` 升级为 `services.hamburger`。

### v1：`domains.hamburger` 配置示例

v1 版本使用 `domain_service + services` 两段式结构：前者负责域名到服务名的映射，后者负责服务类型和扩展参数。

```hamburger
{
  domain_service: [
    {
      domain: "blog.renj.io",
      service: "BlogNext"
    },
    {
      domain: "gallery.renj.io",
      service: "Palace"
    },
    {
      domain: "archive.renj.io",
      service: "Archive"
    }
  ],
  services: [
    {
      host: "127.0.0.1",
      port: 3000,
      proxy_pass: [
        {
          api: "/api",
          service: "RustBlog",
          use_rewrite: false
        },
        {
          api: "/images",
          static_direct: {
            static_root: "/renj.io/app/Blog/images"
          }
        }
      ],
      service_name: "BlogNext",
      service_type: "custom"
    },
    {
      service_name: "Palace",
      service_type: "frontend"
    },
    {
      service_name: "Archive",
      service_type: "frontend"
    },
    {
      service_name: "RustBlog",
      service_type: "backend"
    }
  ]
}
```

### v2：`services.hamburger` 配置示例

v2 版本将域名配置合并进单个服务项，统一通过 `service_domain` 描述域名或正则表达式，配置结构更直接。

```hamburger
{
  services: [
    {
      service_name: "BlogNext",
      service_type: "custom",
      service_domain: "blog.renj.io",
      host: "127.0.0.1",
      port: 3000,
      proxy_pass: [
        {
          api: "/api",
          service: "RustBlog",
          use_rewrite: false
        },
        {
          api: "/images",
          static_direct: {
            static_root: "/renj.io/app/Blog/images"
          }
        }
      ]
    },
    {
      service_name: "Palace",
      service_type: "frontend",
      service_domain: "/(life|gallery).renj.io/"
    },
    {
      service_name: "Archive",
      service_type: "frontend",
      service_domain: "/(^v1|archive).renj.io/"
    },
    {
      service_name: "RustBlog",
      service_type: "backend"
    }
  ]
}
```

### 从 v1 迁移到 v2

核心变化是将“域名映射”从独立数组合并到服务定义本身：

| v1 字段 | v2 字段 | 迁移说明 |
| --- | --- | --- |
| `domain_service[].domain` | `services[].service_domain` | 将域名直接写入对应服务项 |
| `domain_service[].service` | `services[].service_name` | 继续通过服务名关联，但不再单独维护映射表 |
| `services[].service_type` | `services[].service_type` | 保持不变 |
| `services[].host/port/proxy_pass` | `services[].host/port/proxy_pass` | 自定义服务扩展参数保持不变 |

推荐按以下步骤迁移：

1. 以 `domain_service[].service` 为键，对照找到 `services[]` 中对应的服务定义。
2. 将 `domain_service[].domain` 写入该服务的 `service_domain` 字段。
3. 删除旧的 `domain_service` 数组，仅保留统一的 `services` 数组。
4. 如果一个服务需要响应多个域名，可改为正则表达式形式，例如 `/(life|gallery).renj.io/`。
5. 纯后端内部服务如果不直接对外暴露域名，可以继续不填写 `service_domain`。

### 自定义服务

`service_type: "custom"` 仍用于在同一个服务下组合 API 转发与静态文件代理能力，v2 只是把域名入口从独立映射改为 `service_domain` 内联配置。

### 核心行为

- `proxy_pass`：按 `api` 前缀匹配请求并转发；默认转发到 `service` 对应服务的端口
- `static_direct`：当该配置不为空时，表示静态文件代理，行为类似 nginx 的 `alias`
- `service_domain`：支持直接域名和正则表达式，同一域名仍然只能映射一个服务
- 路径映射：会将 `api` 后的请求路径拼接到 `static_root`，并转发到本地路径

例如：

- 请求 `/images/photo.jpg`
- 若 `api` 为 `/images`，`static_root` 为 `/renj.io/app/Blog/images`
- 实际读取本地路径 `/renj.io/app/Blog/images/photo.jpg`
