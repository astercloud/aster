# Image 图片

用于展示图片的组件，支持加载状态和错误处理。

## 基础用法

基本的图片展示。

```vue
<template>
  <Image src="https://example.com/image.jpg" alt="示例图片" />
</template>
```

## 不同尺寸

图片有多种预设尺寸。

```vue
<template>
  <Flex gap="md" align="center">
    <Image src="..." alt="小图" size="sm" />
    <Image src="..." alt="中图" size="md" />
    <Image src="..." alt="大图" size="lg" />
  </Flex>
</template>
```

## 全宽图片

图片可以占满容器宽度。

```vue
<template>
  <Image src="https://example.com/banner.jpg" alt="横幅" size="full" />
</template>
```

## 不同形状

图片支持不同的形状。

```vue
<template>
  <Flex gap="md">
    <Image src="..." alt="方形" shape="square" />
    <Image src="..." alt="圆角" shape="rounded" />
    <Image src="..." alt="圆形" shape="circle" />
  </Flex>
</template>
```

## 加载和错误状态

图片自动处理加载和错误状态。

```vue
<template>
  <!-- 加载中显示加载图标 -->
  <!-- 加载失败显示错误提示 -->
  <Image src="https://example.com/image.jpg" alt="图片" />
</template>
```

## API

### Props

| 参数  | 说明     | 类型                                | 默认值      |
| ----- | -------- | ----------------------------------- | ----------- |
| src   | 图片地址 | `string`                            | -           |
| alt   | 图片描述 | `string`                            | `''`        |
| size  | 图片尺寸 | `'sm' \| 'md' \| 'lg' \| 'full'`    | `'md'`      |
| shape | 图片形状 | `'square' \| 'rounded' \| 'circle'` | `'rounded'` |

## 示例

### 图片画廊

```vue
<template>
  <div class="grid grid-cols-3 gap-4">
    <Image v-for="(img, index) in images" :key="index" :src="img.url" :alt="img.alt" size="full" shape="rounded" />
  </div>
</template>

<script setup>
const images = [
  { url: "https://example.com/1.jpg", alt: "图片 1" },
  { url: "https://example.com/2.jpg", alt: "图片 2" },
  { url: "https://example.com/3.jpg", alt: "图片 3" },
];
</script>
```

### 用户头像

```vue
<template>
  <Image src="https://example.com/avatar.jpg" alt="用户头像" size="lg" shape="circle" />
</template>
```
