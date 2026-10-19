# ADR 002: 动态服务发现生命周期与自动订阅缓存设计

## 1. 状态
提议中 (Proposed) - 2026-10-17

## 2. 背景与问题定义

在微服务拓扑中，上游服务通过 `s.Discover(serviceName)` 或 `s.DiscoverRaw(serviceName)` 发现下游实例。
当前代码库存在“动态发现不会更新”的系统性断层，具体表现在：

1. **自动订阅断层**：
   - `adapter/nacos/naming.go` 内部维护了 `cache map[string][]adapter.Instance`，但 `cache` 仅在 `Subscribe` 回调中写入。
   - `Discover` 从未自动触发目标服务的 `Subscribe`，导致 `cache` 永远为空，加权轮询（WRR）代码永远不命中，且无法建立 Nacos 服务端变更推送通道。
2. **API 暴露断层**：
   - `adapter.NamingClient` 接口定义了 `Subscribe`，但 `SDK` 结构体未对外透传，业务无法监听实例变更以维护长连接池。
3. **容错与防御断层**：
   - 当服务实例全部下线或均处于不健康状态时，`weightedSelect` 强行索引 `instances[0]` 导致切片越界 panic 或返回脏数据。
4. **本地模式闭环断层**：
   - `local/naming.go` 仅在 `Subscribe` 时同步触发一次初始回调，随后的 `Register` / `Deregister` 操作未能通知订阅者。

---

## 3. 架构设计决策

### 决策 1: 首次发现即自动订阅 (Lazy-initialized Auto-Subscription)
- 在 `Discover` / `DiscoverAll` 时，若目标服务未建立订阅，自动通过双重检查锁（DCL）调用 `ensureSubscribed(serviceName, group)`；
- 首次订阅时执行两阶段初始化：
  1. 同步拉取当前所有健康实例（`SelectInstances`），填充到本地读写锁缓存 `ns.cache`；
  2. 注册 Nacos 动态监听（`Subscribe`），当服务端发生实例变更（上线/下线/权重/健康度变化）时，异步刷新 `ns.cache` 并广播通知所有监听者。

### 决策 2: 读写分离本地缓存与安全加权轮询 (WRR)
- 本地维护 `sync.RWMutex` 保护的 `cache map[string][]adapter.Instance`；
- 加权轮询算法（WRR）严格过滤 `Healthy && Enable && Weight > 0` 的节点；
- 若无可用健康实例，返回明确的 `ErrNoHealthyInstance`，绝不发生 `panic`，亦绝不向业务层返回不可用节点。

### 决策 3: 补齐 SDK 统一接口契约 (Contract Completeness)
在统一入口 `sdk.go` 对外补齐并暴露标准方法：
- `Subscribe(serviceName string, callback func([]adapter.Instance)) error`
- `Unsubscribe(serviceName string) error`
- `DiscoverAll(serviceName string) ([]adapter.Instance, error)`

### 决策 4: Local 模式事件总线广播 (Local Event Dispatcher)
- `local.LocalNamingService` 内部维护 `listeners map[string][]func([]adapter.Instance)`；
- 每次 `Register` 或 `Deregister` 变更实例后，向对应服务的 listeners 广播最新实例状态，保证单机与集成测试环境行为与 Nacos 保持 100% 一致。

---

## 4. 时序与数据流架构

```text
[业务服务]                  [SDK / NamingService]                [Nacos Server]
    |                                 |                                |
    |----- 1. Discover("phoenix") --->|                                |
    |                                 |-- 检查是否已订阅 (subscribed?) |
    |                                 |   (若未订阅，启动自动订阅闭环)  |
    |                                 |                                |
    |                                 |-- 2. SelectInstances (健康) -->|
    |                                 |<- 返回初始实例列表 ------------|
    |                                 |-- 3. 写入本地 cache -----------|
    |                                 |                                |
    |                                 |-- 4. Subscribe (注册监听) ---->|
    |                                 |<-- 订阅建立成功 ---------------|
    |                                 |                                |
    |                                 |-- 5. 从 cache 执行加权轮询 ----|
    |<---- 返回所选健康实例 ----------|                                |
    |                                 |                                |
    |                                 |   [未来时刻：下游节点扩缩容]    |
    |                                 |<-- 6. Nacos Push 实例变更 -----|
    |                                 |-- 7. 自动刷新本地 cache -------|
    |                                 |-- 8. 触发业务已注册 listeners -|
    |                                 |                                |
    |----- 9. 下次 Discover() ------->|                                |
    |                                 |-- 从已刷新的 cache 加权选取 ---|
    |<---- 返回最新节点 --------------|                                |
```

---

## 5. 影响与风险评估
- **内存占用**：本地缓存仅保存服务发现目标实例的元数据（通常仅几十个节点），内存开销微乎其微（< 100KB）；
- **网络开销**：由于维护了订阅推送通道，后续 Discover 均走本地内存加权计算，相比每次网络查询 Nacos，网络延迟降低 99%，高并发 QPS 提升一个数量级；
- **优雅停机与注销**：当调用 `Unsubscribe` 时，彻底清理本地 listener、cache 及 Nacos 底层订阅通道，杜绝内存泄漏。
