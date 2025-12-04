# ThinkBubble 思考气泡

显示 Agent 思考过程的气泡组件。

## 基础用法

基本的思考气泡。

```vue
<template>
  <ThinkBubble content="正在分析你的问题..." />
</template>
```

## 不同阶段

显示不同的思考阶段。

```vue
<template>
  <Flex direction="column" gap="md">
    <ThinkBubble content="正在理解问题..." />
    <ThinkBubble content="正在搜索相关信息..." />
    <ThinkBubble content="正在生成回答..." />
  </Flex>
</template>
```

## 无内容

不显示具体内容，只显示思考状态。

```vue
<template>
  <ThinkBubble />
</template>
```

## 使用场景

- Agent 正在处理请求
- 显示推理过程
- 工具调用状态
- 长时间操作的反馈

## API

### Props

| 参数    | 说明     | 类型     | 默认值 |
| ------- | -------- | -------- | ------ |
| content | 思考内容 | `string` | -      |

## 示例

### 与消息配合使用

```vue
<template>
  <Flex direction="column" gap="md">
    <Bubble content="帮我分析这段代码" position="right" />
    <ThinkBubble content="正在分析代码结构..." />
  </Flex>
</template>
```

### 动态更新

```vue
<template>
  <ThinkBubble :content="thinkingText" />
</template>

<script setup>
import { ref, onMounted } from "vue";

const thinkingText = ref("开始分析...");

onMounted(() => {
  setTimeout(() => {
    thinkingText.value = "正在搜索相关信息...";
  }, 1000);

  setTimeout(() => {
    thinkingText.value = "正在生成回答...";
  }, 2000);
});
</script>
```
