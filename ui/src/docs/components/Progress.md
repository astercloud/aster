# Progress 进度条

显示操作进度的组件。

## 基础用法

基本的进度条。

```vue
<template>
  <Progress :percent="50" />
</template>
```

## 带标签

显示进度标签和百分比。

```vue
<template>
  <Progress :percent="30" label="上传中" :show-percent="true" />
</template>
```

## 不同状态

进度条有三种状态。

```vue
<template>
  <Flex direction="column" gap="md">
    <Progress :percent="30" status="normal" label="进行中" />
    <Progress :percent="100" status="success" label="已完成" />
    <Progress :percent="50" status="error" label="上传失败" />
  </Flex>
</template>
```

## 动态进度

动态更新进度。

```vue
<template>
  <div>
    <Progress :percent="progress" label="处理中" />
    <Button @click="start" class="mt-4">开始</Button>
  </div>
</template>

<script setup>
import { ref } from "vue";

const progress = ref(0);

const start = () => {
  progress.value = 0;
  const timer = setInterval(() => {
    progress.value += 10;
    if (progress.value >= 100) {
      clearInterval(timer);
    }
  }, 500);
};
</script>
```

## 不显示百分比

隐藏百分比显示。

```vue
<template>
  <Progress :percent="60" label="下载中" :show-percent="false" />
</template>
```

## API

### Props

| 参数        | 说明           | 类型                               | 默认值     |
| ----------- | -------------- | ---------------------------------- | ---------- |
| percent     | 进度百分比     | `number`                           | `0`        |
| label       | 进度标签       | `string`                           | -          |
| showPercent | 是否显示百分比 | `boolean`                          | `true`     |
| status      | 进度状态       | `'normal' \| 'success' \| 'error'` | `'normal'` |

## 示例

### 文件上传

```vue
<template>
  <div>
    <Progress :percent="uploadProgress" :status="uploadStatus" :label="uploadLabel" />
  </div>
</template>

<script setup>
import { ref, computed } from "vue";

const uploadProgress = ref(0);
const uploadStatus = computed(() => {
  if (uploadProgress.value === 100) return "success";
  if (uploadProgress.value > 0) return "normal";
  return "normal";
});

const uploadLabel = computed(() => {
  if (uploadProgress.value === 100) return "上传完成";
  if (uploadProgress.value > 0) return "上传中";
  return "准备上传";
});
</script>
```

### 任务进度

```vue
<template>
  <Flex direction="column" gap="md">
    <Progress :percent="33" label="步骤 1/3" />
    <Progress :percent="66" label="步骤 2/3" />
    <Progress :percent="100" status="success" label="步骤 3/3" />
  </Flex>
</template>
```
