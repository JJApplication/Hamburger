# Hamburger DSL 配置说明

`internal/dsl_conf` 提供了项目专属的配置解析能力，用于替代 JSON/TOML，当前在 `internal/config` 中通过 `.hamburger` 后缀接入。

## 目标

- 配置结构与原有 JSON 模型保持一致
- 减少模板化重复，支持环境变量、内置符号与表达式
- 保持可读性，支持注释

## 核心特性

### 1) 对象键名支持免引号

示例：

```hamburger
{
  host: "127.0.0.1",
  port: 8080
}
```

说明：

- 普通键可不加引号
- 当键名包含不受支持字符时请加引号，例如 `"/images"`、`"403"`

### 2) 注释支持

使用 `#` 开头表示单行注释：

```hamburger
# 网关监听
{
  port: 8080
}
```

### 3) 环境变量取值 `$VAR`

可从环境变量读取值并根据目标字段类型自动转换：

```hamburger
{
  port: $APP_PORT,
  enabled: $FEATURE_ENABLED
}
```

说明：

- 底层读取使用 `os.LookupEnv`
- 仅建议用于非复杂类型字段：`string / bool / int / uint / float`
- 若环境变量不存在，解析会返回错误
- 双引号字符串内支持环境变量拼接，示例：`"$ROOT/data"`、`"data/$ROOT"`
- 环境变量拼接表达式必须使用双引号包裹

### 4) 内置符号取值 `@NAME`

支持在配置中直接引用程序常量或动态值：

```hamburger
{
  custom_header: {
    Proxy-Server: @AppName,
    Proxy-Copyright: @Copyright
  },
  today: @DATE,
  now: @DATETIME
}
```

当前内置符号：

- `@AppName` -> `internal/constant` 中的 `AppName`
- `@Copyright` -> `internal/constant` 中的 `Copyright`
- `@DATE` -> 当前日期，格式 `2006-01-02`
- `@DATETIME` -> 当前日期时间，格式 `2006-01-02 15:04:05`
- `@ARCH` -> 当前编译目标架构（如 `amd64`、`arm64`）
- `@GOOS` -> 当前编译目标操作系统（如 `windows`、`linux`）
- `@GOVERSION` -> 当前 Go 版本（如 `go1.24.5`）
- `@NUMCORE` -> 当前操作系统可用 CPU 核心数
- `@KERNEL` -> 当前操作系统内核版本字符串

另外可通过 `WithSymbols(map[string]any)` 注册自定义符号。

### 5) 表达式计算

支持基础运算与位运算：

```hamburger
{
  size: 1 << 10,
  workers: (2 + 2) * 4,
  mask: 7 & 3
}
```

支持操作符：

- 算术：`+ - * / %`
- 位运算：`<< >> & | ^`
- 一元：`+ - ~`
- 括号优先级：`( ... )`

## 语法示例

```hamburger
# 主配置
{
  pxy_frontend_file: "config/frontend.hamburger",
  domain_map: "config/domains.hamburger",

  plugin: {
    enabled: false
  },

  api_server_config: {
    host: "127.0.0.1",
    port: $API_PORT
  },

  custom_header: {
    Proxy-Server: @AppName
  },

  max_cores: 1 << 2
}
```

## 在项目中的使用

`internal/config` 已支持 `.hamburger` 后缀：

- `LoadConfig`（主配置）
- `LoadBackendConfig`（后端配置）
- `LoadFrontConfig`（前端配置）

示例：

```go
cfg, err := config.LoadConfig("config/config.hamburger")
```

## 开发接口

核心入口：

```go
err := dsl_conf.Unmarshal(data, &target)
```

可选项：

```go
err := dsl_conf.Unmarshal(data, &target, dsl_conf.WithSymbols(map[string]any{
  "ENV_NAME": "dev",
}))
```

## 注意事项

- DSL 会按目标结构体 `json` tag 做字段匹配
- 未匹配字段会被忽略
- 类型不匹配会返回错误
- 表达式结果最终按目标字段类型做转换
