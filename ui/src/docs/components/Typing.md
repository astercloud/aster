# Typing 打字效果

逐字显示文本的动画效果组件。

## 基础用法

简单的打字机效果显示文本。

```vue
<template>
  <div class="space-y-6">
    <Typing text="欢迎使用 ChatUI 组件库！" />

    <Typing text="这是一个打字机效果的示例" />

    <Typing text="支持中英文混合显示效果" />
  </div>
</template>

<script setup>
import { Typing } from "@/components/ChatUI";
</script>
```

## 自定义速度

控制打字速度和字符间隔。

```vue
<template>
  <div class="space-y-6">
    <div>
      <h3 class="font-semibold mb-2">快速打字 (50ms)</h3>
      <Typing text="这是快速打字效果" :speed="50" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">正常速度 (100ms)</h3>
      <Typing text="这是正常打字速度" :speed="100" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">慢速打字 (200ms)</h3>
      <Typing text="这是慢速打字效果" :speed="200" />
    </div>
  </div>
</template>

<script setup>
import { Typing } from "@/components/ChatUI";
</script>
```

## 循环播放

设置文本循环播放功能。

```vue
<template>
  <div class="space-y-6">
    <div>
      <h3 class="font-semibold mb-2">循环播放</h3>
      <Typing text="这条消息会循环播放..." :loop="true" :delay="1000" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">多条文本循环</h3>
      <Typing :texts="['第一条消息', '第二条消息', '第三条消息']" :loop="true" :speed="80" />
    </div>
  </div>
</template>

<script setup>
import { Typing } from "@/components/ChatUI";
</script>
```

## 延迟开始

设置打字效果的延迟开始时间。

```vue
<template>
  <div class="space-y-6">
    <Typing text="2秒后开始显示" :delay="2000" />

    <Typing text="这条消息会等待3秒" :delay="3000" :speed="60" />
  </div>
</template>

<script setup>
import { Typing } from "@/components/ChatUI";
</script>
```

## 在消息中使用

在聊天气泡中展示打字效果。

```vue
<template>
  <div class="space-y-4">
    <Bubble position="left">
      <Typing text="你好！我是AI助手，很高兴为您服务。" :speed="80" />
    </Bubble>

    <Bubble position="right"> 你好！我想了解一下产品功能 </Bubble>

    <Bubble position="left">
      <Typing text="我们的产品包含丰富的组件库..." :speed="100" />
    </Bubble>
  </div>
</template>

<script setup>
import { Typing, Bubble } from "@/components/ChatUI";
</script>
```

## 动态文本

支持动态变化的文本内容。

```vue
<template>
  <div class="space-y-4">
    <div class="flex gap-2">
      <Button @click="changeText">更换文本</Button>
      <Button @click="pauseText" variant="secondary">暂停</Button>
      <Button @click="resumeText" variant="secondary">继续</Button>
    </div>

    <Typing :key="currentText" :text="currentText" :speed="70" ref="typingRef" />
  </div>
</template>

<script setup>
import { ref } from "vue";
import { Typing, Button } from "@/components/ChatUI";

const typingRef = ref(null);
const texts = ["这是第一条消息", "这是第二条不同的消息", "这是第三条更新的消息"];
const currentIndex = ref(0);
const currentText = ref(texts[0]);

const changeText = () => {
  currentIndex.value = (currentIndex.value + 1) % texts.length;
  currentText.value = texts[currentIndex.value];
};

const pauseText = () => {
  typingRef.value?.pause();
};

const resumeText = () => {
  typingRef.value?.resume();
};
</script>
```

## 自定义样式

通过 CSS 自定义打字效果的样式。

```vue
<template>
  <div class="space-y-6">
    <Typing text="带有光标效果的文本" class="typing-with-cursor" />

    <Typing text="彩色打字效果" class="typing-colorful" />

    <Typing text="大字体打字效果" class="typing-large" />
  </div>
</template>

<script setup>
import { Typing } from "@/components/ChatUI";
</script>

<style scoped>
.typing-with-cursor :deep(.typing-cursor) {
  @apply w-0.5 h-6 bg-blue-500 ml-0.5;
}

.typing-colorful :deep(.typing-text) {
  @apply bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent font-semibold;
}

.typing-large :deep(.typing-text) {
  @apply text-2xl font-bold;
}
</style>
```

## API

### Props

| 参数  | 说明                           | 类型       | 默认值  |
| ----- | ------------------------------ | ---------- | ------- |
| text  | 要显示的文本内容               | `string`   | `''`    |
| texts | 多条文本数组（与 text 二选一） | `string[]` | `[]`    |
| speed | 打字速度（毫秒/字符）          | `number`   | `100`   |
| delay | 开始延迟时间（毫秒）           | `number`   | `0`     |
| loop  | 是否循环播放                   | `boolean`  | `false` |

### Methods

通过 ref 可以访问以下方法：

| 方法    | 说明         | 参数 |
| ------- | ------------ | ---- |
| pause   | 暂停打字效果 | -    |
| resume  | 继续打字效果 | -    |
| restart | 重新开始     | -    |

### 使用说明

#### 性能优化

- 使用 requestAnimationFrame 优化动画性能
- 避免频繁的 DOM 操作
- 支持大量文本的高效渲染

#### 字符处理

- 支持中英文字符的正确显示
- 处理空格和特殊字符
- 保持文本的原始格式

#### 状态管理

- 组件内部维护打字状态
- 支持暂停和恢复功能
- 循环播放时正确重置状态

## 使用场景

- **AI 对话**: 模拟 AI 正在输入的效果
- **代码演示**: 逐行展示代码内容
- **引导教程**: 分步骤显示指导内容
- **品牌介绍**: 逐字展示品牌标语
- **加载状态**: 显示加载过程中的提示文本

## 最佳实践

- 保持文本长度适中，避免过长的打字效果
- 选择合适的打字速度，保证可读性
- 在移动设备上测试打字效果的流畅性
- 考虑为用户提供跳过打字效果的选项
- 避免在同一个页面使用过多同时进行的打字效果
