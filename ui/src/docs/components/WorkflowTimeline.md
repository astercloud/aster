# WorkflowTimeline 工作流时间线

交互式工作流时间线组件，展示 Agent 执行的各个步骤。

## 基础用法

基本的工作流时间线。

```vue
<template>
  <WorkflowTimeline :steps="steps" :current-step="currentStep" title="写作流程" @step-change="handleStepChange" />
</template>

<script setup>
import { ref } from "vue";

const currentStep = ref(0);

const steps = [
  {
    id: "specify",
    name: "定义需求",
    icon: "📝",
    description: "确定主题与受众",
  },
  {
    id: "research",
    name: "信息调研",
    icon: "🔍",
    description: "收集背景信息",
  },
  {
    id: "write",
    name: "创作初稿",
    icon: "✍️",
    description: "生成文章初稿",
  },
];

const handleStepChange = (step) => {
  currentStep.value = step;
};
</script>
```

## 带快捷操作

每个步骤可以包含快捷操作按钮。

```vue
<template>
  <WorkflowTimeline :steps="stepsWithActions" :current-step="currentStep" @step-change="handleStepChange" @action="handleAction" />
</template>

<script setup>
const stepsWithActions = [
  {
    id: "topic",
    name: "选题讨论",
    icon: "💡",
    description: "头脑风暴与定题",
    actions: [
      {
        id: "drain_ideas",
        label: "创意排水",
        icon: "lightbulb",
        variant: "primary",
      },
      {
        id: "title_gen",
        label: "生成标题",
        icon: "wand",
        variant: "secondary",
      },
    ],
  },
  // ...
];

const handleAction = (action) => {
  console.log("Action:", action.id);
};
</script>
```

## 显示返回按钮

添加返回按钮。

```vue
<template>
  <WorkflowTimeline :steps="steps" :current-step="currentStep" title="我的项目" :show-back="true" @back="handleBack" />
</template>

<script setup>
const handleBack = () => {
  console.log("Go back");
};
</script>
```

## API

### Props

| 参数        | 说明             | 类型             | 默认值     |
| ----------- | ---------------- | ---------------- | ---------- |
| steps       | 步骤列表         | `WorkflowStep[]` | `[]`       |
| currentStep | 当前步骤索引     | `number`         | `0`        |
| title       | 标题             | `string`         | `'工作流'` |
| showBack    | 是否显示返回按钮 | `boolean`        | `false`    |

### WorkflowStep 类型

```typescript
interface WorkflowStep {
  id: string;
  name: string;
  icon: string;
  description: string;
  actions?: StepAction[];
}

interface StepAction {
  id: string;
  label: string;
  icon?: string;
  variant?: "primary" | "secondary";
}
```

### Events

| 事件名      | 说明               | 回调参数             |
| ----------- | ------------------ | -------------------- |
| step-change | 步骤改变时触发     | `step: number`       |
| action      | 点击快捷操作时触发 | `action: StepAction` |
| back        | 点击返回按钮时触发 | -                    |

## 使用场景

- Agent 工作流可视化
- 多步骤任务进度展示
- 引导式操作流程
- 写作/创作流程管理

## 示例

### 完整的写作流程

```vue
<template>
  <div class="flex h-screen">
    <WorkflowTimeline :steps="writingSteps" :current-step="currentStep" title="文章创作" :show-back="true" @step-change="handleStepChange" @action="handleQuickAction" @back="goBack" />

    <div class="flex-1">
      <!-- 主内容区域 -->
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";

const currentStep = ref(0);

const writingSteps = [
  {
    id: "specify",
    name: "定义需求",
    icon: "📝",
    description: "确定主题与受众",
  },
  {
    id: "topic",
    name: "选题讨论",
    icon: "💡",
    description: "头脑风暴与定题",
    actions: [{ id: "drain_ideas", label: "创意排水", icon: "lightbulb", variant: "primary" }],
  },
  {
    id: "research",
    name: "信息调研",
    icon: "🔍",
    description: "收集背景信息",
    actions: [{ id: "deep_research", label: "深度调研", icon: "search", variant: "primary" }],
  },
  {
    id: "write",
    name: "创作初稿",
    icon: "✍️",
    description: "生成文章初稿",
    actions: [{ id: "generate_draft", label: "生成草稿", icon: "wand", variant: "primary" }],
  },
  {
    id: "review",
    name: "三遍审校",
    icon: "🔎",
    description: "润色与优化",
    actions: [{ id: "start_review", label: "开始审校", icon: "check", variant: "primary" }],
  },
];

const handleStepChange = (step) => {
  currentStep.value = step;
};

const handleQuickAction = (action) => {
  console.log("Quick action:", action.id);
  // 执行对应的操作
};

const goBack = () => {
  console.log("Go back to dashboard");
};
</script>
```

### 与 Agent 集成

```vue
<template>
  <div class="flex h-screen">
    <WorkflowTimeline :steps="steps" :current-step="currentStep" @action="executeAgentAction" />

    <Chat :messages="messages" @send="handleSend" />
  </div>
</template>

<script setup>
import { ref } from "vue";
import { useAsterClient } from "@/composables/useAsterClient";

const { client } = useAsterClient();
const currentStep = ref(0);
const messages = ref([]);

const executeAgentAction = async (action) => {
  // 根据操作 ID 生成提示词
  const prompts = {
    drain_ideas: "请帮我对当前主题进行创意排水",
    deep_research: "请搜索最新的行业报告和数据",
    generate_draft: "请基于大纲生成文章初稿",
  };

  const prompt = prompts[action.id];
  if (prompt) {
    // 调用 Agent
    const response = await client.agents.chat(agentId, prompt);
    messages.value.push({
      id: Date.now().toString(),
      type: "text",
      content: response.data.text,
      position: "left",
    });
  }
};
</script>
```
