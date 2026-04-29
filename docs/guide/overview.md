# 功能总览

Hamburger 是一个以 Go 构建的网关与代理编排系统，核心目标是用统一入口承载多种网络流量处理能力，并保持高性能与可扩展性。

## 协议支持

- 标准支持：TCP，HTTP，HTTP2，HTTP3，Websocket，GRPC
- 实验性支持：AnyTLS，Trojan

## 架构分层

| 层级 | 模块职责 | 代表目录 |
| --- | --- | --- |
| 应用编排层 | 统一加载配置、初始化组件、控制生命周期 | `app/` `initialize/` |
| 网关转发层 | 请求接入、路由解析、转发与响应处理 | `gateway/core/` `gateway/server/` |
| 治理策略层 | 预处理器与响应修改器链路 | `gateway/prehandler/` `gateway/modifier/` |
| 运行时数据层 | 域名映射、端口映射、定时同步 | `gateway/runtime/` |
| 可观测与管理层 | 统计、健康探针、延迟服务、gRPC 管理面 | `gateway/stat/` `grpc_server/` |
| 扩展与实验层 | WASM 插件、VPN、AnyTLS、Trojan | `gateway/wasm_plugin/` `exp/` |

## 请求处理主路径

```text
客户端请求
  -> 网关入口(server)
  -> 预处理链(prehandler: 认证/限流/域名校验/头清理)
  -> 解析与路由(resolver + runtime)
  -> 代理转发(core/proxy)
  -> 响应修改链(modifier: 安全头/cors/gzip/trace)
  -> 返回客户端
```

## 对接JJApps

当前对接配置已统一为 `config/domains.json`，通过 `domain_service + services` 两部分描述域名与服务能力映射。

- `domain_service`：定义域名对应的服务名
- `services`：定义服务类型与具体配置
- `service_type`：支持 `frontend`、`backend`、`custom`

配置示例：

```json
{
  "domain_service": [
    {
      "domain": "blog.renj.io",
      "service": "BlogNext"
    }
  ],
  "services": [
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
    },
    {
      "service_name": "Blog",
      "service_type": "backend"
    },
    {
      "service_name": "Resume",
      "service_type": "frontend"
    }
  ]
}
```

这样可以在一份配置里同时管理前端服务、后端服务和自定义服务转发逻辑。
