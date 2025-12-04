# Notice 通知提示

用于显示通知消息。

## 基础用法

基本的通知提示。

```vue
<template>
  <Notice content="这是一条信息提示" />
</template>
```

## 不同类型

通知有四种类型。

```vue
<template>
  <Flex direction="column" gap="md">
    <Notice type="info" content="这是一条信息提示" />
    <Notice type="success" content="操作成功" />
    <Notice type="warning" content="请注意检查" />
    <Notice type="error" content="发生错误" />
  </Flex>
</template>
```

## 带标题

通知可以带标题。

```vue
<template>
  <Notice type="success" title="成功" content="操作已成功完成" />
</template>
```

## 可关闭

通知可以被关闭。

```vue
<template>
  <Notice type="info" content="这是一条可关闭的提示" closable @close="handleClose" />
</template>

<script setup>
const handleClose = () => {
  console.log("Notice closed");
};
</script>
```

## API

### Props

| 参数     | 说明       | 类型                                          | 默认值   |
| -------- | ---------- | --------------------------------------------- | -------- |
| type     | 通知类型   | `'info' \| 'success' \| 'warning' \| 'error'` | `'info'` |
| title    | 标题       | `string`                                      | -        |
| content  | 内容       | `string`                                      | -        |
| closable | 是否可关闭 | `boolean`                                     | `false`  |

### Events

| 事件名 | 说明       | 回调参数 |
| ------ | ---------- | -------- |
| close  | 关闭时触发 | -        |
