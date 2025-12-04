# Radio 单选框

用于在多个选项中进行单选。

## 基础用法

基本的单选框。

```vue
<template>
  <Flex gap="md">
    <Radio v-model="value" value="a" name="demo">选项 A</Radio>
    <Radio v-model="value" value="b" name="demo">选项 B</Radio>
    <Radio v-model="value" value="c" name="demo">选项 C</Radio>
  </Flex>

  <div class="mt-4">已选择: {{ value }}</div>
</template>

<script setup>
import { ref } from "vue";
const value = ref("a");
</script>
```

## 禁用状态

禁用单选框。

```vue
<template>
  <Flex gap="md">
    <Radio v-model="value" value="a" name="demo2">正常</Radio>
    <Radio v-model="value" value="b" name="demo2" disabled> 禁用 </Radio>
  </Flex>
</template>
```

## 垂直排列

垂直排列单选框。

```vue
<template>
  <Flex direction="column" gap="md">
    <Radio v-model="value" value="1" name="demo3">选项 1</Radio>
    <Radio v-model="value" value="2" name="demo3">选项 2</Radio>
    <Radio v-model="value" value="3" name="demo3">选项 3</Radio>
  </Flex>
</template>
```

## API

### Props

| 参数       | 说明           | 类型      | 默认值  |
| ---------- | -------------- | --------- | ------- |
| modelValue | 绑定值         | `any`     | -       |
| value      | 单选框的值     | `any`     | -       |
| name       | 原生 name 属性 | `string`  | -       |
| disabled   | 是否禁用       | `boolean` | `false` |

### Events

| 事件名            | 说明         | 回调参数     |
| ----------------- | ------------ | ------------ |
| update:modelValue | 值改变时触发 | `value: any` |

### Slots

| 名称    | 说明           |
| ------- | -------------- |
| default | 单选框标签内容 |

## 示例

### 配置选择

```vue
<template>
  <div>
    <h3>选择主题</h3>
    <Flex direction="column" gap="md">
      <Radio v-model="theme" value="light" name="theme"> 浅色主题 </Radio>
      <Radio v-model="theme" value="dark" name="theme"> 深色主题 </Radio>
      <Radio v-model="theme" value="auto" name="theme"> 跟随系统 </Radio>
    </Flex>
  </div>
</template>

<script setup>
import { ref } from "vue";
const theme = ref("auto");
</script>
```

### 带描述的选项

```vue
<template>
  <Flex direction="column" gap="md">
    <Radio v-model="plan" value="free" name="plan">
      <div>
        <div class="font-semibold">免费版</div>
        <div class="text-sm text-gray-500">基础功能</div>
      </div>
    </Radio>

    <Radio v-model="plan" value="pro" name="plan">
      <div>
        <div class="font-semibold">专业版</div>
        <div class="text-sm text-gray-500">全部功能</div>
      </div>
    </Radio>
  </Flex>
</template>

<script setup>
import { ref } from "vue";
const plan = ref("free");
</script>
```
