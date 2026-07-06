# go-configcenter

Go 服务统一接入 SDK，支持 Nacos 模式和纯本地模式。

## 功能

- 服务注册与注销
- 服务发现（Nacos 动态订阅 / 本地地址表）
- 配置管理（本地文件 merge + Nacos 远程配置）
- go-zero 接口兼容（Subscriber）
- 无 Nacos 时纯本地运行
- 零依赖 go-zero（go-zero 集成在服务侧完成）

## 快速开始

```go
// 初始化 SDK
s := goconfigcenter.MustInit("etc/global.yaml", "rag-service")

// 注册服务
if err := s.Register("0.0.0.0", 8080); err != nil {
    panic(err)
}
defer s.Deregister()

// 用 go-zero configcenter 加载配置
cc := configurator.MustNewConfigCenter[Config](configurator.Config{
    Type: s.ConfigType(),
}, s.Subscriber())
c, err := cc.GetConfig()

// 发现其他服务
addr, err := s.Discover("phoenix")
```

## 配置文件

### Nacos 模式

```yaml
registry:
  type: "nacos"
  addr: "192.168.110.164:8848"
  namespace: "public"
  group: "DEFAULT_GROUP"

services:
  rag-service:
    local:
      - "etc/rag-service.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
  phoenix:
    local:
      - "etc/phoenix.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
```

### 本地模式（无 Nacos）

```yaml
# 不配 registry 或 registry.addr 为空
services:
  rag-service:
    host: 192.168.110.164
    port: 8080
    local:
      - "etc/rag-service.yaml"
  phoenix:
    host: 192.168.110.164
    port: 10011
    local:
      - "etc/phoenix.yaml"
```

## SDK 方法

| 方法 | 说明 |
|------|------|
| `MustInit(file, serviceName)` | 初始化 |
| `Register(ip, port)` | 注册服务 |
| `Deregister()` | 注销服务 |
| `Subscriber()` | 返回 go-zero Subscriber |
| `Discover(name)` | 服务发现（返回 ip:port） |
| `ConfigType()` | 自动判断配置类型 |

## 依赖

- nacos-sdk-go/v2
- koanf/v2
