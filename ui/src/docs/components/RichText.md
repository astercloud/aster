# RichText 富文本

用于渲染 Markdown 格式文本的组件。

## 基础用法

基本的富文本渲染。

```vue
<template>
  <RichText :content="markdown" />
</template>

<script setup>
const markdown = `
# 标题

这是一段**粗体**和*斜体*文本。

- 列表项 1
- 列表项 2
- 列表项 3
`;
</script>
```

## 代码块

支持代码块高亮。

```vue
<template>
  <RichText :content="codeMarkdown" />
</template>

<script setup>
const codeMarkdown = `
\`\`\`javascript
function hello() {
  console.log('Hello World');
}
\`\`\`
`;
</script>
```

## 表格

支持表格渲染。

```vue
<template>
  <RichText :content="tableMarkdown" />
</template>

<script setup>
const tableMarkdown = `
| 参数 | 说明 | 类型 |
| --- | --- | --- |
| content | 内容 | string |
| type | 类型 | string |
`;
</script>
```

## 链接

支持链接渲染。

```vue
<template>
  <RichText :content="linkMarkdown" />
</template>

<script setup>
const linkMarkdown = `
访问 [ChatUI](https://chatui.io) 了解更多。
`;
</script>
```

## API

### Props

| 参数    | 说明          | 类型     | 默认值 |
| ------- | ------------- | -------- | ------ |
| content | Markdown 内容 | `string` | -      |

## 支持的 Markdown 语法

- 标题 (`#`, `##`, `###`)
- 粗体 (`**text**`)
- 斜体 (`*text*`)
- 代码 (`` `code` ``)
- 代码块 (` ```language `)
- 列表 (`-`, `1.`)
- 链接 (`[text](url)`)
- 引用 (`> text`)
- 表格
- 分割线 (`---`)

## 示例

### 文章内容

```vue
<template>
  <RichText :content="article" />
</template>

<script setup>
const article = `
# Vue 3 组件开发指南

## 简介

Vue 3 是一个渐进式 JavaScript 框架。

## 特性

- **响应式系统** - 基于 Proxy 的响应式
- **组合式 API** - 更好的逻辑复用
- **性能提升** - 更快的渲染速度

## 代码示例

\`\`\`vue
<template>
  <div>{{ message }}</div>
</template>

<script setup>
import { ref } from 'vue';
const message = ref('Hello Vue 3');
</script>
\`\`\`

## 总结

Vue 3 带来了许多改进和新特性。
`;
</script>
```
