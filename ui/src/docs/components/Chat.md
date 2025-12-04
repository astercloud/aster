# Chat 聊天容器

完整的聊天界面组件，包含消息列表和输入区域。

## 基础用法

基本的聊天界面。

```vue
<template>
  <Chat :messages="messages" placeholder="输入消息..." @send="handleSend" />
</template>

<script setup>
import { ref } from "vue";

const messages = ref([
  {
    id: "1",
    type: "text",
    content: "你好！",
    position: "left",
  },
  {
    id: "2",
    type: "text",
    content: "很高兴认识你",
    position: "right",
    status: "sent",
  },
]);

const handleSend = (message) => {
  messages.value.push({
    id: Date.now().toString(),
    type: "text",
    content: message.content,
    position: "right",
  });
};
</script>
```

## 快捷回复

添加快捷回复按钮。

```vue
<template>
  <Chat :messages="messages" :quick-replies="quickReplies" @send="handleSend" @quick-reply="handleQuickReply" />
</template>

<script setup>
const quickReplies = [
  { name: "帮我写文章", value: "write" },
  { name: "分析代码", value: "analyze" },
  { name: "生成工作流", value: "workflow" },
];

const handleQuickReply = (reply) => {
  console.log("Quick reply:", reply);
};
</script>
```

## 工具栏

添加工具栏按钮。

```vue
<template>
  <Chat :messages="messages" :toolbar="toolbar" @send="handleSend" />
</template>

<script setup>
const toolbar = [
  { icon: "image", onClick: () => console.log("上传图片") },
  { icon: "attach", onClick: () => console.log("上传文件") },
  { icon: "mic", onClick: () => console.log("语音输入") },
];
</script>
```

## 消息类型

支持多种消息类型。

```vue
<template>
  <Chat :messages="messages" />
</template>

<script setup>
const messages = [
  // 文本消息
  {
    id: "1",
    type: "text",
    content: "这是文本消息",
    position: "left",
  },
  // 思考消息
  {
    id: "2",
    type: "thinking",
    content: "正在思考...",
    position: "left",
  },
  // 打字中
  {
    id: "3",
    type: "typing",
    position: "left",
  },
  // 卡片消息
  {
    id: "4",
    type: "card",
    position: "left",
    card: {
      title: "推荐内容",
      content: "这是内容...",
      actions: [
        { text: "查看", value: "view" },
        { text: "分享", value: "share" },
      ],
    },
  },
  // 文件消息
  {
    id: "5",
    type: "file",
    position: "left",
    file: {
      name: "document.pdf",
      size: 1024000,
      url: "https://example.com/file.pdf",
    },
  },
];
</script>
```

## API

### Props

| 参数         | 说明         | 类型            | 默认值          |
| ------------ | ------------ | --------------- | --------------- |
| messages     | 消息列表     | `Message[]`     | `[]`            |
| placeholder  | 输入框占位符 | `string`        | `'输入消息...'` |
| disabled     | 是否禁用输入 | `boolean`       | `false`         |
| quickReplies | 快捷回复列表 | `QuickReply[]`  | `[]`            |
| toolbar      | 工具栏按钮   | `ToolbarItem[]` | `[]`            |

### Events

| 事件名     | 说明               | 回调参数                            |
| ---------- | ------------------ | ----------------------------------- |
| send       | 发送消息时触发     | `{ type: string, content: string }` |
| quickReply | 点击快捷回复时触发 | `QuickReply`                        |
| cardAction | 点击卡片操作时触发 | `{ value: string }`                 |

### Message 类型

```typescript
interface Message {
  id: string;
  type: "text" | "thinking" | "typing" | "card" | "file";
  content?: string;
  position: "left" | "right";
  status?: "pending" | "sent" | "error";
  user?: {
    avatar?: string;
    name?: string;
  };
  card?: {
    title: string;
    content: string;
    actions?: Array<{ text: string; value: string }>;
  };
  file?: {
    name: string;
    size: number;
    url: string;
  };
}
```
