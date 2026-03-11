# 快速开始

## 1. 环境准备

- Go 环境：用于构建 Hamburger 主程序
- Node.js 18+：用于本目录 VitePress 文档站点

## 2. 启动 Hamburger

```bash
hamburger run -c config/config.json
```

可选命令：

```bash
hamburger test -c config/config.json
hamburger reload -c config/config.json
```

## 3. 启动文档站点

在 `docs` 目录执行：

```bash
npm install
npm run docs:dev
```

默认访问地址：

```text
http://localhost:5173
```

## 4. 生产构建

```bash
npm run docs:build
npm run docs:preview
```

## 5. 常见排查

- 端口冲突：调整 VitePress 或网关监听端口
- 启动失败：优先执行 `hamburger test -c config/config.json`
- 证书问题：确认 TLS 文件路径与权限配置正确
