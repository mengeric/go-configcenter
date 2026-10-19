# SDK 接入与 API 规范指南 (SDK Integration Specification)

本文档定义 `go-configcenter` 的对外接入契约、配置结构与使用方法。

---

## 1. 全局配置文件结构契约 (`argo.yaml` / `global.yaml`)

```yaml
registry:
  type: "nacos"                # 注册中心类型: nacos | local
  addr: "192.168.1.100:8848"   # 注册中心服务地址 (ip:port)
  scheme: "http"               # Discover 返回 scheme，支持 http / grpc
  namespace: "public"          # 命名空间 ID
  group: "DEFAULT_GROUP"       # 默认分组
  username: "${NACOS_USER:-}"  # 鉴权用户名，支持环境变量展开
  password: "${NACOS_PWD:-}"   # 鉴权密码，支持环境变量展开

services:
  phoenix:
    host: "0.0.0.0"            # 服务监听/注册 host (支持 0.0.0.0 或留空触发智能探测)
    port: 10011                # 服务端口
    weight: 10                 # 权重 (默认 10)
    local:                     # 本地配置文件列表（高优先级，按顺序覆盖）
      - "etc/phoenix.yaml"
    remote:                    # 远端配置列表（低优先级，作为基底）
      - dataId: "shared.yaml"
        group: "DEFAULT_GROUP"
```

---

## 2. Go SDK 核心 API 契约

### `MustInit(configPath, serviceName string) *SDK`
- **功能**：初始化 SDK，加载配置并初始化对应驱动；失败时直接 panic。
- **参数**：
  - `configPath`: 全局配置文件路径（如 `etc/argo.yaml`）
  - `serviceName`: 当前服务名称

### `Register(ip string, port uint64) error`
- **功能**：向注册中心注册当前服务实例。
- **自动 IP 探测**：当传入 `ip == ""` 或 `ip == "0.0.0.0"` 时，SDK 自动探测真实出口/容器物理 IP。

### `QuickRegister() error`
- **功能**：快捷注册，从配置文件的 `services.<name>.host` 和 `port` 中读取并注册。若配置为 `0.0.0.0` 则自动推导 IP。

### `Discover(serviceName string) (string, error)`
- **功能**：服务发现，返回带协议头的实例地址（例如 `http://192.168.1.50:10011`），内置加权轮询负载均衡。

### `DiscoverRaw(serviceName string) (string, error)`
- **功能**：服务发现，返回纯 `ip:port` 格式（适用于 gRPC 连接）。

### `DiscoverAll(serviceName string) ([]adapter.Instance, error)`
- **功能**：服务发现，返回目标服务当前所有可用的健康实例列表（含 IP、Port、Weight、Metadata 等）。

### `Subscribe(serviceName string, callback func([]adapter.Instance)) error`
- **功能**：订阅目标服务的实例动态变更通知。当服务实例上线、下线、权重变更或健康状态变化时自动执行回调，且在注册成功后立即用当前实例快照执行一次初始通知。
- **参数**：
  - `serviceName`: 目标服务名
  - `callback`: 实例切片变更回调函数

### `Unsubscribe(serviceName string) error`
- **功能**：取消目标服务的实例动态变更订阅，清理本地缓存与监听器，并注销 Nacos 底层订阅通道。
