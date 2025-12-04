# Dropdown 下拉菜单

用于显示下拉菜单的组件。

## 基础用法

基本的下拉菜单。

```vue
<template>
  <Dropdown :items="items" @select="handleSelect" />
</template>

<script setup>
const items = [
  { key: "1", label: "选项 1" },
  { key: "2", label: "选项 2" },
  { key: "3", label: "选项 3" },
];

const handleSelect = (item) => {
  console.log("Selected:", item);
};
</script>
```

## 自定义触发器

自定义触发下拉菜单的元素。

```vue
<template>
  <Dropdown :items="items" @select="handleSelect">
    <template #trigger>
      <Button>
        更多操作
        <Icon type="more" />
      </Button>
    </template>
  </Dropdown>
</template>
```

## 带图标

菜单项可以包含图标。

```vue
<template>
  <Dropdown :items="items" @select="handleSelect" />
</template>

<script setup>
const items = [
  { key: "edit", label: "编辑", icon: "edit" },
  { key: "copy", label: "复制", icon: "copy" },
  { key: "delete", label: "删除", icon: "delete" },
];
</script>
```

## API

### Props

| 参数  | 说明       | 类型             | 默认值   |
| ----- | ---------- | ---------------- | -------- |
| items | 菜单项列表 | `DropdownItem[]` | `[]`     |
| label | 触发器文本 | `string`         | `'选择'` |

### DropdownItem 类型

```typescript
interface DropdownItem {
  key: string; // 唯一标识
  label: string; // 显示文本
  icon?: string; // 图标（可选）
}
```

### Events

| 事件名 | 说明             | 回调参数             |
| ------ | ---------------- | -------------------- |
| select | 选择菜单项时触发 | `item: DropdownItem` |

### Slots

| 名称    | 说明         |
| ------- | ------------ |
| trigger | 自定义触发器 |

## 示例

### 操作菜单

```vue
<template>
  <Dropdown :items="actions" @select="handleAction">
    <template #trigger>
      <Button variant="text">
        <Icon type="more" />
      </Button>
    </template>
  </Dropdown>
</template>

<script setup>
const actions = [
  { key: "edit", label: "编辑", icon: "edit" },
  { key: "share", label: "分享", icon: "share" },
  { key: "delete", label: "删除", icon: "delete" },
];

const handleAction = (action) => {
  console.log("Action:", action.key);
};
</script>
```

### 用户菜单

```vue
<template>
  <Dropdown :items="userMenu" @select="handleUserAction">
    <template #trigger>
      <Avatar alt="User" />
    </template>
  </Dropdown>
</template>

<script setup>
const userMenu = [
  { key: "profile", label: "个人资料" },
  { key: "settings", label: "设置" },
  { key: "logout", label: "退出登录" },
];
</script>
```
