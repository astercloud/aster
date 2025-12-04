# Popover 气泡卡片

点击触发的弹出内容卡片，支持四个方向定位。

## 基础用法

点击按钮显示气泡卡片内容。

```vue
<template>
  <Popover>
    <template #trigger>
      <Button>点击显示</Button>
    </template>
    <div class="p-4">
      <h3 class="font-semibold mb-2">气泡内容</h3>
      <p class="text-gray-600 dark:text-gray-400">这是气泡卡片中的内容</p>
    </div>
  </Popover>
</template>

<script setup>
import { Popover, Button } from "@/components/ChatUI";
</script>
```

## 不同位置

支持上、下、左、右四个方向的定位。

```vue
<template>
  <div class="grid grid-cols-2 gap-4">
    <div class="text-center">
      <Popover position="top">
        <template #trigger>
          <Button variant="secondary">上方显示</Button>
        </template>
        <div class="p-3">
          <p class="text-sm">内容显示在触发器上方</p>
        </div>
      </Popover>
    </div>

    <div class="text-center">
      <Popover position="bottom">
        <template #trigger>
          <Button variant="secondary">下方显示</Button>
        </template>
        <div class="p-3">
          <p class="text-sm">内容显示在触发器下方</p>
        </div>
      </Popover>
    </div>

    <div class="text-center">
      <Popover position="left">
        <template #trigger>
          <Button variant="secondary">左侧显示</Button>
        </template>
        <div class="p-3">
          <p class="text-sm">内容显示在触发器左侧</p>
        </div>
      </Popover>
    </div>

    <div class="text-center">
      <Popover position="right">
        <template #trigger>
          <Button variant="secondary">右侧显示</Button>
        </template>
        <div class="p-3">
          <p class="text-sm">内容显示在触发器右侧</p>
        </div>
      </Popover>
    </div>
  </div>
</template>

<script setup>
import { Popover, Button } from "@/components/ChatUI";
</script>
```

## 复杂内容

在气泡卡片中放置复杂的交互内容。

```vue
<template>
  <Popover position="bottom">
    <template #trigger>
      <Button icon="settings">设置</Button>
    </template>
    <div class="p-4 w-64">
      <h3 class="font-semibold mb-4">用户设置</h3>

      <div class="space-y-4">
        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">主题</label>
          <select class="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800">
            <option>浅色</option>
            <option>深色</option>
            <option>跟随系统</option>
          </select>
        </div>

        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">通知</label>
          <div class="mt-1">
            <label class="flex items-center">
              <input type="checkbox" class="mr-2" checked />
              <span class="text-sm">启用桌面通知</span>
            </label>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <Button variant="text" size="sm">取消</Button>
          <Button size="sm">保存</Button>
        </div>
      </div>
    </div>
  </Popover>
</template>

<script setup>
import { Popover, Button } from "@/components/ChatUI";
</script>
```

## 菜单样式

创建下拉菜单样式的气泡卡片。

```vue
<template>
  <Popover position="bottom">
    <template #trigger>
      <Button icon="menu" variant="text">菜单</Button>
    </template>
    <div class="py-2 w-48">
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 个人资料 </a>
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 设置 </a>
      <hr class="my-1 border-gray-200 dark:border-gray-600" />
      <a href="#" class="block px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700"> 退出登录 </a>
    </div>
  </Popover>
</template>

<script setup>
import { Popover, Button } from "@/components/ChatUI";
</script>
```

## 用户头像菜单

用户头像点击后显示的菜单选项。

```vue
<template>
  <Popover position="bottom">
    <template #trigger>
      <Avatar alt="User" src="/user-avatar.jpg" class="cursor-pointer hover:ring-2 hover:ring-blue-500" />
    </template>
    <div class="py-2 w-56">
      <div class="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
        <p class="text-sm font-medium text-gray-900 dark:text-white">张三</p>
        <p class="text-sm text-gray-500 dark:text-gray-400">user@example.com</p>
      </div>
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 查看资料 </a>
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 账户设置 </a>
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 帮助中心 </a>
      <hr class="my-1 border-gray-200 dark:border-gray-700" />
      <a href="#" class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"> 退出登录 </a>
    </div>
  </Popover>
</template>

<script setup>
import { Popover, Avatar } from "@/components/ChatUI";
</script>
```

## API

### Props

| 参数     | 说明             | 类型                                     | 默认值     |
| -------- | ---------------- | ---------------------------------------- | ---------- |
| position | 气泡卡片显示位置 | `'top' \| 'bottom' \| 'left' \| 'right'` | `'bottom'` |

### Slots

| 名称    | 说明                   |
| ------- | ---------------------- |
| trigger | 触发气泡卡片显示的内容 |
| default | 气泡卡片中的内容       |

### 使用说明

#### 关闭行为

- 点击气泡卡片外部区域会自动关闭
- 点击气泡卡片内部不会关闭
- 再次点击触发器会切换显示状态

#### 定位系统

- 使用 Teleport 将内容渲染到 body 下，避免 z-index 层级问题
- 自动计算相对于触发器的位置
- 支持动态位置切换

#### 样式定制

可以通过覆盖 CSS 类名来自定义样式：

```css
/* 自定义气泡卡片样式 */
.popover-content {
  @apply bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-xl;
}

/* 自定义遮罩层样式 */
.popover-overlay {
  @apply z-50;
}

/* 自定义不同位置样式 */
.popover-top {
  @apply mb-2;
}

.popover-bottom {
  @apply mt-2;
}
```

## 使用场景

- **下拉菜单**: 用户菜单、操作菜单等
- **设置面板**: 快速设置和配置选项
- **信息提示**: 显示详细信息或帮助内容
- **确认对话框**: 简单的确认操作
- **筛选器**: 数据筛选和排序选项

## 注意事项

- 气泡卡片内容建议控制在合理的尺寸内
- 在移动设备上考虑触摸体验
- 避免在气泡卡片中放置过多的表单元素
- 确保触发器和气泡卡片有良好的视觉关联性
