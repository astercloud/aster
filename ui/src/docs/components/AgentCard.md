# AgentCard Agent 卡片

用于展示 Agent 信息的卡片组件。

## 基础用法

基本的 Agent 卡片。

```vue
<template>
  <AgentCard :agent="agent" @chat="handleChat" @edit="handleEdit" @delete="handleDelete" />
</template>

<script setup>
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

const handleChat = (agent) => {
  console.log("Start chat with:", agent.name);
};

const handleEdit = (agent) => {
  console.log("Edit agent:", agent.id);
};

const handleDelete = (agent) => {
  console.log("Delete agent:", agent.id);
};
</script>
```

## Agent 状态

Agent 有四种状态。

```vue
<template>
  <Flex direction="column" gap="md">
    <AgentCard :agent="{ ...agent, status: 'idle' }" />
    <AgentCard :agent="{ ...agent, status: 'thinking' }" />
    <AgentCard :agent="{ ...agent, status: 'busy' }" />
    <AgentCard :agent="{ ...agent, status: 'error' }" />
  </Flex>
</template>
```

## 带头像

Agent 可以显示自定义头像。

```vue
<template>
  <AgentCard
    :agent="{
      ...agent,
      avatar: 'https://example.com/agent-avatar.jpg',
    }"
  />
</template>
```

## 显示元数据

卡片自动显示 Agent 的模型信息。

```vue
<template>
  <AgentCard
    :agent="{
      id: '1',
      name: '代码助手',
      description: '编程问题解答专家',
      status: 'idle',
      metadata: {
        model: 'gpt-4',
        provider: 'openai',
        version: 'v1.0',
      },
    }"
  />
</template>
```

## API

### Props

| 参数  | 说明       | 类型    | 默认值 |
| ----- | ---------- | ------- | ------ |
| agent | Agent 对象 | `Agent` | -      |

### Agent 类型

```typescript
interface Agent {
  id: string;
  name: string;
  description?: string;
  avatar?: string;
  status: "idle" | "thinking" | "busy" | "error";
  metadata?: {
    model?: string;
    provider?: string;
    version?: string;
    [key: string]: any;
  };
}
```

### Events

| 事件名 | 说明               | 回调参数       |
| ------ | ------------------ | -------------- |
| chat   | 点击对话按钮时触发 | `agent: Agent` |
| edit   | 点击编辑按钮时触发 | `agent: Agent` |
| delete | 点击删除按钮时触发 | `agent: Agent` |

## 使用场景

- Agent 列表展示
- Agent 选择器
- Agent 管理面板
- Agent 市场

## 示例

### Agent 列表

```vue
<template>
  <div class="grid grid-cols-3 gap-4">
    <AgentCard v-for="agent in agents" :key="agent.id" :agent="agent" @chat="startChat" />
  </div>
</template>

<script setup>
import { ref } from "vue";

const agents = ref([
  {
    id: "1",
    name: "写作助手",
    description: "帮助你创作优质内容",
    status: "idle",
    metadata: { model: "claude-3-5-sonnet", provider: "anthropic" },
  },
  {
    id: "2",
    name: "代码助手",
    description: "编程问题解答专家",
    status: "idle",
    metadata: { model: "gpt-4", provider: "openai" },
  },
  {
    id: "3",
    name: "数据分析师",
    description: "数据洞察与可视化",
    status: "thinking",
    metadata: { model: "deepseek-chat", provider: "deepseek" },
  },
]);

const startChat = (agent) => {
  console.log("Starting chat with:", agent.name);
};
</script>
```

### Agent 状态监控

```vue
<template>
  <div>
    <h3>运行中的 Agents</h3>
    <Flex direction="column" gap="md">
      <AgentCard v-for="agent in runningAgents" :key="agent.id" :agent="agent" />
    </Flex>
  </div>
</template>

<script setup>
import { computed } from "vue";

const runningAgents = computed(() => {
  return agents.value.filter((a) => a.status === "thinking" || a.status === "busy");
});
</script>
```
