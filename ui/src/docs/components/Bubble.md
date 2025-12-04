# Bubble 消息气泡

用于显示对话消息的气泡组件。

## 基础用法

基本的消息气泡。

```vue
<template>
  <Flex direction="column" gap="md">
    <Bubble content="你好！" position="left" />
    <Bubble content="很高兴认识你" position="right" />
  </Flex>
</template>
```

## 带头像

消息气泡可以显示头像。

```vue
<template>
  <Flex direction="column" gap="md">
    <Bubble content="我是 AI 助手" position="left" avatar="https://example.com/avatar.jpg" />
    <Bubble content="你好" position="right" avatar="https://example.com/user.jpg" />
  </Flex>
</template>
```

## 消息状态

右侧消息可以显示发送状态。

```vue
<template>
  <Flex direction="column" gap="md">
    <Bubble content="发送中..." position="right" status="pending" />
    <Bubble content="已发送" position="right" status="sent" />
    <Bubble content="发送失败" position="right" status="error" />
  </Flex>
</template>
```

## Markdown 支持

消息内容支持 Markdown 格式。

```vue
<template>
  <Bubble content="这是 **粗体** 和 *斜体* 文本，还有 `代码`" position="left" />
</template>
```

## 代码块

支持代码块高亮。

```vue
<template>
  <Bubble
    :content="`\`\`\`javascript
function hello() {
  console.log('Hello World');
}
\`\`\``"
    position="left"
  />
</template>
```

## API

### Props

| 参数     | 说明                      | 类型                             | 默认值   |
| -------- | ------------------------- | -------------------------------- | -------- |
| content  | 消息内容（支持 Markdown） | `string`                         | -        |
| position | 气泡位置                  | `'left' \| 'right'`              | `'left'` |
| status   | 消息状态（仅右侧有效）    | `'pending' \| 'sent' \| 'error'` | -        |
| avatar   | 头像 URL                  | `string`                         | -        |
