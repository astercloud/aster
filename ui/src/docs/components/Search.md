# Search 搜索框

用于搜索内容的输入框组件。

## 基础用法

基本的搜索框。

```vue
<template>
  <Search v-model="searchValue" placeholder="搜索..." @search="handleSearch" />
</template>

<script setup>
import { ref } from "vue";

const searchValue = ref("");

const handleSearch = (value) => {
  console.log("Search:", value);
};
</script>
```

## 自定义占位符

自定义搜索框的占位符文本。

```vue
<template>
  <Search v-model="value" placeholder="搜索组件、文档..." @search="handleSearch" />
</template>
```

## 实时搜索

监听输入变化进行实时搜索。

```vue
<template>
  <Search v-model="searchValue" @update:modelValue="handleInput" />
</template>

<script setup>
import { ref } from "vue";

const searchValue = ref("");

const handleInput = (value) => {
  // 实时搜索
  console.log("Input:", value);
};
</script>
```

## 清除按钮

搜索框自动显示清除按钮。

```vue
<template>
  <Search v-model="value" />
  <!-- 当有内容时自动显示清除按钮 -->
</template>
```

## API

### Props

| 参数        | 说明   | 类型     | 默认值      |
| ----------- | ------ | -------- | ----------- |
| modelValue  | 绑定值 | `string` | `''`        |
| placeholder | 占位符 | `string` | `'搜索...'` |

### Events

| 事件名            | 说明                     | 回调参数        |
| ----------------- | ------------------------ | --------------- |
| update:modelValue | 值改变时触发             | `value: string` |
| search            | 按下回车或点击搜索时触发 | `value: string` |

## 示例

### 搜索组件

```vue
<template>
  <div>
    <Search v-model="query" placeholder="搜索组件..." @search="performSearch" />

    <div v-if="results.length > 0" class="results">
      <div v-for="item in results" :key="item.id">
        {{ item.name }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";

const query = ref("");
const results = ref([]);

const performSearch = (value) => {
  // 执行搜索逻辑
  results.value = searchComponents(value);
};

const searchComponents = (query) => {
  // 搜索实现
  return [];
};
</script>
```

### 带防抖的搜索

```vue
<template>
  <Search v-model="searchValue" @update:modelValue="debouncedSearch" />
</template>

<script setup>
import { ref } from "vue";
import { debounce } from "lodash-es";

const searchValue = ref("");

const debouncedSearch = debounce((value) => {
  console.log("Searching:", value);
  // 执行搜索
}, 300);
</script>
```
