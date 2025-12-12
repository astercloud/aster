# Aster Custom Claude API Example

演示如何使用 Aster 框架接入自定义 Claude API 端点（包括第三方中转服务）。

## 功能特性

- ✅ 自定义 API 端点支持
- ✅ Store 上下文管理（自动修剪）
- ✅ 流式输出
- ✅ 事件监听
- ✅ 工具调用
- ✅ 200K 上下文窗口

## 快速开始

### 1. 配置环境变量

复制示例配置：
```bash
cp .env.example .env
```

编辑 `.env` 文件，填入您的配置：
```bash
CLAUDE_API_KEY=your-api-key-here
CLAUDE_BASE_URL=https://api.anthropic.com
CLAUDE_MODEL=claude-sonnet-4-5-20250929
```

### 2. 运行示例

```bash
go run main.go
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `CLAUDE_API_KEY` | API 密钥（必填） | - |
| `CLAUDE_BASE_URL` | API 端点 | `https://api.anthropic.com` |
| `CLAUDE_MODEL` | 模型名称 | `claude-sonnet-4-5-20250929` |
| `STORE_DIR` | 持久化目录 | `.aster` |
| `STORE_MAX_MESSAGES` | 最大消息数 | `20` |
| `STORE_AUTO_TRIM` | 自动修剪 | `true` |
| `SANDBOX_WORK_DIR` | 沙箱工作目录 | `./workspace` |

### YAML 配置（可选）

参考 `config.example.yaml` 文件。

## Store 上下文管理

Aster 提供了两层上下文管理机制：

### 1. 持久化层（Store）
- 限制磁盘上保存的消息数量
- 防止无限增长
- FIFO 策略（先进先出）

```go
Store: &types.StoreConfig{
    MaxMessages: 20,   // 最多保留 20 条消息
    AutoTrim:    true, // 自动修剪
}
```

### 2. 运行时层（Context）
- 智能压缩对话历史
- 节省 Token 成本
- 保持上下文连贯性

```go
Context: &types.ContextManagerOptions{
    MaxTokens: 200000, // 200K 上下文窗口
}
```

## 使用第三方中转服务

如果您使用第三方 Claude API 中转服务，只需设置 `CLAUDE_BASE_URL`：

```bash
# 示例：使用中转服务
export CLAUDE_BASE_URL=https://your-relay-service.com
export CLAUDE_API_KEY=your-relay-api-key
```

Aster 的 `CustomClaudeProvider` 会自动处理不同中转服务的响应格式差异。

## 项目结构

```
.
├── .env.example          # 环境变量示例
├── .gitignore           # Git 忽略文件
├── README.md            # 本文档
├── config.example.yaml  # 配置文件示例
└── main.go              # 主程序
```

## 安全注意事项

⚠️ **重要：永远不要将 API 密钥提交到版本控制系统**

- 使用 `.env` 文件存储敏感信息
- 确保 `.env` 在 `.gitignore` 中
- 生产环境使用环境变量或密钥管理服务

## 故障排查

### API 密钥未配置
```
❌ 错误: 未配置 CLAUDE_API_KEY
```
**解决方法**：设置环境变量 `export CLAUDE_API_KEY=your-key`

### 连接超时
**可能原因**：
- 网络问题
- API 端点不可用
- 防火墙限制

**解决方法**：检查网络连接和 API 端点状态

### 消息数持续增长
**可能原因**：未启用 Store 自动修剪

**解决方法**：
```bash
export STORE_AUTO_TRIM=true
export STORE_MAX_MESSAGES=20
```

## 相关资源

- [Aster 框架文档](https://github.com/astercloud/aster)
- [Claude API 文档](https://docs.anthropic.com)

## License

本示例代码遵循 Aster 项目的许可证。
