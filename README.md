# go-configcenter

Go 服务统一接入 SDK，支持 Nacos 模式和纯本地模式。统一多源配置分层合并（Remote + Local）与服务注册发现，内置 Docker / 容器环境注册 IP 智能推导。

> 📖 **项目工程宪章与架构治理**：请参考 [PROJECT.md](PROJECT.md)（含分层依赖规则、技术选型与目录拓扑）与 [AGENTS.md](AGENTS.md)（智能体研发规范）。

---

## 功能特性

- **服务注册与注销**：支持动态权重、健康检查及 Docker/容器环境 IP 自动探测
- **服务发现**：Nacos 动态加权轮询订阅 / 本地静态地址表，支持 HTTP / gRPC
- **配置管理**：本地多文件 merge + Nacos 远程配置深度覆盖
- **go-zero 接口兼容**：实现标准 `Subscriber`，框架零代码侵入
- **纯本地模式**：无 Nacos 依赖时平滑回退，单机与 CI 友好
- **智能 IP 探测**：优先环境变量 (`HOST_IP`/`POD_IP`)，支持 Nacos UDP 路由嗅探与物理网卡过滤保底

---

## 文档索引 (`/docs`)

- [系统架构与处理规范](docs/specs/architecture.md)
- [SDK 接入与 API 规范](docs/specs/sdk-api.md)
- [架构决策记录 (ADR)](docs/adr/001-configcenter-adapter-architecture.md)
- [实施计划永久归档](docs/plans/)
- [实战避坑指南](docs/knowledge/pitfalls/network-and-config-pitfalls.md)

---

## 快速开始

```go
// 1. 初始化 SDK
s := goconfigcenter.MustInit("etc/global.yaml", "rag-service")

// 2. 服务注册（支持自动探测真实 IP，传入 "0.0.0.0" 或 "" 自动解析）
if err := s.Register("0.0.0.0", 8080); err != nil {
    panic(err)
}
defer s.Deregister()

// 或通过配置快捷注册：
// s.QuickRegister()

// 3. 用 go-zero configcenter 加载合并配置
cc := configurator.MustNewConfigCenter[Config](configurator.Config{
    Type: s.ConfigType(),
}, s.Subscriber())
c, err := cc.GetConfig()

// 4. 服务发现（HTTP 用，返回 http://ip:port）
addr, err := s.Discover("phoenix")
// addr = "http://192.168.1.100:10011"

// 5. 服务发现（gRPC 用，返回 ip:port）
addr, err := s.DiscoverRaw("wormhole")
// addr = "192.168.1.100:8885"
```

---

## 配置文件规范

### 合并顺序（后者覆盖前者）

1. **远程配置**（Nacos） — 优先级低，作为基底
2. **本地配置**（Local Files） — 优先级高，逐层覆盖远程

```yaml
registry:
  type: "nacos"
  addr: "192.168.1.100:8848"
  scheme: "http"
  namespace: "public"
  group: "DEFAULT_GROUP"

services:
  rag-service:
    local:
      - "etc/base.yaml"
      - "etc/rag-service.yaml"
    remote:
      - dataId: "shared.yaml"
        group: "SHARED"
      - dataId: "rag.yaml"
        group: "APP"
```

---

## SDK 方法速查

| 方法 | 说明 |
|---|---|
| `MustInit(file, serviceName)` | 初始化 SDK（失败直接 panic） |
| `Init(file, serviceName)` | 初始化 SDK |
| `Register(ip, port)` | 注册服务（传空或 `0.0.0.0` 自动探测出网 IP） |
| `QuickRegister()` | 快速注册（从全局配置 host/port 读取，支持自动 IP 探测） |
| `Deregister()` | 从注册中心注销服务 |
| `Discover(name)` | 动态服务发现（带 scheme，如 `http://ip:port`，自动订阅与加权轮询） |
| `DiscoverRaw(name)` | 动态服务发现（纯 `ip:port`，gRPC 适用，自动订阅） |
| `DiscoverAll(name)` | 发现目标服务所有可用健康实例列表 |
| `Subscribe(name, callback)` | 订阅目标服务实例动态变更（带初始快照回放） |
| `Unsubscribe(name)` | 取消指定服务的实例变更订阅 |
| `Subscriber()` | 获取 go-zero 兼容的动态配置订阅器 |
| `Config()` | 获取全局配置实例 |
| `ServiceName()` | 获取当前服务名称 |

---

## 构建与测试

统一使用根目录 `Makefile` 进行标准化操作：

```bash
make help             # 查看所有支持的 make 命令
make test             # 运行全量单元测试
make test-adversarial # 运行对抗性与边界测试
make fmt tidy         # 格式化与依赖整理
```
