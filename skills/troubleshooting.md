# Skill: 故障排查

## 概述
本文档提供 Hamburger 网关常见问题的诊断和解决方法。

## 配置问题

### 配置加载失败

**症状**
```
config load failed: ...
```

**排查步骤**
1. 运行配置验证
   ```bash
   ./hamburger test -c config/config.hamburger
   ```
2. 检查 JSON 语法（如使用 `.json` 格式）
3. 检查 DSL 语法（如使用 `.hamburger` 格式）
4. 确认子配置文件路径正确

**常见原因**
- JSON 缺少逗号或括号不匹配
- 子配置文件路径错误
- 变量未定义（`@Variable`）

### 端口绑定失败

**症状**
```
listen tcp 0.0.0.0:80: bind: address already in use
```

**排查步骤**
```bash
# 检查端口占用
lsof -i :80
lsof -i :443
lsof -i :8888

# 查看是否有残留进程
ps aux | grep hamburger

# 清理 PID 文件
rm hamburger.pid
```

**解决方案**
- 停止占用端口的进程
- 或修改配置使用其他端口

## SSL/TLS 问题

### 证书加载失败

**症状**
```
tls: failed to find certificate PEM data
```

**排查步骤**
1. 确认证书文件存在
   ```bash
   ls -la config/ssl/fullchain.cer
   ls -la config/ssl/ssl.key
   ```
2. 验证证书格式
   ```bash
   openssl x509 -in config/ssl/fullchain.cer -text -noout
   openssl rsa -in config/ssl/ssl.key -check
   ```
3. 检查证书与域名匹配

### HTTPS 连接失败

**排查步骤**
```bash
# 测试 SSL 连接
openssl s_client -connect renj.io:443 -servername renj.io

# 检查 TLS 版本
curl -v --tlsv1.3 https://renj.io
```

**常见原因**
- 证书过期
- 证书与域名不匹配
- TLS 版本配置过高（客户端不支持）

## 代理问题

### 502 Bad Gateway

**症状**
客户端收到 502 错误

**排查步骤**
1. 检查后端服务是否运行
   ```bash
   curl http://127.0.0.1:<backend-port>/health
   ```
2. 检查后端配置
   - `backend.hamburger` 中的服务地址和端口
   - `domains.hamburger` 中的映射关系
3. 检查防火墙规则
4. 查看网关日志

### 404 Not Found

**排查步骤**
1. 确认域名在 `domains.hamburger` 中有映射
2. 确认前端/后端服务名匹配
3. 检查静态文件路径是否存在
4. 确认 `try_file` 配置（SPA 应用）

### 请求超时

**排查步骤**
1. 检查后端响应时间
2. 调整超时配置
   ```hamburger
   proxy: {
     idle_conn_timeout: 60
   }
   ```
3. 检查连接池配置
   ```hamburger
   proxy: {
     max_conns_per_host: 100
   }
   ```

## 性能问题

### 高 CPU 使用率

**排查步骤**
1. 启用 pprof 分析
   ```hamburger
   pprof: {
     enable: true,
     port: 8889
   }
   ```
2. 查看 CPU profile
   ```bash
   go tool pprof http://localhost:8889/debug/pprof/profile
   ```
3. 检查 Gzip 压缩级别（过高会消耗 CPU）
4. 限制 CPU 核心
   ```hamburger
   max_cores: 4
   ```

### 内存泄漏

**排查步骤**
1. 查看内存 profile
   ```bash
   go tool pprof http://localhost:8889/debug/pprof/heap
   ```
2. 检查缓存配置
   ```hamburger
   proxy_cache: {
     cache_size: 2048    # 限制缓存条目数
   }
   ```
3. 检查连接池配置

### 连接数过高

**排查步骤**
```bash
# 查看当前连接
curl http://127.0.0.1:8888/api/conn

# 检查系统连接
netstat -an | grep :80 | wc -l
```

**解决方案**
- 调整 `max_conns_per_host`
- 启用限流规则
- 检查是否有连接泄漏

## 日志问题

### 日志过多

**解决方案**
```hamburger
log: {
  log_level: "error"    # 降低日志级别
}
```

### 无日志输出

**排查步骤**
1. 检查日志级别设置
2. 确认 `log_file` 路径可写
3. 检查文件权限

## 插件问题

### WASM 插件加载失败

**症状**
```
load wasm failed: ...
```

**排查步骤**
1. 确认 WASM 文件存在
   ```bash
   ls -la plugins/*.wasm
   ```
2. 验证 WASM 格式
   ```bash
   wasm-validate plugins/my-plugin.wasm
   ```
3. 检查导出函数是否存在
   ```bash
   wasm2wat plugins/my-plugin.wasm | grep "(export"
   ```

### 插件执行错误

**排查步骤**
1. 查看插件日志
2. 确认插件配置正确
3. 检查内存分配/释放

## 数据库问题

### MongoDB 连接失败

**症状**
```
mongo connection failed
```

**排查步骤**
```bash
# 测试 MongoDB 连接
mongosh mongodb://localhost:27017

# 检查 MongoDB 状态
systemctl status mongod
```

### 统计数据库损坏

**解决方案**
```bash
# 备份并重建统计数据库
cp data/stat.db data/stat.db.bak
rm data/stat.db
# 重启服务自动重建
```

## 紧急恢复

### 快速重启
```bash
# 停止服务
kill $(cat hamburger.pid)

# 清理
rm hamburger.pid

# 重新启动
./hamburger run -c config/config.hamburger
```

### 配置回滚
```bash
# 恢复备份配置
cp config/config.hamburger.bak config/config.hamburger

# 验证
./hamburger test -c config/config.hamburger

# 重载
./hamburger reload
```

## 诊断命令汇总

```bash
# 配置验证
./hamburger test -c config/config.hamburger

# 健康检查
curl http://127.0.0.1:8888/api/health

# 连接统计
curl http://127.0.0.1:8888/api/conn

# 流量统计
curl http://127.0.0.1:8888/api/stat

# 域名统计
curl http://127.0.0.1:8888/api/domain

# 性能分析
go tool pprof http://localhost:8889/debug/pprof/profile
```
