# ADR 001: 注册中心适配器与配置合并架构选型

## 状态
已接受 (Accepted) - 2025-05-18

## 背景
业务微服务技术栈涵盖原生 Gin 与 go-zero 框架。各服务原先存在独立的 Nacos 客户端代码与本地配置加载逻辑，重复度高且无法在无 Nacos 环境（单机开发、CI 测试）下平滑回退。需要提供轻量级统一 SDK，屏蔽底层差异。

## 决策
1. **接口防腐层设计**：
   在 `adapter/interface.go` 抽取 `NamingClient` 与 `ConfigClient` 接口，解耦具体注册中心驱动。
2. **双引擎实现**：
   - `adapter/nacos`：基于 `nacos-sdk-go/v2` 实现企业级注册与配置监听；
   - `local`：本地内存与文件驱动，单机模式无需运行 Nacos 服务端即可启动。
3. **分层合并（Hierarchical Merging）**：
   采用 `Remote 基底 + Local 文件覆盖` 策略，保障本地开发配置的优先权，避免线上 Nacos 配置误覆盖本地调试参数。
4. **go-zero 接口兼容**：
   通过 `subscriber.Subscriber` 实现 go-zero `configurator.MustNewConfigCenter` 接口，使得 go-zero 无缝支持热更新。

## 影响与收益
- 微服务应用可在本地零依赖开发（使用本地 yaml 地址表）；
- 统一了各微服务服务发现和配置加载代码，减少样板代码 80% 以上。
