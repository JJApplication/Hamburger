# Lua Scripts

默认情况下，Hamburger 会从当前工作目录下的 `lua_scripts` 目录加载所有 `.lua` 文件。

## 配置

可通过配置文件中的 `lua` 节点调整行为：

```json
{
  "lua": {
    "enabled": true,
    "scripts_root": "lua_scripts"
  }
}
```

## 全局对象

- `log`
  - 提供 `Info`、`Debug`、`Warn`、`Error`
  - 同时支持小写 `info`、`debug`、`warn`、`error`
- `hamburger`
  - `app_name`
  - `description`
  - `version`
  - `build_hash`
- `constant`
  - 注入内置常量与部分运行时变量
- `env`
  - 注入当前进程环境变量快照

## 请求中间件

如果脚本定义了全局函数 `request_handle(req)`，则会在内置前置中间件执行完成后被调用。

`req` 包含以下字段：

- `method`
- `host`
- `path`
- `raw_query`
- `url`
- `remote_addr`
- `header`

返回值为 `table` 或 `nil`。支持以下字段：

- `allow`
- `error`
- `host`
- `path`
- `raw_query`
- `header_set`
- `header_add`
- `header_del`

## 示例

```lua
function request_handle(req)
  log.Info("handle request: " .. req.host)

  if req.header["X-Block"] == "1" then
    return {
      allow = false,
      error = "blocked by lua",
    }
  end

  return {
    header_set = {
      ["X-Lua"] = "enabled",
    },
  }
end
```

## 场景示例

### 1. 使用全局日志输出请求方式、参数、host、path

```lua
function request_handle(req)
  log.Info("method=" .. req.method)
  log.Info("host=" .. req.host)
  log.Info("path=" .. req.path)
  log.Info("raw_query=" .. req.raw_query)
  return nil
end
```

### 2. 修改请求头（转发到上游前）

```lua
function request_handle(req)
  return {
    header_set = {
      ["X-Request-From"] = "hamburger-lua",
      ["X-Request-Path"] = req.path,
    },
  }
end
```

### 3. 根据请求头参数放行或拒绝请求

```lua
function request_handle(req)
  local token = req.header["X-Auth-Token"]
  if token == "allow-token" then
    return { allow = true }
  end
  return {
    allow = false,
    error = "unauthorized token",
  }
end
```

### 4. 添加请求头（保留原有值并追加）

```lua
function request_handle(req)
  return {
    header_add = {
      ["X-Trace-Tag"] = "lua-mid",
      ["X-Debug-Mode"] = "on",
    },
  }
end
```

### 5. 自定义返回 error

```lua
function request_handle(req)
  if req.path == "/admin" then
    return {
      allow = false,
      error = "access denied by lua rule: /admin is blocked",
    }
  end
  return nil
end
```

> 说明：当前 `request_handle` 运行在请求阶段，上述 `header_set` / `header_add` / `header_del` 作用于“转发到上游前的请求头”，不是直接修改下行响应头。
