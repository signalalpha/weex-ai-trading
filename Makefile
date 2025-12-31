.PHONY: build build-linux clean help test run

# 项目信息
BINARY_NAME=trader
CMD_PATH=./cmd/trader
BUILD_DIR=./bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME=$(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

# 默认目标
.DEFAULT_GOAL := help

help: ## 显示帮助信息
	@echo "可用的 make 目标:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

build: ## 构建当前平台版本
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux: ## 构建 Linux AMD64 版本（用于服务器部署）
	@echo "构建 Linux AMD64 版本..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"
	@echo ""
	@echo "📦 文件已准备好，可以拷贝到服务器:"
	@echo "   scp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 user@server:/path/to/destination/"

build-all: ## 构建多个平台版本
	@echo "构建多平台版本..."
	@mkdir -p $(BUILD_DIR)
	@echo "  - Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "  - Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	@echo "  - macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "  - macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "✅ 所有平台构建完成"

clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@echo "✅ 清理完成"

test: ## 运行测试
	@echo "运行测试..."
	@go test -v ./...

test-race: ## 运行竞态检测测试
	@echo "运行竞态检测测试..."
	@go test -race -v ./...

fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "✅ 格式化完成"

vet: ## 运行 go vet
	@echo "运行 go vet..."
	@go vet ./...
	@echo "✅ vet 完成"

lint: fmt vet ## 运行代码检查（格式化 + vet）

deps: ## 下载依赖
	@echo "下载依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖下载完成"

run: ## 运行程序（开发模式）
	@echo "运行程序..."
	@go run $(CMD_PATH)/main.go

install: build ## 安装到本地（GOPATH/bin）
	@echo "安装 $(BINARY_NAME)..."
	@go install $(CMD_PATH)
	@echo "✅ 安装完成"

# 开发相关
dev-setup: deps fmt vet ## 设置开发环境（下载依赖、格式化、检查）

# 部署相关
deploy-check: build-linux ## 构建并检查部署文件
	@echo ""
	@echo "📋 部署检查清单:"
	@echo "  ✅ 二进制文件已构建: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"
	@file $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 || true
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 || true
	@echo ""
	@echo "📝 部署步骤:"
	@echo "  1. 在服务器上创建目录: mkdir -p /path/to/trader"
	@echo "  2. 拷贝二进制文件: scp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 user@server:/path/to/trader/"
	@echo "  3. 拷贝配置文件: scp .env user@server:/path/to/trader/ (如果使用 .env)"
	@echo "  4. 在服务器上设置权限: chmod +x /path/to/trader/$(BINARY_NAME)-linux-amd64"
	@echo "  5. 运行: ./$(BINARY_NAME)-linux-amd64 --help"

