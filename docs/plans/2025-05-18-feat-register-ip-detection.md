# 注册 IP 自动探测与 Docker/容器环境适配实施计划

| 项 | 值 |
|---|---|
| 日期 | 2025-05-18 |
| 状态 | Completed |
| 任务类型 | 功能开发 |

## 背景
当前 SDK 在向 Nacos 等注册中心注册服务时（`Register` / `QuickRegister`），需要显式指定具体的 IP 地址。如果在 Docker 容器或混合部署环境中配置了 `0.0.0.0` 或留空，会导致服务注册了无效地址或无法被集群内其他实例访问。需要支持免外部依赖的智能 IP 自动探测，支持 Docker/K8s 环境变量、注册中心出网嗅探与物理网卡过滤保底。

## 变更范围
- 新增 `internal/netutil/ip.go`: 阶梯式 IP 解析工具函数
- 新增 `internal/netutil/ip_test.go`: 单元测试与环境模拟
- 修改 `sdk.go`: `Register` 和 `QuickRegister` 接入 IP 自动推导

## 实施步骤
- [x] Step 1: 编写 `internal/netutil/ip.go` 与 `ip_test.go`
- [x] Step 2: 改造 `sdk.go` 中的 `Register` 与 `QuickRegister`
- [x] Step 3: 执行测试 `go test ./...` 验证逻辑与编译通过

## 验证方式
- 运行 `go test ./...` 确保所有单元测试通过
- 验证空 IP、0.0.0.0、环境变量注入、UDP 探测等各种边界情况

## 风险评估
- UDP 嗅探无需真实发包与握手，即使 Nacos 端口暂时不可达也不会阻塞；兜底网卡过滤有效防止抓取 `docker0` / `veth*` 等虚拟网卡。
