# go-configcenter 智能体研发规范与工程指引 (Agent Guidelines)

> 本文件是本工程的最高 AI 行为准则，所有进入本仓库的智能体与开发者必须严格遵守。

---

## 1. 🧭 工程门户与目录职责导航 (Directory Guidance)

在开展任何开发、重构或检索任务前，必须理解本仓库以下核心文件与目录的职责边界：

| 文件 / 目录 | 职责与定位 | 智能体操作指引 |
|---|---|---|
| `README.md` | 项目对外门面 (Front Door) | 查阅项目定位、快速接入示例与全局文档索引 |
| `PROJECT.md` | 项目工程宪章 (Charter) | **必读**。查阅架构元数据、依赖分层与目录拓扑 |
| `Makefile` | CNCF 标准可执行工程规格 | 每次代码修改后，必须运行 `make build`、`make test` 验证 |
| `docs/specs/` | 业务功能与技术规格 (SSOT) | 开发功能前必查，了解当前系统的权威接口契约与处理管道 |
| `docs/plans/` | 实施计划永久归档库 | 制定计划必须保存在此，与 Git Commit 强绑定，**严禁事后 `rm`** |
| `docs/adr/` | 架构决策记录 (ADR) | 查阅重大历史决策背景，严禁擅自逆转既定选型 |
| `docs/knowledge/pitfalls/` | 实战避坑与编码约束 | 查阅过往排错踩坑经验，避免重蹈覆辙 |
| `tests/adversarial/` | 对抗性与鲁棒性防御测试 | 交付前必须执行 `make test-adversarial` 确保全绿通过 |

---

## 2. 🚨 本工程核心架构硬性红线 (Hard Architectural Boundaries)

1. **分层依赖单向流动**：
   - 底层驱动 `adapter/` 与 `local/` 严禁反向依赖 `sdk.go`。
2. **零框架污染约束**：
   - 保持 SDK 纯粹性，禁止直接 `require github.com/zeromicro/go-zero`，统一通过协议适配层解耦。
3. **函数注释规范**：
   - 所有导出的结构体与函数必须包含完整的标准注释块（说明功能、所有参数、返回值及可能发生的 error）。
4. **IP 探测安全性**：
   - 注册 IP 推导必须规避 `docker0`、`veth*` 等容器桥接网卡，支持多阶梯 Fallback。

---

## 3. 🔄 研发三部曲与实施计划归档 (Plan First)

严格遵循：规划 (Plan) ──> 执行 (Implement) ──> 评审 (Review)

1. **计划必须归档于 `docs/plans/`**：
   - 命名格式：`docs/plans/YYYY-MM-DD-<type>-<task-name>.md`；
   - 头部记录元数据：`| 日期 | 状态 | 任务类型 |`；
   - **严禁使用一次性临时草稿或事后 `rm` 删除**。
2. **任务完成状态回填**：
   - 验证通过后，将计划书头部状态更新为 `Status: Completed`。
3. **版本提交双向索引**：
   - 每次任务对应的 Commit 记录中必须包含本计划的相对路径索引（例如 `Plan: docs/plans/...`）。
   - （注：Git 提交安全策略如严禁自动 push、提交风格解析等均遵循全局提示词规范约束）。
