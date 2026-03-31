# Hamburger Skills Index

> 本目录包含 Hamburger API 网关的 AI Agent 使用经验与开发指南

## Available Skills

### 1. [配置管理](skills/config-management.md)
- `.hamburger` DSL 配置语法与变量系统
- `config.json` 标准 JSON 配置
- 子配置文件拆分（backend/frontend/domains）
- 配置验证与生成

### 2. [路由与代理](skills/routing-proxy.md)
- 域名到前后端映射规则
- 前端静态文件服务配置
- 后端 API 代理与重写
- 负载均衡与后端连接池

### 3. [中间件与安全](skills/middleware-security.md)
- CORS 跨域配置
- Gzip 压缩优化
- 流量控制（限流规则）
- Trace 追踪与安全头
- IP 黑白名单

### 4. [部署与运维](skills/deployment-ops.md)
- CLI 命令使用（run/test/reload/generate）
- 进程管理与 PID 文件
- 日志配置与输出
- 多服务器生命周期管理

### 5. [插件开发](skills/plugin-development.md)
- WASM 插件
- WASI 运行时与内存管理
- 请求/响应处理钩子
- 插件配置与生命周期

### 6. [API 管理](skills/api-management.md)
- 管理 API 端点列表
- JWT 认证
- 服务控制（启动/停止/重启）
- 统计与健康检查

### 7. [故障排查](skills/troubleshooting.md)
- 配置错误诊断
- 连接问题排查
- 代理失败分析
- 性能瓶颈定位

### 8. [性能优化](skills/performance-tuning.md)
- HTTP/2 与 HTTP/3 调优
- 连接池参数优化
- 缓存策略配置
- nbio/fasthttp 底层调优

## Quick Reference

### 启动命令
```bash
# 运行网关
./hamburger run -c config/config.hamburger

# 测试配置
./hamburger test -c config/config.hamburger

# 重载配置
./hamburger reload

# 生成配置模板
./hamburger generate
```

### 关键端口
| 端口 | 用途 |
|------|------|
| 80 | HTTP 网关 |
| 443 | HTTPS 网关 |
| 7777 | 前端代理 |
| 8888 | 管理 API |
| 8889 | pprof (可选) |

### 配置文件层级
```
config/config.hamburger (主配置)
├── pxy_backend_file → backend.hamburger (后端代理)
├── pxy_frontend_file → frontend.hamburger (前端代理)
└── domain_map → domains.hamburger (域名映射)
```
