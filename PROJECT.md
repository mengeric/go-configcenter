# go-configcenter 项目宪章 (Project Charter)

> 本文档为 `go-configcenter` 的工程宪章，定义系统边界、分层依赖规则、技术栈选型与交付策略。

---

## 1. 项目概述 (Metadata)

| 项 | 值 |
|---|---|
| 项目名称 | go-configcenter |
| 模块路径 | `github.com/mengeric/go-configcenter` |
| Go 版本 | Go 1.24+ |
| 核心定位 | 统一微服务配置合并与服务注册发现 SDK |
| 核心依赖 | `nacos-sdk-go/v2`, `knadh/koanf/v2`, `yaml.v2` |
| 主分支策略 | `main` 稳定分支 |

---

## 2. 架构设计与依赖规则 (Architecture & Boundaries)

```text
[SDK 统一外壳 (sdk.go)]
       │
       ├──> [配置合并核心 (configmerge/)]
       ├──> [网络与 IP 嗅探 (internal/netutil/)]
       └──> [驱动适配层 (adapter/)]
                 ├──> [nacos 适配驱动 (adapter/nacos/)]
                 └──> [local 本地驱动 (local/)]
```

### 单向分层约束
1. **上层调用下层**：`sdk` 包可调用 `adapter`、`configmerge` 与 `netutil`；
2. **底层驱动独立**：`adapter/nacos` 与 `local` 严禁反向依赖 `sdk`；
3. **零框架污染**：`go-configcenter` 核心包严禁直接 import `go-zero` 框架包，通过实现其接口协议（`subscriber.Subscriber`）进行无依赖兼容。

---

## 3. 现行工程目录拓扑 (Directory Topology)

```text
go-configcenter/
├── Makefile                               # CNCF 标准构建与自解释执行入口
├── README.md                              # 门面：快速上手与使用示例
├── PROJECT.md                             # 宪章：系统边界、元数据与架构分层
├── AGENTS.md                              # 指南：AI 编码准则与目录指引
├── sdk.go                                 # SDK 统一 Facade 入口
├── config.go                              # 全局配置模型定义 (ArgoConfig)
├── options.go                             # 配置项与扩展
├── adapter/                               # 注册中心与配置中心驱动抽象
│   ├── interface.go                       # NamingClient / ConfigClient 接口
│   ├── instance.go                        # 服务实例统一实体
│   └── nacos/                             # Nacos v2 驱动实现
├── configmerge/                           # 多源配置深度合并核心引擎
├── local/                                 # 纯本地 Fallback 驱动实现
├── subscriber/                            # 兼容 go-zero 的动态配置订阅器
├── internal/
│   └── netutil/                           # 内部网络与智能注册 IP 探测
├── docs/                                  # 行业规范文档库
│   ├── specs/                             # 业务与技术规格 (SSOT)
│   ├── adr/                               # 架构决策记录
│   ├── plans/                             # 实施计划永久归档库
│   └── knowledge/pitfalls/                # 实战避坑经验
├── tests/
│   └── adversarial/                       # Goodhart 边界抗性测试
└── examples/                              # Gin 与 go-zero 完整接入示例
```

---

## 4. 交付与测试策略
- 修改任何核心逻辑前，必须在 `docs/plans/` 归档实施计划；
- 代码提交前必须执行 `make test` 及 `make test-adversarial` 确保全绿通过。
