.PHONY: help install-hooks lint fmt vet test test-integration clean build build-studio

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-hooks: ## 安装 git hooks
	@echo "安装 git hooks..."
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-commit
	@echo "✓ Git hooks 已安装到 .githooks/"
	@echo ""
	@echo "如需禁用 hooks，运行: git config --unset core.hooksPath"

lint: ## 运行 golangci-lint
	@echo "运行 golangci-lint..."
	@golangci-lint run --timeout=5m

lint-prod: ## 只检查生产代码（排除测试文件）
	@echo "检查生产代码（排除测试）..."
	@golangci-lint run ./examples/... ./pkg/... 2>&1 | grep -E "^(examples|pkg)/[^:]+\.go:" | grep -v "_test.go:" || echo "✓ 生产代码全部通过"

lint-fix: ## 运行 golangci-lint 并自动修复
	@echo "运行 golangci-lint (自动修复)..."
	@golangci-lint run --fix --timeout=5m

fmt: ## 格式化代码
	@echo "格式化代码..."
	@gofmt -w .
	@echo "✓ 代码已格式化"

vet: ## 运行 go vet
	@echo "运行 go vet..."
	@go vet ./...
	@echo "✓ go vet 通过"

test: ## 运行单元测试
	@echo "运行单元测试..."
	@go test ./... -v -short

test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	@go test ./test/integration/... -v

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ 覆盖率报告已生成: coverage.html"

clean: ## 清理构建产物
	@echo "清理构建产物..."
	@rm -rf .aster* coverage.out coverage.html
	@go clean
	@echo "✓ 清理完成"

build: ## 构建项目 (不包含 Studio)
	@echo "构建项目..."
	@go build -o bin/aster ./cmd/aster
	@go build -o bin/aster-server ./cmd/aster-server
	@echo "✓ 构建完成: bin/aster, bin/aster-server"

build-studio: ## 构建前端 Studio
	@echo "构建 Studio 前端..."
	@cd studio && npm install && npm run build
	@rm -rf server/studio/dist
	@cp -r studio/dist server/studio/dist
	@echo "✓ Studio 构建完成并复制到 server/studio/dist"

build-with-studio: build-studio ## 构建项目 (包含 Studio)
	@echo "构建项目 (包含 Studio)..."
	@go build -tags studio -o bin/aster ./cmd/aster
	@go build -tags studio -o bin/aster-server ./cmd/aster-server
	@echo "✓ 构建完成 (包含 Studio): bin/aster, bin/aster-server"

check: fmt vet lint ## 运行所有检查 (fmt + vet + lint)
	@echo ""
	@echo "✅ 所有检查通过!"

pre-commit: check ## 模拟 pre-commit 检查
	@echo "✅ Pre-commit 检查通过!"
