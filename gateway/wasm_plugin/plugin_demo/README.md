# wasm 插件示例

## 作用
- 根据请求头 X-Allow-Method 判断是否允许请求方法
- 当请求方法不匹配时，拒绝请求并返回 403

## 编译
使用 TinyGo 编译为 WASI 模块：

```bash
tinygo build -o plugins/method_check.wasm -target=wasi ./gateway/wasm_plugin/plugin_demo
```

## 配置示例

```json
{
  "plugin": {
    "enabled": true,
    "root": "plugins",
    "plugins": [
      {
        "name": "method_check",
        "enabled": true,
        "params": {
          "note": "demo"
        }
      }
    ]
  }
}
```

## 使用方式
- 将编译产物放到插件目录：plugins/method_check.wasm
- 请求时添加头部：X-Allow-Method: POST
- 当请求方法不是 POST 时会被拒绝
