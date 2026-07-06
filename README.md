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

### 合并顺序

配置按以下顺序合并（后者覆盖前者）：

1. **远程配置**（Nacos） — 优先级低
2. **本地配置** — 优先级高，覆盖远程

```yaml
services:
  rag-service:
    local:
      - "etc/base.yaml"       # 1. 最先加载
      - "etc/rag-service.yaml" # 2. 覆盖 base
    remote:
      - dataId: "shared.yaml"   # 3. 被 local 覆盖
        group: "SHARED"
      - dataId: "rag.yaml"      # 4. 被 local 覆盖
        group: "APP"
```

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
