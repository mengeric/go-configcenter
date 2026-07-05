# go-configcenter

Go 服务统一接入 SDK，支持 Nacos 模式和纯本地模式。

## 功能

- 服务注册与注销
- 服务发现（Nacos 动态订阅 / 本地地址表）
- 配置管理（本地文件 merge + Nacos 远程配置）
- go-zero 接口兼容（Subscriber / Resolver）
- 无 Nacos 时纯本地运行
- 零依赖 go-zero（go-zero 集成在服务侧完成）

## 快速开始

```go
s := goconfigcenter.MustInit("argo.yaml", "galaxy")

s.Register("0.0.0.0", 8888)
defer s.Deregister()

sub := s.Subscriber()
cc := configcenter.MustNewConfigCenter[Config](configcenter.Config{
    Type: s.ConfigType(),
}, sub)
c := cc.Load()

server := rest.MustNewServer(c.RestConf)
// ...
```

## 配置文件

```yaml
registry:
  type: "nacos"
  addr: "192.168.110.164:8848"
  namespace: "public"
  group: "DEFAULT_GROUP"

services:
  galaxy:
    local:
      - "etc/base.yaml"
      - "etc/galaxy.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
      - dataId: "galaxy.yaml"
        group: "APP"
  phoenix:
    local:
      - "etc/base.yaml"
      - "etc/phoenix.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
      - dataId: "phoenix.yaml"
        group: "APP"
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

## go-zero 集成

SDK 不依赖 go-zero，go-zero 集成在服务侧完成：

```go
// go-zero Resolver 封装
type resolver struct {
    sdk *goconfigcenter.SDK
}

func (r *resolver) Resolve(key string) (resolver.Endpoints, error) {
    addr, err := r.sdk.Discover(key)
    if err != nil {
        return nil, err
    }
    return resolver.Endpoints{{Addr: addr}}, nil
}

// 使用
zrpc.MustClient(c.RpcClientConf,
    zrpc.WithResolver(&resolver{sdk: s}),
)
```

## 依赖

- nacos-sdk-go/v2
- koanf/v2
