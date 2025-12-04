# Avatar 头像

用于展示用户或 Agent 的头像。

## 基础用法

基本的头像展示。

```vue
<template>
  <Avatar alt="User" />
</template>
```

## 不同尺寸

头像有五种尺寸可选。

```vue
<template>
  <Flex gap="md" align="center">
    <Avatar alt="U" size="xs" />
    <Avatar alt="S" size="sm" />
    <Avatar alt="M" size="md" />
    <Avatar alt="L" size="lg" />
    <Avatar alt="X" size="xl" />
  </Flex>
</template>
```

## 状态指示

头像可以显示在线状态。

```vue
<template>
  <Flex gap="md" align="center">
    <Avatar alt="在线" size="md" status="online" />
    <Avatar alt="忙碌" size="md" status="busy" />
    <Avatar alt="离线" size="md" status="offline" />
  </Flex>
</template>
```

## 自定义图片

使用自定义图片作为头像。

```vue
<template>
  <Avatar src="https://example.com/avatar.jpg" alt="User Name" size="lg" />
</template>
```

## 形状

头像支持圆形和方形。

```vue
<template>
  <Flex gap="md">
    <Avatar alt="圆形" shape="circle" />
    <Avatar alt="方形" shape="square" />
  </Flex>
</template>
```

## API

### Props

| 参数   | 说明                   | 类型                                   | 默认值     |
| ------ | ---------------------- | -------------------------------------- | ---------- |
| src    | 头像图片 URL           | `string`                               | -          |
| alt    | 图片描述，也用作占位符 | `string`                               | `''`       |
| size   | 头像尺寸               | `'xs' \| 'sm' \| 'md' \| 'lg' \| 'xl'` | `'md'`     |
| shape  | 头像形状               | `'circle' \| 'square'`                 | `'circle'` |
| status | 在线状态               | `'online' \| 'offline' \| 'busy'`      | -          |
