# List 列表

用于展示列表数据的组件。

## 基础用法

基本的列表。

```vue
<template>
  <List :items="items" />
</template>

<script setup>
const items = ["项目 1", "项目 2", "项目 3"];
</script>
```

## 自定义项

自定义列表项的渲染。

```vue
<template>
  <List :items="users" @select="handleSelect">
    <template #default="{ item }">
      <div class="flex items-center gap-3">
        <Avatar :alt="item.name" />
        <div>
          <div class="font-semibold">{{ item.name }}</div>
          <div class="text-sm text-gray-500">{{ item.email }}</div>
        </div>
      </div>
    </template>
  </List>
</template>

<script setup>
const users = [
  { id: 1, name: "张三", email: "zhang@example.com" },
  { id: 2, name: "李四", email: "li@example.com" },
  { id: 3, name: "王五", email: "wang@example.com" },
];

const handleSelect = (user) => {
  console.log("Selected:", user);
};
</script>
```

## 带图标

列表项可以包含图标。

```vue
<template>
  <List :items="menuItems">
    <template #default="{ item }">
      <div class="flex items-center gap-2">
        <Icon :type="item.icon" />
        <span>{{ item.label }}</span>
      </div>
    </template>
  </List>
</template>

<script setup>
const menuItems = [
  { icon: "home", label: "首页" },
  { icon: "settings", label: "设置" },
  { icon: "user", label: "个人中心" },
];
</script>
```

## API

### Props

| 参数  | 说明     | 类型    | 默认值 |
| ----- | -------- | ------- | ------ |
| items | 列表数据 | `any[]` | `[]`   |

### Events

| 事件名 | 说明             | 回调参数    |
| ------ | ---------------- | ----------- |
| select | 选择列表项时触发 | `item: any` |

### Slots

| 名称    | 说明         | 参数              |
| ------- | ------------ | ----------------- |
| default | 自定义列表项 | `{ item, index }` |

## 示例

### 联系人列表

```vue
<template>
  <List :items="contacts" @select="handleContact">
    <template #default="{ item }">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <Avatar :alt="item.name" :status="item.status" />
          <div>
            <div class="font-semibold">{{ item.name }}</div>
            <div class="text-sm text-gray-500">{{ item.lastMessage }}</div>
          </div>
        </div>
        <div class="text-xs text-gray-400">
          {{ item.time }}
        </div>
      </div>
    </template>
  </List>
</template>

<script setup>
const contacts = [
  {
    id: 1,
    name: "张三",
    status: "online",
    lastMessage: "你好",
    time: "10:30",
  },
  // ...
];
</script>
```

### 设置列表

```vue
<template>
  <List :items="settings" @select="handleSetting">
    <template #default="{ item }">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <Icon :type="item.icon" />
          <span>{{ item.label }}</span>
        </div>
        <Icon type="chevron-right" />
      </div>
    </template>
  </List>
</template>

<script setup>
const settings = [
  { key: "account", label: "账号设置", icon: "user" },
  { key: "privacy", label: "隐私设置", icon: "lock" },
  { key: "notification", label: "通知设置", icon: "bell" },
];
</script>
```
