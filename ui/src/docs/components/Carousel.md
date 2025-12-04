# Carousel 轮播图

用于展示图片、卡片或其他内容的轮播组件，支持自动播放、手动切换和导航指示。

## 基础用法

基本的轮播图用法，显示一组项目并支持手动切换。

```vue
<template>
  <Carousel :items="slides">
    <template #default="{ item }">
      <img :src="item.image" :alt="item.title" class="w-full h-64 object-cover" />
    </template>
  </Carousel>
</template>

<script setup>
import { Carousel } from "@/components/ChatUI";

const slides = [
  { image: "/slide1.jpg", title: "Slide 1" },
  { image: "/slide2.jpg", title: "Slide 2" },
  { image: "/slide3.jpg", title: "Slide 3" },
];
</script>
```

## 自定义内容

可以自定义轮播项的内容，使用插槽获取当前项和索引信息。

```vue
<template>
  <Carousel :items="cards" :show-dots="false">
    <template #default="{ item, index }">
      <div class="p-6 bg-white dark:bg-gray-800 rounded-lg shadow-lg">
        <h3 class="text-lg font-semibold mb-2">{{ item.title }}</h3>
        <p class="text-gray-600 dark:text-gray-400">{{ item.description }}</p>
        <div class="mt-4 text-sm text-gray-500">第 {{ index + 1 }} 项</div>
      </div>
    </template>
  </Carousel>
</template>

<script setup>
import { Carousel } from "@/components/ChatUI";

const cards = [
  {
    title: "功能特性",
    description: "支持自动播放、手动切换、自定义内容等功能",
  },
  {
    title: "响应式设计",
    description: "适配不同屏幕尺寸，在移动端也能正常使用",
  },
  {
    title: "易于集成",
    description: "简单的 API 设计，轻松集成到现有项目中",
  },
];
</script>
```

## 隐藏导航箭头

可以隐藏左右导航箭头，仅使用底部指示点进行导航。

```vue
<template>
  <Carousel :items="items" :show-arrows="false">
    <template #default="{ item }">
      <div class="h-48 flex items-center justify-center bg-gradient-to-br from-blue-500 to-purple-600 text-white text-xl font-bold rounded-lg">
        {{ item }}
      </div>
    </template>
  </Carousel>
</template>

<script setup>
import { Carousel } from "@/components/ChatUI";

const items = ["First Slide", "Second Slide", "Third Slide"];
</script>
```

## 隐藏指示点

对于一些特殊的用例，可以隐藏底部的圆点指示器。

```vue
<template>
  <Carousel :items="gallery" :show-dots="false">
    <template #default="{ item }">
      <img :src="item.url" :alt="item.caption" class="w-full h-80 object-cover" />
      <div class="absolute bottom-0 left-0 right-0 bg-black/50 text-white p-4">
        {{ item.caption }}
      </div>
    </template>
  </Carousel>
</template>

<script setup>
import { Carousel } from "@/components/ChatUI";

const gallery = [
  { url: "/photo1.jpg", caption: "美丽的风景" },
  { url: "/photo2.jpg", caption: "城市夜景" },
];
</script>
```

## 自动播放

启用自动播放功能，可以设置自动切换的时间间隔。

```vue
<template>
  <Carousel :items="banners" :autoplay="true" :interval="2000">
    <template #default="{ item }">
      <div class="relative">
        <img :src="item.image" :alt="item.title" class="w-full h-96 object-cover" />
        <div class="absolute inset-0 bg-black/30 flex items-center justify-center">
          <div class="text-center text-white">
            <h2 class="text-3xl font-bold mb-2">{{ item.title }}</h2>
            <p class="text-lg">{{ item.subtitle }}</p>
          </div>
        </div>
      </div>
    </template>
  </Carousel>
</template>

<script setup>
import { Carousel } from "@/components/ChatUI";

const banners = [
  {
    title: "产品发布",
    subtitle: "全新功能上线",
    image: "/banner1.jpg",
  },
  {
    title: "用户活动",
    subtitle: "限时优惠进行中",
    image: "/banner2.jpg",
  },
];
</script>
```

## API

### Props

| 参数       | 说明                     | 类型      | 默认值  |
| ---------- | ------------------------ | --------- | ------- |
| items      | 轮播项目数组             | `any[]`   | `[]`    |
| showArrows | 是否显示左右导航箭头     | `boolean` | `true`  |
| showDots   | 是否显示底部指示点       | `boolean` | `true`  |
| autoplay   | 是否自动播放             | `boolean` | `false` |
| interval   | 自动播放间隔时间（毫秒） | `number`  | `3000`  |

### Slots

| 名称    | 说明             | 作用域参数                     |
| ------- | ---------------- | ------------------------------ |
| default | 自定义轮播项内容 | `{ item: any, index: number }` |

### 使用说明

#### 项目数据格式

`items` 数组可以包含任何类型的数据，通过插槽可以访问到：

- `item`: 当前轮播项的数据
- `index`: 当前项的索引（从0开始）

#### 样式定制

轮播图使用 Tailwind CSS 类名，可以通过覆盖以下类名来自定义样式：

```css
/* 自定义轮播容器样式 */
.carousel {
  @apply rounded-lg shadow-lg;
}

/* 自定义导航箭头样式 */
.carousel-arrow {
  @apply bg-blue-500/80 text-white;
}

/* 自定义指示点样式 */
.carousel-dot {
  @apply w-3 h-3 bg-blue-500/50;
}

.carousel-dot.active {
  @apply bg-blue-500 scale-125;
}
```

## 使用场景

- **产品展示**: 展示产品图片或特性介绍
- **图片画廊**: 照片浏览和幻灯片展示
- **内容推荐**: 推荐文章、新闻或活动信息
- **教程步骤**: 分步展示教程或引导流程
- **用户引导**: 新手引导或功能介绍

## 注意事项

- 确保轮播项目有合适的高度和内容
- 自动播放时建议提供手动控制选项
- 在移动设备上测试触摸滑动体验
- 考虑为图片添加适当的加载状态
