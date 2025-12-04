# Checkbox 复选框

用于在多个选项中进行多选。

## 基础用法

基本的复选框。

```vue
<template>
  <Checkbox v-model="checked"> 同意用户协议 </Checkbox>
</template>

<script setup>
import { ref } from "vue";
const checked = ref(false);
</script>
```

## 禁用状态

禁用复选框。

```vue
<template>
  <Flex direction="column" gap="md">
    <Checkbox v-model="checked1" disabled> 禁用未选中 </Checkbox>
    <Checkbox v-model="checked2" disabled> 禁用已选中 </Checkbox>
  </Flex>
</template>

<script setup>
import { ref } from "vue";
const checked1 = ref(false);
const checked2 = ref(true);
</script>
```

## 复选框组

多个复选框组合使用。

```vue
<template>
  <Flex direction="column" gap="md">
    <Checkbox v-model="options.vue">Vue 3</Checkbox>
    <Checkbox v-model="options.react">React</Checkbox>
    <Checkbox v-model="options.angular">Angular</Checkbox>
  </Flex>

  <div class="mt-4">已选择: {{ selectedOptions }}</div>
</template>

<script setup>
import { ref, computed } from "vue";

const options = ref({
  vue: true,
  react: false,
  angular: false,
});

const selectedOptions = computed(() => {
  return Object.keys(options.value)
    .filter((key) => options.value[key])
    .join(", ");
});
</script>
```

## API

### Props

| 参数       | 说明     | 类型      | 默认值  |
| ---------- | -------- | --------- | ------- |
| modelValue | 绑定值   | `boolean` | `false` |
| disabled   | 是否禁用 | `boolean` | `false` |

### Events

| 事件名            | 说明         | 回调参数         |
| ----------------- | ------------ | ---------------- |
| update:modelValue | 值改变时触发 | `value: boolean` |

### Slots

| 名称    | 说明           |
| ------- | -------------- |
| default | 复选框标签内容 |

## 示例

### 全选功能

```vue
<template>
  <div>
    <Checkbox v-model="checkAll" @update:modelValue="handleCheckAll"> 全选 </Checkbox>

    <Divider />

    <Flex direction="column" gap="sm">
      <Checkbox v-for="item in items" :key="item.id" v-model="item.checked">
        {{ item.label }}
      </Checkbox>
    </Flex>
  </div>
</template>

<script setup>
import { ref, computed } from "vue";

const items = ref([
  { id: 1, label: "选项 1", checked: false },
  { id: 2, label: "选项 2", checked: false },
  { id: 3, label: "选项 3", checked: false },
]);

const checkAll = computed({
  get: () => items.value.every((item) => item.checked),
  set: (value) => {
    items.value.forEach((item) => {
      item.checked = value;
    });
  },
});

const handleCheckAll = (value) => {
  items.value.forEach((item) => {
    item.checked = value;
  });
};
</script>
```
