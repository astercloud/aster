# Agent Example

这个示例演示了如何创建和使用 Agent SDK 的 Agent。

## 功能演示

1. **创建文件** - 使用 `Write` 工具创建文件
2. **读取文件** - 使用 `Read` 工具读取文件内容
3. **执行命令** - 使用 `Bash` 工具执行 bash 命令
4. **事件监听** - 监听 Progress 和 Monitor 事件

## 运行示例

### 前置条件

设置 Anthropic API Key:

```bash
export ANTHROPIC_API_KEY=your_api_key_here
```

### 运行

```bash
cd examples/agent
go run main.go
```

## 期望输出

示例将展示:

1. **Agent创建** - 显示生成的 Agent ID
2. **事件流** - 实时显示:
   - 状态变化 (Ready -> Working -> Ready)
   - 断点变化 (7段断点系统)
   - 文本流式输出
   - 工具调用开始/结束
   - Token使用统计
3. **对话结果** - 显示每次对话的最终回复
4. **最终状态** - 显示 Agent 的运行统计信息

## 代码结构

```go
main()
  ├─ 创建依赖 (Registry, Factory, Store)
  ├─ 注册模板
  ├─ 创建 Agent
  ├─ 订阅事件
  ├─ 发送消息 & 等待完成
  └─ 清理资源
```

## 事件处理

### Progress Events

- `ProgressTextChunkEvent` - 流式文本增量
- `ProgressToolStartEvent` - 工具开始执行
- `ProgressToolEndEvent` - 工具执行完成
- `ProgressDoneEvent` - 步骤完成

### Monitor Events

- `MonitorStateChangedEvent` - Agent状态变化
- `MonitorBreakpointChangedEvent` - 断点状态变化
- `MonitorTokenUsageEvent` - Token使用统计
- `MonitorErrorEvent` - 错误事件

## 工作空间

Agent 在 `./workspace` 目录下执行所有文件操作,该目录会:

- 自动创建(如果不存在)
- 作为沙箱边界
- 隔离文件系统访问
