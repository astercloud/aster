# Sidebar 侧边栏

用于页面侧边导航的组件。

## 基础用法

基本的侧边栏。

```vue
<template>
  <Sidebar title="导航">
    <div>侧边栏内容</div>
  </Sidebar>
</template>
```

## 可折叠

侧边栏可以折叠。

```vue
<template>
  <Sidebar title="导航" collapsible :default-collapsed="false">
    <div>侧边栏内容</div>
  </Sidebar>
</template>
```

## 右侧位置

侧边栏可以显示在右侧。

```vue
<template>
  <Sidebar title="设置" position="right">
    <div>右侧侧边栏内容</div>
  </Sidebar>
</template>
```

## 自定义头部和底部

自定义侧边栏的头部和底部。

```vue
<template>
  <Sidebar>
    <template #header>
      <div class="flex items-center gap-2">
        <Avatar alt="User" />
        <span>用户名</span>
      </div>
    </template>

    <div>侧边栏内容</div>

    <template #footer>
      <Button variant="primary" class="w-full"> 退出登录 </Button>
    </template>
  </Sidebar>
</template>
```

## API

### Props

| 参数             | 说明         | 类型                | 默认值   |
| ---------------- | ------------ | ------------------- | -------- |
| title            | 侧边栏标题   | `string`            | `''`     |
| position         | 侧边栏位置   | `'left' \| 'right'` | `'left'` |
| collapsible      | 是否可折叠   | `boolean`           | `false`  |
| defaultCollapsed | 默认是否折叠 | `boolean`           | `false`  |

### Slots

| 名称    | 说明     |
| ------- | -------- |
| header  | 头部内容 |
| default | 主体内容 |
| footer  | 底部内容 |

## 示例

### 导航侧边栏

```vue
<template>
  <Sidebar title="组件" collapsible>
    <div class="space-y-2">
      <div v-for="item in navItems" :key="item.key" class="nav-item">
        {{ item.label }}
      </div>
    </div>
  </Sidebar>
</template>

<script setup>
const navItems = [
  { key: "button", label: "Button 按钮" },
  { key: "input", label: "Input 输入框" },
  { key: "modal", label: "Modal 模态框" },
];
</script>

<style scoped>
.nav-item {
  @apply px-3 py-2 rounded-lg hover:bg-gray-100 cursor-pointer;
}
</style>
```

### 设置侧边栏

```vue
<template>
  <Sidebar title="设置" position="right">
    <template #header>
      <h3 class="font-bold">应用设置</h3>
    </template>

    <div class="space-y-4">
      <div>
        <label>主题</label>
        <Radio v-model="theme" value="light">浅色</Radio>
        <Radio v-model="theme" value="dark">深色</Radio>
      </div>

      <div>
        <label>语言</label>
        <select class="w-full">
          <option>中文</option>
          <option>English</option>
        </select>
      </div>
    </div>

    <template #footer>
      <Button variant="primary" class="w-full"> 保存设置 </Button>
    </template>
  </Sidebar>
</template>

<script setup>
import { ref } from "vue";
const theme = ref("light");
</script>
```
