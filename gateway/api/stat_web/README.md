# Hamburger Stat Web

独立的 Vite + React + TypeScript + Tailwind 统计前端。它不修改也不复用根目录的 `dashboard/`，也不会由 Go 网关托管构建产物。

## 开发

```bash
npm install
npm run dev
```

默认访问 `http://localhost:5174`。开发环境默认请求同源 `/api/stat` 和 `/api/geo`；如果前端与 API 不同源，复制 `.env.example` 为 `.env.local` 并设置：

```dotenv
VITE_API_BASE_URL=https://gateway.example.com
```

API 基地址只填写 origin，不要附加 `/api`。接口需要允许前端来源访问，并正确配置 CORS。

## 仅查看前端（Mock）

不启动 Go 网关也可以直接查看完整页面。复制 `.env.example` 为 `.env.local`，将 Mock 开关打开：

```dotenv
VITE_API_BASE_URL=
VITE_USE_MOCK=true
```

然后重启开发服务器：

```bash
npm run dev
```

页面右上角会显示“模拟数据”。Mock 会覆盖所有时间窗口、请求/状态/延迟/资源/流量、域名详情和 GEO 数据；改回 `VITE_USE_MOCK=false` 后即可请求真实 API。环境变量在 Vite 启动时读取，修改后需要重启开发服务器。

## 检查与构建

```bash
npm run typecheck
npm run lint
npm run test:run
npm run build
npm run preview
```

`dist/` 是可独立部署的静态目录。将它发布到 Nginx、对象存储、CDN 或其他静态托管服务，并将 `VITE_API_BASE_URL` 在构建前注入。Go 服务不读取或托管该目录。

页面每 30 秒分别刷新 `/api/stat` 与累计 GEO `/api/geo`；时间窗口只影响 Stat 时序数据。浏览器启用减弱动画、WebGL 不可用或 WebGL 初始化失败时，全球来源模块降级为静态地图与 Top 10 排行。
