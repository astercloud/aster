# Logic Memory 示例

这个示例演示了 Aster SDK 的 Logic Memory 功能，展示了如何让 AI Agent 从用户交互中自动学习偏好和行为模式。

## 前置要求

1. **Go 1.24+** 已安装
2. **国内镜像配置**（可选，加速下载依赖）

### Go 国内镜像配置

如果下载依赖较慢，可以配置国内镜像：

```bash
# 方式 1: 使用 goproxy.cn（推荐）
export GOPROXY=https://goproxy.cn,direct

# 方式 2: 使用七牛云镜像
export GOPROXY=https://goproxy.io,direct

# 方式 3: 使用阿里云镜像
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 永久配置（写入 ~/.zshrc 或 ~/.bashrc）
echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.zshrc
source ~/.zshrc
```

## 运行示例

### 1. 下载依赖

```bash
cd /path/to/aster
go mod download
```

### 2. 运行示例

```bash
# 直接运行
go run ./examples/logic-memory

# 或者先编译再运行
go build ./examples/logic-memory
./logic-memory
```

### 3. 运行测试

```bash
# 运行所有测试
go test ./...

# 只运行 Logic Memory 相关测试
go test ./pkg/memory/logic/...
go test ./pkg/middleware/... -run LogicMemory
```

## 示例内容

这个示例包含 5 个演示场景：

1. **基础用法** - 手动记录和检索 Memory
2. **事件处理** - 通过 PatternMatcher 自动识别模式
3. **Middleware 集成** - 自动捕获和注入 Memory
4. **Memory 合并** - Consolidation 功能演示
5. **Memory 清理** - Pruning 功能演示

## 常见问题

### Q: 运行时报错 "package not found"

**A:** 确保已下载所有依赖：
```bash
go mod download
go mod tidy
```

### Q: 测试时显示 "[no test files]"

**A:** 这是正常的，表示该包没有测试文件。Logic Memory 的测试在：
- `pkg/memory/logic/*_test.go`
- `pkg/middleware/logic_memory_test.go`

### Q: 编译警告 "redundant newline"

**A:** 这是代码风格警告，不影响运行。如果看到这个警告，说明代码需要格式化：
```bash
go fmt ./examples/logic-memory
```

## 相关文档

- [Logic Memory 完整文档](../../docs/content/04.memory/11.logic-memory.md)
- [Logic Memory 设计计划](../../docs/content/04.memory/12.logic-memory-plan.md)
- [Memory 系统总览](../../docs/content/04.memory/1.overview.md)
