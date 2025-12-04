# Aster Agent UI 组件文档

专为 AI Agent 应用设计的 Vue 3 组件库

## 🤖 什么是 Aster Agent UI？

Aster Agent UI 是一个专门为 AI Agent 应用设计的组件库，提供了构建 Agent 管理、对话、工作流等功能所需的所有 UI 组件。

## ✨ 核心特性

- 🤖 **Agent 专属** - 专为 AI Agent 场景设计的组件
- 💬 **对话界面** - 完整的 Agent 对话体验
- 🔄 **工作流** - Agent 工作流可视化
- 👥 **多 Agent** - 支持多 Agent 协作
- 🎨 **现代设计** - 简洁美观的界面
- 🌙 **深色模式** - 完整的暗色主题
- 💪 **TypeScript** - 完整的类型定义
- ⚡️ **高性能** - 基于 Vue 3 Composition API

## 🚀 快速开始

### 安装依赖

```bash
npm install marked
```

### 创建你的第一个 Agent

```vue
<template>
  <AgentCard :agent="agent" @chat="startChat" />
</template>

<script setup>
import { AgentCard } from "@/components/Agent";

const agent = {
  id: "1",
  name: "写作助手",
  description: "帮助你创作优质内容",
  status: "idle",
  metadata: {
    model: "claude-3-5-sonnet",
    provider: "anthropic",
  },
};

const startChat = (agent) => {
  console.log("Start chat with:", agent.name);
};
</script>
```

### 创建 Agent 对话界面

```vue
<template>
  <Chat :messages="messages" @send="handleSend" />
</template>

<script setup>
import { ref } from "vue";
import { Chat } from "@/components/ChatUI";

const messages = ref([
  {
    id: "1",
    type: "text",
    content: "你好！我是 AI Agent，有什么可以帮助你的吗？",
    position: "left",
  },
]);

const handleSend = async (message) => {
  // 添加用户消息
  messages.value.push({
    id: Date.now().toString(),
    type: "text",
    content: message.content,
    position: "right",
  });

  // 调用 Agent API
  const response = await callAgent(message.content);

  // 添加 Agent 回复
  messages.value.push({
    id: Date.now().toString(),
    type: "text",
    content: response,
    position: "left",
  });
};
</script>
```

## 📦 组件分类

### Agent 组件

专为 AI Agent 设计的核心组件。

- [AgentCard](/docs/components/AgentCard.md) - Agent 信息卡片
- [AgentList](/docs/components/AgentList.md) - Agent 列表
- [AgentForm](/docs/components/AgentForm.md) - Agent 表单
- [WorkflowCard](/docs/components/WorkflowCard.md) - 工作流卡片
- [RoomCard](/docs/components/RoomCard.md) - 协作房间卡片

### 对话组件

用于构建 Agent 对话界面。

- [Chat](/docs/components/Chat.md) - 聊天容器
- [Bubble](/docs/components/Bubble.md) - 消息气泡
- [ThinkBubble](/docs/components/ThinkBubble.md) - 思考气泡
- [Card](/docs/components/Card.md) - 卡片消息

### 基础组件

通用的 UI 组件。

- [Button](/docs/components/Button.md) - 按钮
- [Avatar](/docs/components/Avatar.md) - 头像
- [Icon](/docs/components/Icon.md) - 图标
- [Tag](/docs/components/Tag.md) - 标签

### 表单组件

用于数据输入。

- [Input](/docs/components/Input.md) - 输入框
- [Search](/docs/components/Search.md) - 搜索框
- [Checkbox](/docs/components/Checkbox.md) - 复选框
- [Radio](/docs/components/Radio.md) - 单选框

### 反馈组件

用于用户反馈。

- [Notice](/docs/components/Notice.md) - 通知提示
- [Modal](/docs/components/Modal.md) - 模态框
- [Progress](/docs/components/Progress.md) - 进度条
- [Tooltip](/docs/components/Tooltip.md) - 工具提示

## 🎯 使用场景

### Agent 管理

```vue
<template>
  <AgentDashboard @chat="handleChat" />
</template>
```

### Agent 对话

```vue
<template>
  <AgentChatSession :agent="selectedAgent" @back="goBack" />
</template>
```

### 工作流编排

```vue
<template>
  <WorkflowCard :workflow="workflow" @run="runWorkflow" />
</template>
```

### 多 Agent 协作

```vue
<template>
  <RoomCard :room="room" @join="joinRoom" />
</template>
```

## 🔧 与 Aster 后端集成

### 使用 useAsterClient

```vue
<script setup>
import { useAsterClient } from "@/composables/useAsterClient";

const { client } = useAsterClient();

// 获取 Agent 列表
const agents = await client.agents.list();

// 与 Agent 对话
const response = await client.agents.chat(agentId, message);

// 创建 Agent
const newAgent = await client.agents.create({
  template_id: "chat",
  name: "我的 Agent",
  model_config: {
    provider: "anthropic",
    model: "claude-3-5-sonnet",
  },
});
</script>
```

### WebSocket 实时通信

```vue
<script setup>
import { useAsterClient } from "@/composables/useAsterClient";

const { client, ws, subscribe } = useAsterClient();

// 初始化 WebSocket
await client.reconnect();

// 订阅 Agent 事件
const unsubscribe = subscribe(["progress"], {
  agentId: "agent-1",
});

// 监听消息
ws.value.onMessage((message) => {
  console.log("Agent message:", message);
});
</script>
```

## 🎨 主题定制

所有组件使用 Tailwind CSS，支持深色模式。

### 自定义颜色

在 `tailwind.config.js` 中配置：

```javascript
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: "#3b82f6",
        secondary: "#64748b",
      },
    },
  },
};
```

## 📖 最佳实践

### Agent 状态管理

使用 Vue 的响应式系统管理 Agent 状态：

```vue
<script setup>
import { ref, computed } from "vue";

const agents = ref([]);
const activeAgents = computed(() => agents.value.filter((a) => a.status !== "idle"));
</script>
```

### 错误处理

显示友好的错误消息：

```vue
<script setup>
const handleError = (error) => {
  // 显示错误通知
  showNotice({
    type: "error",
    content: error.message,
  });
};
</script>
```

### 加载状态

使用思考气泡显示 Agent 处理状态：

```vue
<template>
  <ThinkBubble v-if="isThinking" content="Agent 正在思考..." />
</template>
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License
