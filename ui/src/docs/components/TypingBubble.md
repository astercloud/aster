# TypingBubble 输入中气泡

显示对方正在输入状态的动态气泡组件。

## 基础用法

简单的输入状态显示。

```vue
<template>
  <div class="space-y-4">
    <TypingBubble />

    <div class="flex items-center gap-2 text-sm text-gray-500">
      <TypingBubble />
      <span>对方正在输入...</span>
    </div>
  </div>
</template>

<script setup>
import { TypingBubble } from "@/components/ChatUI";
</script>
```

## 在对话流中使用

与其他消息组件一起使用，形成真实的聊天场景。

```vue
<template>
  <div class="space-y-4">
    <!-- 用户消息 -->
    <Bubble position="right"> 你好，能帮我分析一下这个数据吗？ </Bubble>

    <!-- 输入状态 -->
    <Bubble position="left">
      <TypingBubble />
    </Bubble>

    <!-- AI 回复 -->
    <Bubble position="left"> 当然可以！请提供您想要分析的数据... </Bubble>

    <!-- 再次输入状态 -->
    <Bubble position="left">
      <TypingBubble />
    </Bubble>
  </div>
</template>

<script setup>
import { TypingBubble, Bubble } from "@/components/ChatUI";
</script>
```

## 自定义样式

通过 props 自定义气泡的样式和行为。

```vue
<template>
  <div class="space-y-6">
    <div>
      <h3 class="font-semibold mb-2">默认样式</h3>
      <TypingBubble />
    </div>

    <div>
      <h3 class="font-semibold mb-2">快速动画</h3>
      <TypingBubble :speed="0.8" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">慢速动画</h3>
      <TypingBubble :speed="0.3" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">更多点数</h3>
      <TypingBubble :dots="4" />
    </div>

    <div>
      <h3 class="font-semibold mb-2">大尺寸</h3>
      <TypingBubble size="lg" />
    </div>
  </div>
</template>

<script setup>
import { TypingBubble } from "@/components/ChatUI";
</script>
```

## 带状态文本

配合文本显示更详细的输入状态信息。

```vue
<template>
  <div class="space-y-4">
    <div class="flex items-center gap-3">
      <Avatar alt="Bot" size="sm" />
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <TypingBubble />
          <span class="text-sm text-gray-500">AI 助手正在思考...</span>
        </div>
      </div>
    </div>

    <div class="flex items-center gap-3">
      <Avatar alt="User" size="sm" />
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <TypingBubble position="right" />
          <span class="text-sm text-gray-500">用户正在输入...</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { TypingBubble, Avatar } from "@/components/ChatUI";
</script>
```

## 动态控制

通过 ref 控制气泡的显示和隐藏。

```vue
<template>
  <div class="space-y-4">
    <div class="flex gap-2">
      <Button @click="startTyping">开始输入</Button>
      <Button @click="stopTyping" variant="secondary">停止输入</Button>
      <Button @click="toggleTyping" variant="text">切换</Button>
    </div>

    <div class="space-y-4">
      <Bubble position="left">
        <TypingBubble v-if="isTyping" ref="typingRef" />
      </Bubble>

      <Bubble position="right" v-if="!isTyping"> 对方已停止输入 </Bubble>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { TypingBubble, Bubble, Button } from "@/components/ChatUI";

const isTyping = ref(false);
const typingRef = ref(null);

const startTyping = () => {
  isTyping.value = true;
  // 5秒后自动停止
  setTimeout(() => {
    isTyping.value = false;
  }, 5000);
};

const stopTyping = () => {
  isTyping.value = false;
};

const toggleTyping = () => {
  isTyping.value = !isTyping.value;
  if (isTyping.value) {
    setTimeout(() => {
      isTyping.value = false;
    }, 3000);
  }
};
</script>
```

## 实际应用场景

模拟真实的聊天应用中的输入状态。

```vue
<template>
  <div class="max-w-2xl mx-auto">
    <h2 class="text-xl font-bold mb-4">客服对话模拟</h2>

    <div class="space-y-4">
      <!-- 用户咨询 -->
      <Bubble position="right"> 你好，我想咨询一下产品的使用问题 </Bubble>

      <!-- 客服输入中 -->
      <Bubble position="left" v-if="step === 1">
        <TypingBubble />
      </Bubble>

      <!-- 客服回复 -->
      <Bubble position="left" v-if="step === 2"> 您好！我是客服小助手，很高兴为您服务。请问您遇到了什么问题？ </Bubble>

      <!-- 用户详细描述 -->
      <Bubble position="right" v-if="step === 3"> 我在使用文件上传功能时遇到了问题，文件大小限制是多少？ </Bubble>

      <!-- 客服思考中 -->
      <Bubble position="left" v-if="step === 4">
        <TypingBubble :speed="0.6" :dots="3" />
      </Bubble>

      <!-- 客服详细回答 -->
      <Bubble position="left" v-if="step === 5">
        关于文件上传，我们的系统支持以下限制：<br />
        • 单个文件最大 10MB<br />
        • 支持 jpg、png、pdf、doc 等常见格式<br />
        • 如需上传更大文件，请联系客服
      </Bubble>
    </div>

    <!-- 控制按钮 -->
    <div class="flex gap-2 mt-4">
      <Button @click="nextStep" :disabled="step >= 5">下一步</Button>
      <Button @click="reset" variant="secondary">重置</Button>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { TypingBubble, Bubble, Button } from "@/components/ChatUI";

const step = ref(1);

const nextStep = () => {
  if (step.value < 5) {
    step.value++;

    // 自动处理输入状态
    if (step.value === 1 || step.value === 4) {
      setTimeout(() => {
        step.value++;
      }, 2000);
    }
  }
};

const reset = () => {
  step.value = 1;
};
</script>
```

## API

### Props

| 参数     | 说明           | 类型                   | 默认值   |
| -------- | -------------- | ---------------------- | -------- |
| position | 气泡位置       | `'left' \| 'right'`    | `'left'` |
| dots     | 动画点数量     | `number`               | `3`      |
| speed    | 动画速度（秒） | `number`               | `1`      |
| size     | 气泡大小       | `'sm' \| 'md' \| 'lg'` | `'md'`   |

### 使用说明

#### 动画效果

- 使用 CSS 动画实现点的跳动效果
- 支持自定义动画速度和点数
- 动画循环播放，模拟真实的输入状态

#### 视觉样式

- 默认使用浅色背景和深色点
- 支持不同尺寸的气泡
- 与其他聊天组件保持一致的视觉风格

#### 位置定位

- 支持 left 和 right 两个位置
- 可以与其他消息组件配合使用
- 自动适应不同的聊天场景

## 使用场景

- **即时通讯**: 显示对方正在输入的状态
- **AI 对话**: 模拟 AI 正在思考和输入
- **客服系统**: 显示客服正在回复的状态
- **实时协作**: 显示其他协作者的输入状态
- **游戏聊天**: 在多人游戏中显示输入状态

## 最佳实践

- 与实际的消息发送逻辑配合使用
- 设置合理的动画速度，避免过快或过慢
- 在实际输入开始时显示，输入完成时隐藏
- 考虑网络延迟，适当延长显示时间
- 避免同时显示过多的输入状态气泡
