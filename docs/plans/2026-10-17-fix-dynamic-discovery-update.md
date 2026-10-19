# 修复动态服务发现自动更新与代码审计门闸实施计划

| 项 | 值 |
|---|---|
| 日期 | 2026-10-17 |
| 状态 | Completed |
| 任务类型 | 缺陷修复与工程审计门闸建立 |

## 1. 背景与根因剖析

业务在使用 `go-configcenter` 时反馈“动态发现不会更新”。经代码审计与深入推演，确认存在以下五个层面的根因与设计断层：

1. **Nacos 自动订阅缺失（Auto-Subscribe on Discover）**：
   - 在 `adapter/nacos/naming.go` 中，`Discover` 优先查本地缓存 `ns.cache`，若未命中直接调用底层 `SelectOneHealthyInstance`；
   - 首次发现时**未主动调用 Nacos `Subscribe` 发起实例监听**，且未将查询到的实例填充进缓存。导致本地缓存永远为空，加权轮询（WRR）永远无法命中缓存，服务端实例上下线完全无法动态推送到本地。
2. **SDK 入口 API 契约断层**：
   - `adapter.NamingClient` 接口定义了 `Subscribe`，但顶级 `sdk.go`（`SDK` 结构体）没有暴露任何针对服务实例的订阅和监听方法；
   - 外部调用者若想在服务实例列表变动时动态刷新内部长连接池（如 gRPC 连接池、HTTP client upstream），无从获取订阅能力。
3. **实例全下线/不健康时的数组越界与脏数据问题**：
   - 在 `adapter/nacos/naming.go` 的 `weightedSelect` 中，当可用实例列表为空或全部不健康时（例如服务全部下线），代码执行 `if len(healthy) == 0 { return instances[0] }`；
   - 若 `instances` 为空切片直接引发数组越界 panic；若全不健康则强行返回故障实例，未提供无可用健康实例的错误处理。
4. **多监听者回调管理与防空指针**：
   - `adapter/nacos/naming.go` 的回调处理中未判断 `callback != nil`；缺少多监听者广播机制。
5. **本地模式（LocalNamingService）缺少变更广播**：
   - `local/naming.go` 中的 `Subscribe` 仅在初次调用时同步回调一次，未保存回调函数；随后同一进程执行 `Register` 或 `Deregister` 时，未向订阅者广播实例变动。

---

## 2. 代码审计与质量门闸（Quality Gates）规约

为保证修改零妥协、高可追溯，本次任务建立并执行以下五道质量门闸（Gates）：

```text
[代码实现]
    │
    ▼
┌──────────────────────────────────────────────┐
│ Gate 1: 架构契约与依赖合规门闸                 │
│ 1. 严格遵守单向依赖流动（底层严禁依赖 sdk.go） │
│ 2. 零框架污染（严禁直接 require go-zero）    │
│ 3. 接口契约完备性（DiscoverAll, Subscribe）   │
└──────────────────────┬───────────────────────┘
                       │ PASS
                       ▼
┌──────────────────────────────────────────────┐
│ Gate 2: Clean Code 与复杂度门闸 (Audit)      │
│ 1. 普通方法 ≤ 50 行，编排方法 ≤ 80 行         │
│ 2. 主方法 Step-down 降序编排，私有方法下沉   │
│ 3. 专有名词全大写（IP, URL, HTTP 等）        │
│ 4. 导出标识符必须拥有完整标准块注释           │
└──────────────────────┬───────────────────────┘
                       │ PASS
                       ▼
┌──────────────────────────────────────────────┐
│ Gate 3: 并发与对抗防御门闸 (Adversarial)     │
│ 1. 实例全部下线/不健康场景绝对防御（零 Panic） │
│ 2. Discover 自动订阅的双重检查锁并发安全     │
│ 3. 多 Listener 广播防阻塞与防空指针安全      │
└──────────────────────┬───────────────────────┘
                       │ PASS
                       ▼
┌──────────────────────────────────────────────┐
│ Gate 4: 编译与测试全绿门闸 (CI Green)        │
│ 1. `go vet ./...` 零告警                      │
│ 2. `make test` 100% 通过                      │
│ 3. `make test-adversarial` 100% 通过          │
└──────────────────────┬───────────────────────┘
                       │ PASS
                       ▼
┌──────────────────────────────────────────────┐
│ Gate 5: 实施状态回填与交付评审               │
│ 1. 更新本计划书状态为 Completed               │
│ 2. 交付前自检清单逐项核销                     │
└──────────────────────────────────────────────┘
```

---

## 3. 变更范围

- 修改 `adapter/interface.go`: `NamingClient` 接口补充 `DiscoverAll` 方法
- 修改 `adapter/interface_test.go`: 补充 `MockNamingClient.DiscoverAll` 与合规测试
- 修改 `adapter/nacos/naming.go`: 实现 `Discover` 自动订阅、多 listener 管理、防 panic 加权轮询、`DiscoverAll`
- 修改 `local/naming.go`: 实现本地模式动态事件总线、`DiscoverAll`
- 修改 `local/naming_test.go`: 补充本地模式 Register/Deregister 动态通知测试、`DiscoverAll` 测试
- 修改 `sdk.go`: 暴露 `Subscribe`、`Unsubscribe` 与 `DiscoverAll`
- 修改 `sdk_test.go`: 覆盖服务发现订阅、全量发现与注销全生命周期测试
- 修改 `tests/adversarial/adversarial_test.go`: 增加实例全下线、并发订阅、空回调、不健康实例等对抗性测试
- 新增 `tests/integration/discovery_integration_test.go`: 建立端到端动态注册、发现、订阅快照与注销全生命周期集成测试
- 修改 `Makefile`: 增加 `audit` 自动化审计检测门闸与 `test-integration` 目标

---

## 4. 实施步骤与门闸自检

- [x] **Step 1 [Gate 1]**: 调整 `adapter/interface.go` 与 `adapter/interface_test.go`，扩充接口契约并验证通过
- [x] **Step 2 [Gate 2 & Gate 3]**: 重构 `adapter/nacos/naming.go`（自动订阅、安全 WRR、多 listener 广播），遵循函数行数与 Step-down
- [x] **Step 3 [Gate 2 & Gate 3]**: 改造 `local/naming.go`（本地事件总线广播、DiscoverAll）
- [x] **Step 4 [Gate 1 & Gate 2]**: 在 `sdk.go` 暴露 `Subscribe`、`Unsubscribe` 与 `DiscoverAll`，编写完备标准注释
- [x] **Step 5 [Gate 3 & Gate 4]**: 编写单元测试与对抗性防御测试（覆盖动态更新、下线清空、多 listener、并发安全）
- [x] **Step 6 [Gate 4]**: 执行 `make audit`（或静态检查）、`make test` 与 `make test-adversarial` 确保全绿
- [x] **Step 7 [Gate 5]**: 回填计划书状态为 `Completed`，执行最终审查
