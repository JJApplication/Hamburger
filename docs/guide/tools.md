# 扩展工具

`cmd/` 目录提供了一组独立工具，用于配置转换、配置巡检、管理面调用与实验服务单独启动。

## 工具清单

| 工具目录 | 主要用途 | 常见场景 |
| --- | --- | --- |
| `cmd/hamburger_convert` | 配置格式转换（json/yaml/toml/hamburger） | 迁移配置、统一格式 |
| `cmd/hamburger_test` | 加载并校验主配置，输出关键配置表格 | 发布前校验 |
| `cmd/hamburger_cli` | gRPC 管理面命令行工具 | 查询状态、重载配置、重启组件 |
| `cmd/hamburger_client` | API 交互式客户端 | 运维调试、服务控制 |
| `cmd/hamburger_xxx` | 独立 Trojan 启动入口 | 单模块实验运行 |

## hamburger_convert

用于在多种配置格式之间互转，支持文件输出或直接打印到标准输出。

```bash
go run ./cmd/hamburger_convert -in config/config.hamburger -out config/config.json
go run ./cmd/hamburger_convert -in config/config.json -to hamburger
```

可选格式：

- `json`
- `yaml`
- `toml`
- `hamburger`

## hamburger_test

用于验证主配置与子配置可加载性，并打印前端、后端、网关、实验配置摘要。

```bash
go run ./cmd/hamburger_test -c config/config.hamburger
```

适合在发布前或变更后快速检查配置正确性。

## hamburger_cli

面向 gRPC 管理面的命令行工具，支持状态查询与控制动作。

```bash
go run ./cmd/hamburger_cli status runtime
go run ./cmd/hamburger_cli control reload-config --file config/config.hamburger
go run ./cmd/hamburger_cli control restart gateway
```

常用全局参数：

- `--type`：连接类型，`tcp` 或 `uds`
- `--addr`：管理面地址，默认 `127.0.0.1:8081`
- `--timeout`：请求超时
- `--insecure`：是否使用非 TLS 连接

## hamburger_client

交互式 API 客户端，启动后可执行 `health`、`stat`、`service`、`server` 等命令。

```bash
go run ./cmd/hamburger_client -host 127.0.0.1 -port 8888
```

进入交互后可使用：

- `help` 查看命令
- `service start <domain>` / `service stop <domain>`
- `server restart <gateway|front|api|backend|latency|vpn|trojan|anytls>`

## hamburger_xxx

用于按配置单独启动 Trojan 服务。

```bash
go run ./cmd/hamburger_xxx -c config/trojan.json
```

仅在需要将 Trojan 独立运行或调试时使用。
