# 实验特性

本章节介绍 Hamburger 当前可用的 DNS 实验服务能力与配置方式，便于在网关进程内直接启用 DNS 转发与 DoH 服务。

## DNS 服务用途

- 在网关进程中启动 DNS 服务，接收 UDP/TCP DNS 请求并转发到上游 DNS
- 可选开启 DoH（DNS over HTTPS）入口，统一走 HTTPS 查询
- 适用于内网网关、自托管 DNS 转发、测试环境 DNS 聚合等场景

## 配置字段

`ExpConfig` 中的 `dns_server` 结构如下：

```go
type DNSServerConfig struct {
	Enabled  bool               `yaml:"enabled" json:"enabled"`
	Host     string             `yaml:"host" json:"host"`
	Port     int                `yaml:"port" json:"port"`
	Upstream string             `yaml:"upstream" json:"upstream"`
	Timeout  int                `yaml:"timeout" json:"timeout"`
	DOH      DNSServerDOHConfig `yaml:"doh" json:"doh"`
}
```

字段说明：

- `enabled`：是否启用 DNS 服务
- `host`：监听地址
- `port`：DNS 监听端口（UDP/TCP 同端口）
- `upstream`：上游 DNS 地址，格式如 `8.8.8.8:53`
- `timeout`：转发与 DoH 读写超时，单位秒
- `doh`：DoH 子配置，支持独立监听地址、路径与证书

## 配置示例

```hamburger
{
  exp_config: {
    dns_server: {
      enabled: true,
      host: "0.0.0.0",
      port: 53,
      upstream: "8.8.8.8:53",
      timeout: 5,
      doh: {
        enabled: true,
        host: "0.0.0.0",
        port: 8443,
        path: "/dns-query",
        cert_file: "certs/doh.crt",
        key_file: "certs/doh.key"
      }
    }
  }
}
```

## 运行行为

- 启用后会同时监听 DNS UDP/TCP
- DoH 开启时会在 `doh.path` 注册 HTTPS 查询端点
- DoH 证书为必填项，缺少 `cert_file` 或 `key_file` 会启动失败
- DNS 查询默认转发到 `upstream`，超时按 `timeout` 生效

## 默认值

当配置未填写时，DNS 模块使用以下默认值：

- `host`: `0.0.0.0`
- `port`: `53`
- `upstream`: `8.8.8.8:53`
- `timeout`: `5` 秒
- `doh.path`: `/dns-query`
