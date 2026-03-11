---
layout: home

hero:
  name: Hamburger
  text: 现代化高性能网关与代理编排平台
  tagline: 支持多协议接入、预认证安全链路、可观测统计与插件扩展
  actions:
    - theme: brand
      text: 开始阅读
      link: /guide/overview
    - theme: alt
      text: 配置指南
      link: /guide/configuration

features:
  - icon: 🚀
    title: 多协议高性能入口
    details: 支持 HTTP/HTTPS/HTTP2/HTTP3 与 nbio/fasthttp 路径，满足高并发网关场景。
  - icon: 🛡️
    title: 前置安全治理
    details: 预认证、限流、域名校验、头清洗、熔断与防盗链，形成可组合防护链路。
  - icon: 📈
    title: 可观测与运维友好
    details: 内置统计与地理维度分析，支持健康探针、延迟服务、配置重载与运行时同步。
  - icon: 🔌
    title: 开放扩展能力
    details: 提供 WASM 请求响应钩子和 gRPC 管理面，便于平台集成与二次开发。
---

<HeroTerminal />

## 为什么选择 Hamburger

Hamburger 将网关转发、前后端代理、运行时同步与实验能力整合为统一服务编排入口，既能快速上线，也便于后续深度扩展。

- 面向生产的分层架构，启动链路可观测、可重载
- 规则驱动的域名与端口映射，适配多业务接入
- 预处理与响应修改器解耦，策略扩展成本低
- 配置结构化清晰，支持模块化拆分与合并
