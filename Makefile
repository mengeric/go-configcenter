# ------------------------------------------------------------------------------
# CNCF / Kubernetes / Operator-SDK Standard Makefile
# ------------------------------------------------------------------------------

SHELL := /bin/bash
.DEFAULT_GOAL := help

# 允许本地扩展与环境变量覆盖
-include .env
-include Makefile.local

# ------------------------------------------------------------------------------
# 基础变量（使用 ?= 允许外部命令行或 CI/CD 覆盖）
# ------------------------------------------------------------------------------
APP_NAME    ?= go-configcenter
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)

##@ General
.PHONY: help
help: ## 显示 Makefile 可用目标与分组帮助信息
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mUsage:\033[0m\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: fmt tidy
fmt: ## 执行代码格式化 (gofmt)
	go fmt ./...

tidy: ## 整理 Go 依赖模块
	go mod tidy

##@ Testing
.PHONY: test test-adversarial test-coverage
test: ## 运行全量单元测试
	go test ./... -v

test-adversarial: ## 运行抗性与边界防御测试套件
	go test -v ./tests/adversarial/...

test-coverage: ## 运行测试并生成覆盖率报告
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
