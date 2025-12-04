# Icon 图标

显示图标的组件。

## 基础用法

基本的图标显示。

```vue
<template>
  <Icon type="send" />
</template>
```

## 不同尺寸

图标有三种尺寸。

```vue
<template>
  <Flex gap="md" align="center">
    <Icon type="send" size="sm" />
    <Icon type="send" size="md" />
    <Icon type="send" size="lg" />
  </Flex>
</template>
```

## 可用图标

内置的图标类型：

```vue
<template>
  <Flex gap="md" wrap>
    <Icon type="send" />
    <Icon type="image" />
    <Icon type="mic" />
    <Icon type="attach" />
    <Icon type="emoji" />
    <Icon type="more" />
    <Icon type="close" />
    <Icon type="check" />
    <Icon type="loading" />
  </Flex>
</template>
```

## 加载图标

显示加载动画。

```vue
<template>
  <Icon type="loading" />
  <!-- 自动旋转动画 -->
</template>
```

## 自定义颜色

通过 CSS 自定义图标颜色。

```vue
<template>
  <Icon type="send" class="text-blue-500" />
  <Icon type="check" class="text-green-500" />
  <Icon type="close" class="text-red-500" />
</template>
```

## API

### Props

| 参数 | 说明     | 类型                   | 默认值 |
| ---- | -------- | ---------------------- | ------ |
| type | 图标类型 | `IconType`             | -      |
| size | 图标尺寸 | `'sm' \| 'md' \| 'lg'` | `'md'` |

### IconType

```typescript
type IconType =
  | "send" // 发送
  | "image" // 图片
  | "mic" // 麦克风
  | "attach" // 附件
  | "emoji" // 表情
  | "more" // 更多
  | "close" // 关闭
  | "check" // 勾选
  | "loading"; // 加载
```

## 示例

### 按钮中使用

```vue
<template>
  <Button>
    <Icon type="send" />
    发送消息
  </Button>
</template>
```

### 状态指示

```vue
<template>
  <Flex gap="md">
    <div class="flex items-center gap-2">
      <Icon type="loading" class="text-blue-500" />
      <span>加载中...</span>
    </div>

    <div class="flex items-center gap-2">
      <Icon type="check" class="text-green-500" />
      <span>已完成</span>
    </div>
  </Flex>
</template>
```
