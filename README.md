# Hamburger

<p align="center">
  <img src="./docs/logo-hamburger.jpg" alt="Hamburger Logo" width="220" />
</p>

Hamburger 是一个基于 Go 构建的网关与代理编排服务项目，集成了前后端代理、网关能力以及多种实验性服务模块。

## 官网

- http://hamburger.renj.io

## 项目结构

- `app/`：应用启动入口与生命周期管理
- `gateway/`：网关核心能力与中间件
- `frontend_proxy/`：前端代理服务
- `backend_proxy/`：后端代理服务
- `exp/`：实验功能模块（如 VPN、DNS、WebDAV、Trojan 等）
- `internal/config/`：配置模型与配置加载逻辑
- `dashboard/`：管理面板前端
- `docs/`：项目文档站点内容

## 快速说明

- 主配置文件位于 `config/config.json` 与 `config/config.hamburger`
- 可按需开启 `exp_config` 下的实验服务
- 建议先阅读 `docs/` 下的文档了解详细配置项
