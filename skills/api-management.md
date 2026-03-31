# Skill: API 管理

## 概述
Hamburger 提供基于 HTTP 的管理 API，用于监控、配置和服务控制。

## 管理 API 配置

```hamburger
api_server_config: {
  enabled: true,
  host: "127.0.0.1",    # 仅本地访问
  port: 8888,
  http2: {
    enabled: true,
    insecure: true       # 允许明文 HTTP/2
  },
  bblot: {
    enabled: true,
    file: "data/hamburger-api.db"
  }
}
```

## API 端点

### 公开端点 (无需认证)

#### 健康检查
```
GET /api/health
```
响应：
```json
{
  "status": "ok"
}
```

#### 统计信息
```
GET /api/stat
```
返回网关流量统计数据。

#### 地理位置
```
GET /api/geo
```
返回地理位置统计数据。

#### 域名统计
```
GET /api/domain
```
返回各域名请求统计。

#### 连接信息
```
GET /api/conn
```
返回当前连接数统计。

### 认证端点 (需要 JWT)

#### 登录
```
POST /api/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```
响应：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### 登出
```
POST /api/logout
Authorization: Bearer <token>
```

#### 用户管理

**获取用户**
```
GET /api/user
Authorization: Bearer <token>
```

**创建用户**
```
POST /api/user
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "newuser",
  "password": "password",
  "role": "admin"
}
```

**更新用户**
```
PUT /api/user
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "admin",
  "password": "newpassword"
}
```

**删除用户**
```
DELETE /api/user
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "olduser"
}
```

#### 服务控制

**启动服务**
```
POST /api/service/start
Authorization: Bearer <token>
Content-Type: application/json

{
  "service": "frontend"
}
```

**停止服务**
```
POST /api/service/stop
Authorization: Bearer <token>
Content-Type: application/json

{
  "service": "frontend"
}
```

#### 服务器控制

**重启服务器**
```
POST /api/server/restart
Authorization: Bearer <token>
Content-Type: application/json

{
  "server": "http-server"
}
```

**停止服务器**
```
POST /api/server/stop
Authorization: Bearer <token>
Content-Type: application/json

{
  "server": "http-server"
}
```

## JWT 认证

### 配置
```hamburger
api_server_config: {
  jwt: {
    secret: "your-secret-key",
    expire: 86400    # 24 小时
  }
}
```

### 使用
所有认证端点需要在请求头中包含 JWT Token：
```
Authorization: Bearer <token>
```

## 数据库

管理 API 使用 bbolt (嵌入式键值数据库) 存储：
- 用户信息
- 认证 Token
- 服务状态

数据库文件位置：`data/hamburger-api.db`

## 安全建议

1. **绑定本地** - 管理 API 始终绑定 `127.0.0.1`
2. **强密码** - 使用强密码保护管理账户
3. **定期轮换** - 定期更换 JWT Secret
4. **Token 过期** - 设置合理的 Token 过期时间
5. **HTTPS** - 生产环境使用反向代理提供 HTTPS

## 使用示例

### curl 示例
```bash
# 健康检查
curl http://127.0.0.1:8888/api/health

# 登录
curl -X POST http://127.0.0.1:8888/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# 获取统计 (使用 Token)
curl http://127.0.0.1:8888/api/stat \
  -H "Authorization: Bearer <token>"

# 重启服务
curl -X POST http://127.0.0.1:8888/api/server/restart \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"server":"http-server"}'
```
