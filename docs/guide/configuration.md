# 配置指南

Hamburger 使用结构化配置文件管理运行参数，推荐以主配置为入口，再按业务拆分子配置文件。

## 配置文件结构

| 文件 | 作用 |
| --- | --- |
| `config/config.json` | 主配置入口，定义服务端口、中间件、数据库、实验模块等 |
| `config/frontend.json` | 前端代理与静态站点相关配置 |
| `config/domains.json` | 域名到前后端服务的映射关系 |
| `config/trojan.json` | Trojan 实验模块配置入口 |

## 主配置关键字段

```json
{
  "pxy_frontend_file": "config/frontend.json",
  "pxy_backend_file": "config/backend.json",
  "servers": {
    "gateway": {
      "enable_http3": true,
      "enable_fast_proxy": true
    }
  },
  "middleware": {
    "gateway": {
      "prehandlers": {
        "pre_auth": {
          "enabled": true,
          "mode": "jwt"
        }
      }
    }
  }
}
```

## 配置组织建议

- 按职责拆分：入口配置、前端配置、域名映射、实验配置分离管理
- 按环境管理：为开发、测试、生产维护独立配置副本
- 先校验后启动：上线前使用 `test` 子命令验证配置有效性
- 保持一致性：域名映射与服务端口配置同步更新

## 注意事项

- `pxy_backend_file` 指向的文件应确保存在并可解析
- 启用 PreAuth 时应同步配置认证模式和凭证来源
- 开启 HTTP3 时需正确准备证书与监听端口策略
