# Tooltip 工具提示

用于显示简短提示信息的组件。

## 基础用法

基本的工具提示。

```vue
<template>
  <Tooltip content="这是提示信息">
    <Button>悬停查看提示</Button>
  </Tooltip>
</template>
```

## 不同位置

工具提示可以显示在四个方向。

```vue
<template>
  <Flex gap="md" justify="center">
    <Tooltip content="顶部提示" position="top">
      <Button>上</Button>
    </Tooltip>

    <Tooltip content="右侧提示" position="right">
      <Button>右</Button>
    </Tooltip>

    <Tooltip content="底部提示" position="bottom">
      <Button>下</Button>
    </Tooltip>

    <Tooltip content="左侧提示" position="left">
      <Button>左</Button>
    </Tooltip>
  </Flex>
</template>
```

## 长文本

提示内容较长时自动换行。

```vue
<template>
  <Tooltip content="这是一段比较长的提示信息，会自动换行显示">
    <Button>长文本提示</Button>
  </Tooltip>
</template>
```

## API

### Props

| 参数     | 说明     | 类型                                     | 默认值  |
| -------- | -------- | ---------------------------------------- | ------- |
| content  | 提示内容 | `string`                                 | -       |
| position | 显示位置 | `'top' \| 'bottom' \| 'left' \| 'right'` | `'top'` |

### Slots

| 名称    | 说明           |
| ------- | -------------- |
| default | 触发提示的元素 |

## 示例

### 图标提示

```vue
<template>
  <Flex gap="md">
    <Tooltip content="发送消息">
      <Button icon="send" variant="text" />
    </Tooltip>

    <Tooltip content="上传图片">
      <Button icon="image" variant="text" />
    </Tooltip>

    <Tooltip content="语音输入">
      <Button icon="mic" variant="text" />
    </Tooltip>
  </Flex>
</template>
```

### 禁用状态说明

```vue
<template>
  <Tooltip content="请先登录">
    <Button disabled>提交</Button>
  </Tooltip>
</template>
```
