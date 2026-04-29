# 配置指南

Hamburger 使用结构化配置文件管理运行参数，推荐以主配置为入口，再按业务拆分子配置文件。

## 配置文件结构

| 文件 | 作用 |
| --- | --- |
| `config/config.json` | 主配置入口，定义服务端口、中间件、数据库、实验模块等 |
| `config/frontend.json` | 前端代理与静态站点相关配置 |
| `config/domains.json` | 域名到前后端服务的映射关系 |
| `config/trojan.json` | Trojan 实验模块配置入口 |

也可以直接使用 `config/config.hamburger` 作为主配置入口，并保持子配置文件继续使用 `json/toml/hamburger`。

## 主配置关键字段

```json
{
  "pxy_frontend_file": "config/frontend.json",
  "pxy_backend_file": "config/backend.json",
  "servers": {
    "gateway": {
      "enable_http3": true,
      "enable_fast_proxy": true
    }
  },
  "middleware": {
    "gateway": {
      "prehandlers": {
        "pre_auth": {
          "enabled": true,
          "mode": "jwt"
        }
      }
    }
  }
}
```

## 配置组织建议

- 按职责拆分：入口配置、前端配置、域名映射、实验配置分离管理
- 按环境管理：为开发、测试、生产维护独立配置副本
- 先校验后启动：上线前使用 `test` 子命令验证配置有效性
- 保持一致性：域名映射与服务端口配置同步更新

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

## 自定义服务

`domains.json` 支持配置 `service_type: "custom"` 的自定义服务，用于在同一个服务下组合 API 转发与静态文件代理能力。

参考 `config/domains.json` 中 `BlogNext`（类型为 `custom`）配置：

```json
{
  "service_name": "BlogNext",
  "service_type": "custom",
  "host": "127.0.0.1",
  "port": 3000,
  "proxy_pass": [
    {
      "api": "/api",
      "service": "Blog",
      "use_rewrite": false
    },
    {
      "api": "/images",
      "static_direct": {
        "static_root": "/renj.io/app/Blog/images"
      }
    }
  ]
}
```

### 核心行为

- `proxy_pass`：按 `api` 前缀匹配请求并转发；默认转发到 `service` 对应服务的端口
- `static_direct`：当该配置不为空时，表示静态文件代理，行为类似 nginx 的 `alias`
- 路径映射：会将 `api` 后的请求路径拼接到 `static_root`，并转发到本地路径

例如：

- 请求 `/images/photo.jpg`
- 若 `api` 为 `/images`，`static_root` 为 `/renj.io/app/Blog/images`
- 实际读取本地路径 `/renj.io/app/Blog/images/photo.jpg`
