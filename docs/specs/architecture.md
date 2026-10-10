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
