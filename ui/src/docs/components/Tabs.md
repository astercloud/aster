# Tabs 标签页

用于内容分类展示的标签页组件。

## 基础用法

基本的标签页。

```vue
<template>
  <Tabs :tabs="tabs" v-model="activeTab">
    <div v-if="activeTab === 'tab1'">标签一的内容</div>
    <div v-if="activeTab === 'tab2'">标签二的内容</div>
    <div v-if="activeTab === 'tab3'">标签三的内容</div>
  </Tabs>
</template>

<script setup>
import { ref } from "vue";

const activeTab = ref("tab1");

const tabs = [
  { key: "tab1", label: "标签一" },
  { key: "tab2", label: "标签二" },
  { key: "tab3", label: "标签三" },
];
</script>
```

## 监听切换

监听标签页切换事件。

```vue
<template>
  <Tabs :tabs="tabs" v-model="activeTab" @change="handleChange">
    <!-- 内容 -->
  </Tabs>
</template>

<script setup>
const handleChange = (key) => {
  console.log("Tab changed to:", key);
};
</script>
```

## 动态标签

动态生成标签页。

```vue
<template>
  <Tabs :tabs="dynamicTabs" v-model="activeTab">
    <div v-for="tab in dynamicTabs" :key="tab.key">
      <div v-if="activeTab === tab.key">{{ tab.label }} 的内容</div>
    </div>
  </Tabs>
</template>

<script setup>
import { ref } from "vue";

const activeTab = ref("tab1");
const dynamicTabs = ref([
  { key: "tab1", label: "标签一" },
  { key: "tab2", label: "标签二" },
]);

// 可以动态添加标签
const addTab = () => {
  const newKey = `tab${dynamicTabs.value.length + 1}`;
  dynamicTabs.value.push({
    key: newKey,
    label: `标签${dynamicTabs.value.length + 1}`,
  });
};
</script>
```

## API

### Props

| 参数       | 说明               | 类型     | 默认值 |
| ---------- | ------------------ | -------- | ------ |
| tabs       | 标签页配置         | `Tab[]`  | `[]`   |
| modelValue | 当前激活的标签 key | `string` | -      |

### Tab 类型

```typescript
interface Tab {
  key: string; // 标签唯一标识
  label: string; // 标签显示文本
}
```

### Events

| 事件名            | 说明               | 回调参数      |
| ----------------- | ------------------ | ------------- |
| update:modelValue | 激活标签改变时触发 | `key: string` |
| change            | 切换标签时触发     | `key: string` |

### Slots

| 名称    | 说明       |
| ------- | ---------- |
| default | 标签页内容 |

## 示例

### 设置页面

```vue
<template>
  <Tabs :tabs="settingTabs" v-model="activeTab">
    <div v-if="activeTab === 'general'">
      <h3>通用设置</h3>
      <!-- 通用设置表单 -->
    </div>

    <div v-if="activeTab === 'security'">
      <h3>安全设置</h3>
      <!-- 安全设置表单 -->
    </div>

    <div v-if="activeTab === 'notification'">
      <h3>通知设置</h3>
      <!-- 通知设置表单 -->
    </div>
  </Tabs>
</template>

<script setup>
import { ref } from "vue";

const activeTab = ref("general");

const settingTabs = [
  { key: "general", label: "通用" },
  { key: "security", label: "安全" },
  { key: "notification", label: "通知" },
];
</script>
```

### 数据展示

```vue
<template>
  <Tabs :tabs="dataTabs" v-model="activeTab">
    <div v-if="activeTab === 'overview'">
      <!-- 概览数据 -->
    </div>

    <div v-if="activeTab === 'details'">
      <!-- 详细数据 -->
    </div>

    <div v-if="activeTab === 'history'">
      <!-- 历史记录 -->
    </div>
  </Tabs>
</template>
```
