# Input 输入框

用于接收用户输入的文本。

## 基础用法

基本的输入框。

```vue
<template>
  <Input v-model="value" placeholder="请输入内容" />
</template>

<script setup>
import { ref } from "vue";
const value = ref("");
</script>
```

## 带标签

输入框可以带标签。

```vue
<template>
  <Input v-model="username" label="用户名" placeholder="请输入用户名" />
</template>
```

## 不同类型

支持多种输入类型。

```vue
<template>
  <Flex direction="column" gap="md">
    <Input v-model="text" type="text" label="文本" />
    <Input v-model="password" type="password" label="密码" />
    <Input v-model="email" type="email" label="邮箱" />
    <Input v-model="number" type="number" label="数字" />
  </Flex>
</template>
```

## 错误提示

显示错误信息。

```vue
<template>
  <Input v-model="value" label="用户名" error="用户名不能为空" />
</template>
```

## 禁用状态

禁用输入框。

```vue
<template>
  <Input v-model="value" label="禁用输入框" disabled />
</template>
```

## API

### Props

| 参数        | 说明       | 类型                                          | 默认值   |
| ----------- | ---------- | --------------------------------------------- | -------- |
| modelValue  | 绑定值     | `string \| number`                            | -        |
| type        | 输入框类型 | `'text' \| 'password' \| 'email' \| 'number'` | `'text'` |
| label       | 标签文本   | `string`                                      | -        |
| placeholder | 占位符     | `string`                                      | -        |
| disabled    | 是否禁用   | `boolean`                                     | `false`  |
| error       | 错误信息   | `string`                                      | -        |

### Events

| 事件名            | 说明           | 回调参数                  |
| ----------------- | -------------- | ------------------------- |
| update:modelValue | 值改变时触发   | `value: string \| number` |
| blur              | 失去焦点时触发 | -                         |
| focus             | 获得焦点时触发 | -                         |
