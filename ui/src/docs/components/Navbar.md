# Navbar 导航栏

用于页面顶部导航的组件。

## 基础用法

基本的导航栏。

```vue
<template>
  <Navbar title="Aster Agent" />
</template>
```

## 自定义品牌区

自定义左侧品牌区域。

```vue
<template>
  <Navbar>
    <template #brand>
      <div class="flex items-center gap-2">
        <img src="/logo.png" class="w-8 h-8" />
        <span class="font-bold">My App</span>
      </div>
    </template>
  </Navbar>
</template>
```

## 添加菜单

在中间添加导航菜单。

```vue
<template>
  <Navbar title="Aster Agent">
    <template #menu>
      <a href="#home" class="nav-link">首页</a>
      <a href="#docs" class="nav-link">文档</a>
      <a href="#about" class="nav-link">关于</a>
    </template>
  </Navbar>
</template>

<style scoped>
.nav-link {
  @apply text-gray-600 hover:text-gray-900 transition-colors;
}
</style>
```

## 添加操作按钮

在右侧添加操作按钮。

```vue
<template>
  <Navbar title="Aster Agent">
    <template #actions>
      <Search v-model="search" placeholder="搜索..." />
      <Button variant="primary">登录</Button>
    </template>
  </Navbar>
</template>

<script setup>
import { ref } from "vue";
const search = ref("");
</script>
```

## 完整示例

包含所有插槽的完整导航栏。

```vue
<template>
  <Navbar>
    <template #brand>
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 bg-blue-500 rounded-lg"></div>
        <span class="text-xl font-bold">ChatUI</span>
      </div>
    </template>

    <template #menu>
      <a href="#" class="nav-link">组件</a>
      <a href="#" class="nav-link">文档</a>
      <a href="#" class="nav-link">示例</a>
    </template>

    <template #actions>
      <Button variant="text">GitHub</Button>
      <Button variant="primary">开始使用</Button>
    </template>
  </Navbar>
</template>
```

## API

### Props

| 参数  | 说明       | 类型     | 默认值          |
| ----- | ---------- | -------- | --------------- |
| title | 导航栏标题 | `string` | `'Aster Agent'` |

### Slots

| 名称    | 说明         |
| ------- | ------------ |
| brand   | 左侧品牌区域 |
| menu    | 中间菜单区域 |
| actions | 右侧操作区域 |

## 示例

### 响应式导航栏

```vue
<template>
  <Navbar>
    <template #brand>
      <span class="font-bold">My App</span>
    </template>

    <template #menu>
      <div class="hidden md:flex gap-6">
        <a href="#">首页</a>
        <a href="#">产品</a>
        <a href="#">文档</a>
      </div>

      <Button class="md:hidden" variant="text">
        <Icon type="menu" />
      </Button>
    </template>

    <template #actions>
      <Avatar alt="User" />
    </template>
  </Navbar>
</template>
```
