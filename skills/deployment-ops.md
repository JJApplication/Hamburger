# Skill: 部署与运维

## 概述
Hamburger 提供完整的 CLI 命令集用于部署、运维和日常管理。

## CLI 命令

### 启动网关
```bash
# 默认配置启动
./hamburger

# 指定配置文件
./hamburger run -c config/config.hamburger
```

### 测试配置
```bash
# 验证配置文件
./hamburger test -c config/config.hamburger
```
输出 `config ok` 表示配置正确。

### 重载配置
```bash
# 优雅重启（发送 SIGTERM 到当前进程，然后启动新进程）
./hamburger reload
```
重载流程：
1. 读取 `hamburger.pid` 获取进程 ID
2. 发送 `SIGTERM` 信号
3. 等待进程退出（最多 30 秒）
4. 启动新进程

### 生成配置
```bash
# 交互式生成配置模板
./hamburger generate
```

## 进程管理

### PID 文件
- 默认位置：`hamburger.pid`
- 启动时创建，关闭时删除
- 重载时用于定位进程

### 信号处理
| 信号 | 行为 |
|------|------|
| SIGTERM | 优雅关闭所有服务 |
| SIGINT | 同 SIGTERM |

### 优雅关闭流程
```
SIGTERM → 停止接收新请求
        → 等待现有请求完成
        → 关闭所有服务器
        → 删除 PID 文件
        → 退出进程
```

## 服务器组件

Hamburger 启动时管理多个服务器：

| 组件 | 端口 | 说明 |
|------|------|------|
| Frontend Proxy | 7777 | 静态文件服务 |
| Gateway Manager | 80/443 | 网关代理 |
| Backend Proxy | 动态 | 后端代理 |
| API Server | 8888 | 管理 API |
| Latency Server | 动态 | 延迟测试 |
| Static Direct | 动态 | 静态直连 |
| VPN Server | 动态 | VPN 服务 |
| AnyTLS Server | 动态 | AnyTLS 协议 |
| DNS Server | 动态 | DNS 服务 |
| WebDAV Server | 动态 | WebDAV 服务 |
| Trojan Server | 动态 | Trojan 代理 |
| gRPC Server | 动态 | gRPC 服务 |

## 日志配置

### 配置
```hamburger
log: {
  log_level: "error",    # debug | info | warn | error
  log_file: "",          # 空=stdout
  color: true
}
```

### 日志级别建议
| 级别 | 场景 |
|------|------|
| debug | 开发调试 |
| info | 测试环境 |
| error | 生产环境（推荐） |
| warn | 需要更多信息时 |

### 日志输出
- 默认输出到标准输出
- 设置 `log_file` 输出到文件
- 使用 `color: true` 启用彩色输出

## 调试与性能分析

### pprof
```hamburger
pprof: {
  enable: false,
  port: 8889
}
```

启用后访问：
```
http://localhost:8889/debug/pprof/
http://localhost:8889/debug/pprof/profile
http://localhost:8889/debug/pprof/heap
```

### 调试模式
```hamburger
debug: true    # 启用详细调试信息
```

### CPU 核心限制
```hamburger
max_cores: 4   # 限制使用的 CPU 核心数
```

## 定时任务

### 同步器配置
```hamburger
syncer: {
  job_sync_domains: 60,         # 域名同步间隔 (秒)
  job_sync_domain_ports: 60     # 域名端口同步间隔 (秒)
}
```

## 数据目录

Hamburger 运行时创建的数据文件：

```
data/
├── hamburger-api.db    # API 数据库 (bbolt)
├── stat.db             # 统计数据库
├── stat.json           # 统计缓存
├── geo.json            # 地理位置数据
└── domain.json         # 域名数据
```

## 部署检查清单

### 部署前
- [ ] 运行 `hamburger test` 验证配置
- [ ] 确认 SSL 证书有效
- [ ] 检查端口未被占用
- [ ] 确认数据目录有写权限

### 部署后
- [ ] 检查所有服务启动状态
- [ ] 验证域名路由正确
- [ ] 测试 HTTPS 证书
- [ ] 确认管理 API 可访问
- [ ] 检查日志无异常

### 更新配置
- [ ] 备份当前配置
- [ ] 运行 `hamburger test` 验证新配置
- [ ] 使用 `hamburger reload` 优雅重载

## 监控建议

1. **健康检查** - 定期调用 `/api/health`
2. **日志监控** - 关注 error 级别日志
3. **统计监控** - 使用 `/api/stat` 获取流量数据
4. **连接监控** - 使用 `/api/conn` 查看连接数
5. **性能分析** - 启用 pprof 进行深度分析

## 常见问题

### 端口被占用
```bash
# 检查端口占用
lsof -i :80
lsof -i :443
lsof -i :8888

# 修改配置中的端口或停止占用进程
```

### 配置重载失败
```bash
# 手动停止进程
kill $(cat hamburger.pid)

# 重新启动
./hamburger run -c config/config.hamburger
```

### 日志过多
```hamburger
# 降低日志级别
log: {
  log_level: "error"
}
```
