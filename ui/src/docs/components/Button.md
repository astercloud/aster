# Button 按钮

按钮用于触发操作。

## 基础用法

基本的按钮用法。

```vue
<template>
  <Button>默认按钮</Button>
</template>
```

## 按钮类型

按钮有三种类型：主要按钮、次要按钮和文本按钮。

```vue
<template>
  <Flex gap="md">
    <Button variant="primary">主要按钮</Button>
    <Button variant="secondary">次要按钮</Button>
    <Button variant="text">文本按钮</Button>
  </Flex>
</template>
```

## 按钮尺寸

按钮有三种尺寸：小、中、大。

```vue
<template>
  <Flex gap="md" align="center">
    <Button size="sm">小按钮</Button>
    <Button size="md">中按钮</Button>
    <Button size="lg">大按钮</Button>
  </Flex>
</template>
```

## 带图标的按钮

按钮可以配置图标。

```vue
<template>
  <Flex gap="md">
    <Button icon="send">发送</Button>
    <Button icon="image" variant="secondary">上传图片</Button>
    <Button icon="mic" variant="text">语音</Button>
  </Flex>
</template>
```

## 禁用状态

按钮可以被禁用。

```vue
<template>
  <Flex gap="md">
    <Button disabled>禁用按钮</Button>
    <Button variant="primary" disabled>主要按钮</Button>
  </Flex>
</template>
```

## API

### Props

| 参数     | 说明     | 类型                                 | 默认值      |
| -------- | -------- | ------------------------------------ | ----------- |
| variant  | 按钮类型 | `'primary' \| 'secondary' \| 'text'` | `'primary'` |
| size     | 按钮尺寸 | `'sm' \| 'md' \| 'lg'`               | `'md'`      |
| icon     | 图标名称 | `string`                             | -           |
| disabled | 是否禁用 | `boolean`                            | `false`     |

### Events

| 事件名 | 说明           | 回调参数 |
| ------ | -------------- | -------- |
| click  | 点击按钮时触发 | -        |

### Slots

| 名称    | 说明     |
| ------- | -------- |
| default | 按钮内容 |
