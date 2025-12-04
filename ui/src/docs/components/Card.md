# Card 卡片

用于显示结构化内容的卡片组件。

## 基础用法

基本的卡片。

```vue
<template>
  <Card title="推荐文章" content="这是一篇关于 AI Agent 的深度文章" />
</template>
```

## 带操作按钮

卡片可以包含操作按钮。

```vue
<template>
  <Card
    title="推荐内容"
    content="这是内容描述..."
    :actions="[
      { text: '查看详情', value: 'view' },
      { text: '分享', value: 'share' },
    ]"
    @action="handleAction"
  />
</template>

<script setup>
const handleAction = (action) => {
  console.log("Action:", action.value);
};
</script>
```

## 无标题

卡片可以不显示标题。

```vue
<template>
  <Card content="这是一段纯文本内容，没有标题" :actions="[{ text: '确定', value: 'ok' }]" />
</template>
```

## 富文本内容

内容支持 HTML。

```vue
<template>
  <Card title="格式化内容" content="<p>这是<strong>粗体</strong>文本</p><p>这是<em>斜体</em>文本</p>" />
</template>
```

## 使用场景

- 推荐内容展示
- 操作确认
- 信息卡片
- 选项选择

## API

### Props

| 参数    | 说明         | 类型       | 默认值 |
| ------- | ------------ | ---------- | ------ |
| title   | 卡片标题     | `string`   | -      |
| content | 卡片内容     | `string`   | -      |
| actions | 操作按钮列表 | `Action[]` | `[]`   |

### Action 类型

```typescript
interface Action {
  text: string; // 按钮文本
  value: string; // 按钮值
}
```

### Events

| 事件名 | 说明               | 回调参数         |
| ------ | ------------------ | ---------------- |
| action | 点击操作按钮时触发 | `action: Action` |

## 示例

### 确认对话框

```vue
<template>
  <Card
    title="确认删除"
    content="确定要删除这条消息吗？此操作无法撤销。"
    :actions="[
      { text: '取消', value: 'cancel' },
      { text: '删除', value: 'delete' },
    ]"
    @action="handleConfirm"
  />
</template>

<script setup>
const handleConfirm = (action) => {
  if (action.value === "delete") {
    console.log("Deleting...");
  }
};
</script>
```

### 选项卡片

```vue
<template>
  <Card
    title="选择操作"
    content="请选择你想要执行的操作："
    :actions="[
      { text: '生成文章', value: 'write' },
      { text: '分析代码', value: 'analyze' },
      { text: '创建工作流', value: 'workflow' },
    ]"
    @action="handleSelect"
  />
</template>
```
