# Modal 模态框

用于显示模态对话框。

## 基础用法

基本的模态框。

```vue
<template>
  <Button @click="visible = true">打开模态框</Button>

  <Modal v-model:visible="visible" title="提示">
    <p>这是模态框的内容</p>
  </Modal>
</template>

<script setup>
import { ref } from "vue";
const visible = ref(false);
</script>
```

## 不同尺寸

模态框有四种尺寸。

```vue
<template>
  <Flex gap="md">
    <Button @click="showModal('sm')">小</Button>
    <Button @click="showModal('md')">中</Button>
    <Button @click="showModal('lg')">大</Button>
    <Button @click="showModal('xl')">超大</Button>
  </Flex>

  <Modal v-model:visible="visible" :size="size" title="模态框">
    <p>这是 {{ size }} 尺寸的模态框</p>
  </Modal>
</template>

<script setup>
import { ref } from "vue";
const visible = ref(false);
const size = ref("md");

const showModal = (s) => {
  size.value = s;
  visible.value = true;
};
</script>
```

## 自定义页脚

自定义模态框页脚。

```vue
<template>
  <Button @click="visible = true">打开</Button>

  <Modal v-model:visible="visible" title="确认操作">
    <p>确定要执行此操作吗？</p>

    <template #footer>
      <Flex justify="end" gap="md">
        <Button variant="secondary" @click="visible = false"> 取消 </Button>
        <Button variant="primary" @click="handleConfirm"> 确定 </Button>
      </Flex>
    </template>
  </Modal>
</template>

<script setup>
import { ref } from "vue";
const visible = ref(false);

const handleConfirm = () => {
  console.log("Confirmed");
  visible.value = false;
};
</script>
```

## 禁止点击遮罩关闭

设置 `closeOnOverlay` 为 `false`。

```vue
<template>
  <Modal v-model:visible="visible" title="重要提示" :close-on-overlay="false">
    <p>此模态框不能通过点击遮罩关闭</p>
  </Modal>
</template>
```

## API

### Props

| 参数           | 说明             | 类型                           | 默认值  |
| -------------- | ---------------- | ------------------------------ | ------- |
| visible        | 是否显示         | `boolean`                      | `false` |
| title          | 标题             | `string`                       | `''`    |
| size           | 尺寸             | `'sm' \| 'md' \| 'lg' \| 'xl'` | `'md'`  |
| closeOnOverlay | 点击遮罩是否关闭 | `boolean`                      | `true`  |

### Events

| 事件名         | 说明               | 回调参数         |
| -------------- | ------------------ | ---------------- |
| update:visible | 显示状态改变时触发 | `value: boolean` |
| close          | 关闭时触发         | -                |

### Slots

| 名称    | 说明       |
| ------- | ---------- |
| default | 模态框内容 |
| title   | 自定义标题 |
| footer  | 自定义页脚 |
