# 架构全景与处理规范 (Architecture Specification)

本文档是 `go-configcenter` 统一配置中心与服务注册发现 SDK 的系统架构全景规范（单一可信源 SSOT）。

---

## 1. 系统边界与核心职责

`go-configcenter` 为微服务提供无侵入的统一接入层，解耦服务代码与底层注册配置基础设施（Nacos / 本地文件）：

```text
+-------------------------------------------------------+
|                   业务服务 (Gin / go-zero)            |
+-------------------------------------------------------+
                           |
                           v
+-------------------------------------------------------+
|                    go-configcenter SDK                |
|  - SDK: 统一入口 (MustInit / Register / Discover)     |
|  - ConfigMerge: 多源配置分层覆盖 (Remote -> Local)    |
|  - Subscriber: go-zero configcenter 兼容订阅驱动      |
|  - NetUtil: 容器/Docker 注册 IP 智能推导              |
+-------------------------------------------------------+
               |                               |
       [Nacos 模式]                      [本地 Fallback 模式]
               v                               v
+-----------------------------+  +----------------------+
|       adapter/nacos         |  |        local         |
|  - Nacos NamingClient       |  |  - LocalNaming       |
|  - Nacos ConfigClient       |  |  - LocalConfig       |
+-----------------------------+  +----------------------+
```

---

## 2. 配置加载与分层合并时序

配置加载遵循**优先级覆盖**原则（后者覆盖前者）：

```text
1. 加载 argo.yaml / global.yaml，展开 ${ENV_VAR} 与 ${ENV_VAR:-default}
2. 从 Nacos 远程配置中心拉取 remote 节点配置（基底低优先级）
3. 按顺序读取本地 local 配置文件，深度 merge 覆盖远程配置（高优先级）
4. Subscriber 监听变更（Nacos 远程推送或文件变动），动态推送新版本配置
```

---

## 3. 服务注册与 IP 判定时序

在多容器（Docker Bridge/Host）及生产 Kubernetes 环境下，服务实例 IP 自动探测流程如下：

```text
Register(ip, port)
        |
        v
  ip 是否为空或 "0.0.0.0" ?
        |
   +----+----+
   | 是      | 否
   v         v
ResolveRegisterIP()  直接使用指定 IP
   |
   +---> 1. 检查环境变量 HOST_IP / POD_IP / REGISTER_IP
   |
   +---> 2. UDP 嗅探注册中心出网出口 IP (net.DialUDP -> targetAddr)
   |
   +---> 3. 枚举物理网卡并过滤 docker0/veth*/lo 等虚拟网卡
```

---

## 4. 关键领域接口定义

### 命名服务接口 (`adapter.NamingClient`)

```go
type NamingClient interface {
    Register(ip string, port uint64, serviceName, group string, weight float64) (bool, error)
    Deregister(ip string, port uint64, serviceName, group string) (bool, error)
    Discover(serviceName, group string) (string, error)
    DiscoverAll(serviceName, group string) ([]Instance, error)
    Subscribe(serviceName, group string, callback func([]Instance)) error
    Unsubscribe(serviceName, group string) error
}
```

### 配置服务接口 (`adapter.ConfigClient`)

```go
type ConfigClient interface {
    GetConfig(dataId, group string) (string, error)
    WatchConfig(dataId, group string, callback func(string)) error
}
```

---

## 5. 动态服务发现与自动订阅机制 (Dynamic Discovery Pipeline)

为避免业务启动后实例变更无法感知及无订阅导致的本地缓存失活，服务发现设计遵循**首次发现即自动订阅**原则：

```text
1. 业务调用 Discover(serviceName) / DiscoverAll(serviceName)；
2. SDK 检查该服务是否已订阅：
   - 若未订阅：触发 ensureSubscribed，同步获取健康实例初始化本地内存 cache，并向 Nacos 注册动态长连接订阅；
   - 若已订阅：直接跳过初始化；
3. Discover 从本地安全读写锁 cache 中执行加权轮询负载均衡 (WRR)；
4. 当远程实例发生变动（上线/下线/宕机/权重调整），Nacos 服务端主动推送：
   - 命名服务自动刷新本地 cache；
   - 广播通知所有已注册的业务监听者（listeners），实现长连接池/连接拓扑的平滑热更新；
5. 若全部实例不可用或下线，加权算法返回明确错误，绝不 panic，绝不返回脏数据。
```
