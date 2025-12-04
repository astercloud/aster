# WorkflowCard 工作流卡片

用于展示 Agent 工作流的卡片组件。

## 基础用法

基本的工作流卡片。

```vue
<template>
  <WorkflowCard :workflow="workflow" @run="handleRun" @edit="handleEdit" />
</template>

<script setup>
const workflow = {
  id: "1",
  name: "文章生成流程",
  description: "自动生成高质量文章",
  steps: [
    { id: "1", name: "主题分析", type: "agent", status: "completed" },
    { id: "2", name: "内容生成", type: "agent", status: "running" },
    { id: "3", name: "质量检查", type: "tool", status: "pending" },
  ],
  status: "running",
  currentStep: 1,
};

const handleRun = (workflow) => {
  console.log("Run workflow:", workflow.id);
};

const handleEdit = (workflow) => {
  console.log("Edit workflow:", workflow.id);
};
</script>
```

## 工作流状态

工作流有多种状态。

```vue
<template>
  <Flex direction="column" gap="md">
    <WorkflowCard :workflow="{ ...workflow, status: 'idle' }" />
    <WorkflowCard :workflow="{ ...workflow, status: 'running' }" />
    <WorkflowCard :workflow="{ ...workflow, status: 'paused' }" />
    <WorkflowCard :workflow="{ ...workflow, status: 'completed' }" />
    <WorkflowCard :workflow="{ ...workflow, status: 'error' }" />
  </Flex>
</template>
```

## 步骤显示

显示工作流的执行步骤。

```vue
<template>
  <WorkflowCard
    :workflow="{
      id: '1',
      name: '数据处理流程',
      steps: [
        { id: '1', name: '数据采集', type: 'tool', status: 'completed' },
        { id: '2', name: 'Agent 分析', type: 'agent', status: 'running' },
        { id: '3', name: '结果输出', type: 'tool', status: 'pending' },
      ],
      status: 'running',
      currentStep: 1,
    }"
  />
</template>
```

## API

### Props

| 参数     | 说明       | 类型       | 默认值 |
| -------- | ---------- | ---------- | ------ |
| workflow | 工作流对象 | `Workflow` | -      |

### Workflow 类型

```typescript
interface Workflow {
  id: string;
  name: string;
  description?: string;
  steps: WorkflowStep[];
  status: "idle" | "running" | "paused" | "completed" | "error";
  currentStep?: number;
}

interface WorkflowStep {
  id: string;
  name: string;
  type: "agent" | "tool" | "condition" | "loop";
  status: "pending" | "running" | "completed" | "error" | "skipped";
  config?: Record<string, any>;
}
```

### Events

| 事件名 | 说明               | 回调参数             |
| ------ | ------------------ | -------------------- |
| run    | 点击运行按钮时触发 | `workflow: Workflow` |
| pause  | 点击暂停按钮时触发 | `workflow: Workflow` |
| edit   | 点击编辑按钮时触发 | `workflow: Workflow` |
| delete | 点击删除按钮时触发 | `workflow: Workflow` |

## 使用场景

- 工作流管理
- 自动化任务展示
- Agent 编排
- 流程监控

## 示例

### 工作流列表

```vue
<template>
  <div class="space-y-4">
    <WorkflowCard v-for="workflow in workflows" :key="workflow.id" :workflow="workflow" @run="runWorkflow" @edit="editWorkflow" />
  </div>
</template>

<script setup>
import { ref } from "vue";

const workflows = ref([
  {
    id: "1",
    name: "内容生成流程",
    description: "自动生成和优化内容",
    steps: [
      { id: "1", name: "主题分析", type: "agent", status: "completed" },
      { id: "2", name: "内容生成", type: "agent", status: "completed" },
      { id: "3", name: "质量检查", type: "tool", status: "completed" },
    ],
    status: "completed",
  },
  {
    id: "2",
    name: "代码审查流程",
    description: "Agent 自动审查代码",
    steps: [
      { id: "1", name: "代码分析", type: "agent", status: "completed" },
      { id: "2", name: "问题检测", type: "agent", status: "running" },
      { id: "3", name: "生成报告", type: "tool", status: "pending" },
    ],
    status: "running",
    currentStep: 1,
  },
]);

const runWorkflow = (workflow) => {
  console.log("Running workflow:", workflow.name);
};

const editWorkflow = (workflow) => {
  console.log("Editing workflow:", workflow.id);
};
</script>
```
