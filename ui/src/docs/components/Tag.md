# Tag 标签

用于标记和分类的标签组件。

## 基础用法

基本的标签。

```vue
<template>
  <Tag>默认标签</Tag>
</template>
```

## 不同颜色

标签有多种颜色。

```vue
<template>
  <Flex gap="sm" wrap>
    <Tag>默认</Tag>
    <Tag color="primary">主要</Tag>
    <Tag color="success">成功</Tag>
    <Tag color="warning">警告</Tag>
    <Tag color="error">错误</Tag>
  </Flex>
</template>
```

## 不同尺寸

标签有三种尺寸。

```vue
<template>
  <Flex gap="sm" align="center">
    <Tag size="sm">小标签</Tag>
    <Tag size="md">中标签</Tag>
    <Tag size="lg">大标签</Tag>
  </Flex>
</template>
```

## 可关闭

标签可以被关闭。

```vue
<template>
  <Tag closable @close="handleClose"> 可关闭标签 </Tag>
</template>

<script setup>
const handleClose = () => {
  console.log("Tag closed");
};
</script>
```

## 动态标签

动态添加和删除标签。

```vue
<template>
  <Flex gap="sm" wrap>
    <Tag v-for="tag in tags" :key="tag" closable @close="removeTag(tag)">
      {{ tag }}
    </Tag>

    <Button size="sm" @click="addTag"> + 添加标签 </Button>
  </Flex>
</template>

<script setup>
import { ref } from "vue";

const tags = ref(["标签1", "标签2", "标签3"]);

const removeTag = (tag) => {
  const index = tags.value.indexOf(tag);
  if (index > -1) {
    tags.value.splice(index, 1);
  }
};

const addTag = () => {
  tags.value.push(`标签${tags.value.length + 1}`);
};
</script>
```

## API

### Props

| 参数     | 说明       | 类型                                                          | 默认值      |
| -------- | ---------- | ------------------------------------------------------------- | ----------- |
| color    | 标签颜色   | `'default' \| 'primary' \| 'success' \| 'warning' \| 'error'` | `'default'` |
| size     | 标签尺寸   | `'sm' \| 'md' \| 'lg'`                                        | `'md'`      |
| closable | 是否可关闭 | `boolean`                                                     | `false`     |

### Events

| 事件名 | 说明           | 回调参数 |
| ------ | -------------- | -------- |
| close  | 关闭标签时触发 | -        |

### Slots

| 名称    | 说明     |
| ------- | -------- |
| default | 标签内容 |

## 示例

### 状态标签

```vue
<template>
  <Flex gap="sm">
    <Tag color="success">在线</Tag>
    <Tag color="warning">忙碌</Tag>
    <Tag color="error">离线</Tag>
  </Flex>
</template>
```

### 分类标签

```vue
<template>
  <div>
    <h3>文章标签</h3>
    <Flex gap="sm" wrap>
      <Tag color="primary">Vue 3</Tag>
      <Tag color="primary">TypeScript</Tag>
      <Tag color="primary">Tailwind CSS</Tag>
    </Flex>
  </div>
</template>
```
