# MultimodalInput 多模态输入

支持文本、图片、语音等多种输入方式的聊天输入组件。

## 基础用法

基本的文本输入和发送功能。

```vue
<template>
  <MultimodalInput v-model="message" placeholder="输入消息..." @send="handleSend" />
</template>

<script setup>
import { ref } from "vue";
import { MultimodalInput } from "@/components/ChatUI";

const message = ref("");

const handleSend = (data) => {
  console.log("发送:", data);
  // data: { text: '消息内容', image?: { data: 'base64', preview: 'dataURL' } }
};
</script>
```

## 图片输入

启用图片上传功能，支持图片预览和删除。

```vue
<template>
  <MultimodalInput v-model="text" :enable-image="true" placeholder="输入消息或上传图片..." @send="handleSendWithImage" />
</template>

<script setup>
import { ref } from "vue";
import { MultimodalInput } from "@/components/ChatUI";

const text = ref("");

const handleSendWithImage = (data) => {
  if (data.image) {
    console.log("图片数据:", data.image.data);
    console.log("预览URL:", data.image.preview);
  }
  console.log("文本内容:", data.text);
  // 处理包含图片的消息发送
};
</script>
```

## 语音输入

启用语音识别功能，支持中文语音输入。

```vue
<template>
  <MultimodalInput v-model="voiceText" :enable-voice="true" placeholder="输入文本或点击麦克风说话..." @send="handleVoiceSend" />
</template>

<script setup>
import { ref } from "vue";
import { MultimodalInput } from "@/components/ChatUI";

const voiceText = ref("");

const handleVoiceSend = (data) => {
  console.log("语音转文本结果:", data.text);
  // 处理语音输入的结果
};
</script>
```

## 禁用文件上传

只保留文本和图片输入，禁用文件上传功能。

```vue
<template>
  <MultimodalInput v-model="message" :enable-file="false" placeholder="输入消息..." @send="handleSend" />
</template>

<script setup>
import { ref } from "vue";
import { MultimodalInput } from "@/components/ChatUI";

const message = ref("");

const handleSend = (data) => {
  console.log("发送消息:", data);
};
</script>
```

## 完整功能

启用所有功能：文本、图片、语音和文件上传。

```vue
<template>
  <div class="space-y-4">
    <div class="bg-gray-100 dark:bg-gray-800 p-4 rounded-lg">
      <h3 class="font-semibold mb-2">多模态聊天输入</h3>
      <MultimodalInput v-model="fullMessage" :enable-image="true" :enable-voice="true" :enable-file="true" placeholder="支持文本、图片、语音和文件..." @send="handleFullSend" />
    </div>

    <div class="text-sm text-gray-600 dark:text-gray-400">
      <p>• 点击图片图标上传图片</p>
      <p>• 点击麦克风图标开始语音输入（需Chrome浏览器）</p>
      <p>• 点击附件图标上传文件</p>
      <p>• Enter 发送，Shift+Enter 换行</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { MultimodalInput } from "@/components/ChatUI";

const fullMessage = ref("");

const handleFullSend = (data) => {
  console.log("完整数据:", data);

  // 处理不同类型的输入
  if (data.text) {
    console.log("文本内容:", data.text);
  }

  if (data.image) {
    console.log("包含图片");
    // 上传图片到服务器
  }

  // 发送到后端API
  // sendToAPI(data);
};
</script>
```

## API

### Props

| 参数        | 说明             | 类型      | 默认值          |
| ----------- | ---------------- | --------- | --------------- |
| modelValue  | 输入框绑定值     | `string`  | `''`            |
| placeholder | 输入框占位符     | `string`  | `'输入消息...'` |
| disabled    | 是否禁用输入     | `boolean` | `false`         |
| enableImage | 是否启用图片上传 | `boolean` | `true`          |
| enableVoice | 是否启用语音输入 | `boolean` | `true`          |
| enableFile  | 是否启用文件上传 | `boolean` | `false`         |

### Events

| 事件名            | 说明               | 回调参数            |
| ----------------- | ------------------ | ------------------- |
| update:modelValue | 输入内容变化时触发 | `value: string`     |
| send              | 发送消息时触发     | `data: MessageData` |

### MessageData 类型

```typescript
interface MessageData {
  text: string; // 文本内容
  image?: {
    // 可选的图片数据
    data: string; // Base64 图片数据
    preview: string; // 预览 URL (data URL)
  };
}
```

## 功能特性

### 自动调整高度

输入框会根据内容自动调整高度，最多 120px。

### 快捷键支持

- `Enter`: 发送消息
- `Shift+Enter`: 换行

### 图片预览

- 支持常见图片格式
- 提供缩略图预览
- 可删除已选择的图片

### 语音识别

- 基于 Web Speech API
- 支持中文语音识别
- 实时转换语音为文本

### 浏览器兼容性

- 语音输入需要 Chrome、Edge 等支持 Web Speech API 的浏览器
- 图片上传支持所有现代浏览器
- 文本输入完全兼容所有浏览器

## 使用场景

- **即时通讯**: 支持富媒体的聊天应用
- **AI 对话**: 支持多模态输入的AI助手
- **客服系统**: 支持图片和文本的客户支持
- **社交应用**: 具备丰富输入方式的社交平台
- **内容创作**: 支持多媒体内容发布的编辑器

## 注意事项

- 语音输入需要在 HTTPS 环境下运行
- 图片数据会转换为 Base64 格式，注意内存使用
- 建议为文件上传实现完整的后端处理逻辑
- 语音识别准确度受网络环境和麦克风质量影响
