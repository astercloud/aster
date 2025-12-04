# ScrollView 滚动视图

优化的滚动容器，支持自定义滚动条样式和性能优化。

## 基础用法

创建可滚动的内容区域。

```vue
<template>
  <ScrollView class="h-64">
    <div class="p-4">
      <h3 class="font-semibold mb-2">可滚动内容</h3>
      <div class="space-y-4">
        <div v-for="i in 20" :key="i" class="p-4 bg-gray-100 dark:bg-gray-800 rounded">内容项 {{ i }}</div>
      </div>
    </div>
  </ScrollView>
</template>

<script setup>
import { ScrollView } from "@/components/ChatUI";
</script>
```

## 自动高度

根据内容自动调整高度，设置最大高度限制。

```vue
<template>
  <ScrollView class="max-h-96">
    <div class="p-6">
      <h2 class="text-xl font-bold mb-4">聊天记录</h2>
      <div class="space-y-3">
        <div v-for="i in 50" :key="i" class="flex items-start gap-3">
          <Avatar :alt="`User${i}`" size="sm" />
          <div class="flex-1">
            <p class="font-medium">用户{{ i }}</p>
            <p class="text-gray-600 dark:text-gray-400 text-sm">这是第 {{ i }} 条消息内容</p>
          </div>
        </div>
      </div>
    </div>
  </ScrollView>
</template>

<script setup>
import { ScrollView, Avatar } from "@/components/ChatUI";
</script>
```

## 水平滚动

支持水平方向的滚动。

```vue
<template>
  <ScrollView class="h-32 w-full">
    <div class="flex gap-4 p-4">
      <div v-for="i in 10" :key="i" class="flex-shrink-0 w-48 h-24 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center text-white font-bold">卡片 {{ i }}</div>
    </div>
  </ScrollView>
</template>

<script setup>
import { ScrollView } from "@/components/ChatUI";
</script>
```

## 滚动到指定位置

提供方法控制滚动位置。

```vue
<template>
  <div class="space-y-4">
    <div class="flex gap-2">
      <Button @click="scrollToTop">滚动到顶部</Button>
      <Button @click="scrollToBottom">滚动到底部</Button>
      <Button @click="scrollToIndex(10)">滚动到第10项</Button>
    </div>

    <ScrollView ref="scrollViewRef" class="h-64">
      <div class="p-4">
        <div v-for="i in 30" :key="i" :ref="`item-${i}`" class="p-4 mb-2 bg-gray-100 dark:bg-gray-800 rounded">
          <h4 class="font-semibold">项目 {{ i }}</h4>
          <p class="text-gray-600 dark:text-gray-400">这是第 {{ i }} 个项目的内容</p>
        </div>
      </div>
    </ScrollView>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { ScrollView, Button } from "@/components/ChatUI";

const scrollViewRef = ref(null);

const scrollToTop = () => {
  scrollViewRef.value?.scrollTo({ top: 0, behavior: "smooth" });
};

const scrollToBottom = () => {
  scrollViewRef.value?.scrollTo({ top: scrollViewRef.value.scrollHeight, behavior: "smooth" });
};

const scrollToIndex = (index) => {
  const element = scrollViewRef.value?.querySelector(`[ref="item-${index}"]`);
  if (element) {
    element.scrollIntoView({ behavior: "smooth", block: "center" });
  }
};
</script>
```

## 滚动事件监听

监听滚动事件并获取滚动位置。

```vue
<template>
  <div class="space-y-4">
    <div class="bg-gray-100 dark:bg-gray-800 p-4 rounded">
      <div class="grid grid-cols-3 gap-4 text-sm">
        <div>
          <span class="font-medium">滚动位置:</span>
          <span>{{ Math.round(scrollPosition.top) }}px</span>
        </div>
        <div>
          <span class="font-medium">滚动比例:</span>
          <span>{{ Math.round(scrollProgress) }}%</span>
        </div>
        <div>
          <span class="font-medium">方向:</span>
          <span>{{ scrollDirection }}</span>
        </div>
      </div>
    </div>

    <ScrollView class="h-64" @scroll="handleScroll">
      <div class="p-4">
        <div v-for="i in 100" :key="i" class="p-4 mb-2 bg-gray-100 dark:bg-gray-800 rounded">
          <p class="font-medium">项目 {{ i }}</p>
          <p class="text-sm text-gray-600 dark:text-gray-400">滚动查看更多内容...</p>
        </div>
      </div>
    </ScrollView>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { ScrollView } from "@/components/ChatUI";

const scrollPosition = ref({ top: 0, left: 0 });
const scrollProgress = ref(0);
const scrollDirection = ref("");

const handleScroll = (event) => {
  const element = event.target;
  scrollPosition.value = {
    top: element.scrollTop,
    left: element.scrollLeft,
  };

  scrollProgress.value = (element.scrollTop / (element.scrollHeight - element.clientHeight)) * 100;

  // 简单的方向检测
  const currentScrollTop = element.scrollTop;
  if (currentScrollTop > scrollPosition.value.top) {
    scrollDirection.value = "向下";
  } else if (currentScrollTop < scrollPosition.value.top) {
    scrollDirection.value = "向上";
  }
};
</script>
```

## 自定义滚动条

通过 CSS 自定义滚动条样式。

```vue
<template>
  <ScrollView class="h-64 custom-scrollbar">
    <div class="p-4">
      <h3 class="font-semibold mb-2">自定义滚动条</h3>
      <div class="space-y-4">
        <div v-for="i in 25" :key="i" class="p-4 bg-gray-100 dark:bg-gray-800 rounded">内容项 {{ i }} - 自定义样式的滚动条</div>
      </div>
    </div>
  </ScrollView>
</template>

<script setup>
import { ScrollView } from "@/components/ChatUI";
</script>

<style scoped>
.custom-scrollbar {
  /* Webkit 浏览器 */
  scrollbar-width: thin;
  scrollbar-color: #3b82f6 transparent;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #3b82f6;
  border-radius: 4px;
  border: 2px solid transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background-color: #2563eb;
}
</style>
```

## API

### Props

ScrollView 是一个无状态的容器组件，没有特殊的 props。

### Events

| 事件名 | 说明       | 回调参数       |
| ------ | ---------- | -------------- |
| scroll | 滚动时触发 | `event: Event` |

### 方法

通过 ref 可以访问原生的滚动方法：

| 方法           | 说明           | 参数                             |
| -------------- | -------------- | -------------------------------- |
| scrollTo       | 滚动到指定位置 | `options: ScrollToOptions`       |
| scrollBy       | 滚动指定距离   | `options: ScrollToOptions`       |
| scrollIntoView | 滚动到指定元素 | `options: ScrollIntoViewOptions` |

### 使用说明

#### 性能优化

- 使用 CSS transform 而非 top/left 属性
- 启用硬件加速和合成层
- 避免在滚动事件中进行重计算

#### 滚动条定制

支持完全自定义滚动条样式：

- 使用 CSS scrollbar-\* 属性
- 使用 ::-webkit-scrollbar 伪元素
- 可以隐藏原生滚动条并实现自定义滚动

#### 响应式设计

- 自动适应容器尺寸
- 支持触摸设备的滚动惯性
- 在移动设备上优化滚动体验

## 使用场景

- **聊天列表**: 消息历史记录的滚动显示
- **数据表格**: 大量数据的表格滚动
- **图片画廊**: 图片集的水平滚动
- **文档阅读**: 长文档的垂直滚动
- **选项列表**: 长选项菜单的滚动

## 最佳实践

- 为滚动容器设置明确的尺寸限制
- 在长列表中考虑使用虚拟滚动优化性能
- 提供滚动到顶部/底部的快速操作
- 为滚动位置提供持久化和恢复功能
