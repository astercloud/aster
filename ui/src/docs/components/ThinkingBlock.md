# ThinkingBlock 思考块

显示 Agent 完整推理过程的组件，支持工具调用、结果展示和人工审批。

## 基础用法

基本的思考块展示。

```vue
<template>
  <ThinkingBlock :thoughts="thoughts" :is-finished="false" />
</template>

<script setup>
import { ref } from "vue";

const thoughts = ref([
  {
    id: "1",
    stage: "Thinking",
    reasoning: "用户想要生成一篇文章，我需要先分析主题...",
    decision: "开始主题分析",
    timestamp: new Date().toISOString(),
  },
]);
</script>
```

## 工具调用展示

显示 Agent 调用工具的过程。

```vue
<template>
  <ThinkingBlock
    :thoughts="[
      {
        id: '1',
        stage: 'Tool: WebSearch',
        reasoning: '需要搜索最新的行业数据',
        decision: '调用搜索工具',
        timestamp: new Date().toISOString(),
        toolCall: {
          toolName: 'web_search',
          args: {
            query: 'AI Agent 最新趋势',
            limit: 10,
          },
        },
      },
      {
        id: '2',
        stage: 'Tool Result',
        reasoning: '搜索完成，获得 10 条结果',
        decision: '分析搜索结果',
        timestamp: new Date().toISOString(),
        toolResult: {
          toolName: 'web_search',
          result: {
            count: 10,
            items: ['结果1', '结果2'],
          },
        },
      },
    ]"
    :is-finished="true"
  />
</template>
```

## 人工审批（HITL）

显示需要人工审批的操作。

```vue
<template>
  <ThinkingBlock :thoughts="thoughts" :is-finished="false" :pending-approval="approvalRequest" @approve="handleApprove" @reject="handleReject" />
</template>

<script setup>
import { ref } from "vue";

const thoughts = ref([
  {
    id: "1",
    stage: "Human in the Loop",
    reasoning: "需要执行敏感操作：删除文件",
    decision: "等待用户审批",
    timestamp: new Date().toISOString(),
    approvalRequest: {
      id: "approval-1",
      toolName: "delete_file",
      args: {
        path: "/important/file.txt",
      },
    },
  },
]);

const approvalRequest = ref({
  id: "approval-1",
  toolName: "delete_file",
  args: {
    path: "/important/file.txt",
  },
});

const handleApprove = (request) => {
  console.log("Approved:", request);
  // 继续执行 Agent
};

const handleReject = () => {
  console.log("Rejected");
  // 终止执行
};
</script>
```

## 可折叠

思考块可以折叠/展开。

```vue
<template>
  <ThinkingBlock :thoughts="thoughts" :is-finished="true" :default-expanded="false" />
</template>
```

## API

### Props

| 参数            | 说明         | 类型                | 默认值  |
| --------------- | ------------ | ------------------- | ------- |
| thoughts        | 思考事件列表 | `ThinkAloudEvent[]` | `[]`    |
| isFinished      | 是否已完成   | `boolean`           | `false` |
| pendingApproval | 待审批请求   | `ApprovalRequest`   | -       |
| defaultExpanded | 默认是否展开 | `boolean`           | `true`  |

### ThinkAloudEvent 类型

```typescript
interface ThinkAloudEvent {
  id: string;
  stage: string; // 'Thinking', 'Tool: WebSearch', 'HITL' 等
  reasoning: string; // 推理过程
  decision: string; // 决策
  timestamp: string;
  context?: Record<string, any>;
  toolCall?: {
    toolName: string;
    args: Record<string, any>;
  };
  toolResult?: {
    toolName: string;
    result: Record<string, any>;
  };
  approvalRequest?: ApprovalRequest;
}
```

### ApprovalRequest 类型

```typescript
interface ApprovalRequest {
  id: string;
  toolName: string;
  args: Record<string, any>;
}
```

### Events

| 事件名  | 说明               | 回调参数                   |
| ------- | ------------------ | -------------------------- |
| approve | 批准审批请求时触发 | `request: ApprovalRequest` |
| reject  | 拒绝审批请求时触发 | -                          |

## 使用场景

- Agent 推理过程可视化
- 工具调用监控
- 调试 Agent 行为
- 人工审批敏感操作
- Agent 透明度展示

## 示例

### 完整的 Agent 执行流程

```vue
<template>
  <div class="space-y-4">
    <Bubble content="帮我写一篇关于 AI 的文章" position="right" />

    <ThinkingBlock :thoughts="agentThoughts" :is-finished="isFinished" :pending-approval="pendingApproval" @approve="handleApprove" @reject="handleReject" />

    <Bubble v-if="agentResponse" :content="agentResponse" position="left" />
  </div>
</template>

<script setup>
import { ref } from "vue";

const agentThoughts = ref([
  {
    id: "1",
    stage: "Thinking",
    reasoning: "用户想要一篇关于 AI 的文章，我需要先确定主题方向",
    decision: "开始主题分析",
    timestamp: new Date().toISOString(),
  },
  {
    id: "2",
    stage: "Tool: WebSearch",
    reasoning: "需要搜索 AI 的最新趋势",
    decision: "调用搜索工具",
    timestamp: new Date().toISOString(),
    toolCall: {
      toolName: "web_search",
      args: { query: "AI 最新趋势 2024" },
    },
  },
  {
    id: "3",
    stage: "Tool Result",
    reasoning: "获得了 10 条搜索结果",
    decision: "分析并提炼关键信息",
    timestamp: new Date().toISOString(),
    toolResult: {
      toolName: "web_search",
      result: { count: 10, summary: "..." },
    },
  },
]);

const isFinished = ref(false);
const pendingApproval = ref(null);
const agentResponse = ref("");
</script>
```
