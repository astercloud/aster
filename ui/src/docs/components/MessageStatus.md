# MessageStatus 消息状态

用于显示消息发送状态的图标组件，包含发送中、已发送、已读和错误四种状态。

## 基础用法

显示不同状态的消息状态图标。

```vue
<template>
  <Flex direction="column" gap="md">
    <div class="flex items-center gap-2">
      <span>发送中:</span>
      <MessageStatus status="pending" />
    </div>
    <div class="flex items-center gap-2">
      <span>已发送:</span>
      <MessageStatus status="sent" />
    </div>
    <div class="flex items-center gap-2">
      <span>已读:</span>
      <MessageStatus status="read" />
    </div>
    <div class="flex items-center gap-2">
      <span>发送失败:</span>
      <MessageStatus status="error" />
    </div>
  </Flex>
</template>

<script setup>
import { MessageStatus, Flex } from "@/components/ChatUI";
</script>
```

## 在消息中使用

在聊天消息气泡中显示状态信息。

```vue
<template>
  <div class="space-y-4">
    <!-- 发送中的消息 -->
    <Bubble position="right" status="pending"> 这条消息正在发送中... </Bubble>

    <!-- 已发送的消息 -->
    <Bubble position="right" status="sent"> 消息已成功发送 </Bubble>

    <!-- 已读的消息 -->
    <Bubble position="right" status="read"> 对方已阅读此消息 </Bubble>

    <!-- 发送失败的消息 -->
    <Bubble position="right" status="error"> 消息发送失败，请重试 </Bubble>
  </div>
</template>

<script setup>
import { Bubble } from "@/components/ChatUI";
</script>
```

## 与时间戳结合

通常与发送时间一起显示在消息底部。

```vue
<template>
  <Bubble position="right" status="read">
    这是一条已读的消息
    <template #footer>
      <div class="flex items-center gap-2 text-xs text-gray-500 mt-2">
        <span>10:30 AM</span>
        <MessageStatus status="read" />
      </div>
    </template>
  </Bubble>
</template>

<script setup>
import { Bubble, MessageStatus } from "@/components/ChatUI";
</script>
```

## 状态管理示例

结合实际的消息状态管理使用。

```vue
<template>
  <div class="space-y-4">
    <div v-for="message in messages" :key="message.id">
      <Bubble :position="message.sender === 'user' ? 'right' : 'left'" :status="message.status">
        {{ message.content }}
        <template v-if="message.sender === 'user'" #footer>
          <div class="flex items-center justify-between mt-2">
            <span class="text-xs text-gray-500">{{ formatTime(message.timestamp) }}</span>
            <MessageStatus :status="message.status" />
          </div>
        </template>
      </Bubble>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { Bubble, MessageStatus } from "@/components/ChatUI";

const messages = ref([
  {
    id: 1,
    content: "你好！",
    sender: "user",
    status: "read",
    timestamp: new Date(Date.now() - 60000),
  },
  {
    id: 2,
    content: "你好！很高兴认识你",
    sender: "bot",
    status: "read",
    timestamp: new Date(Date.now() - 50000),
  },
  {
    id: 3,
    content: "这是一条正在发送的消息...",
    sender: "user",
    status: "pending",
    timestamp: new Date(),
  },
]);

const formatTime = (date) => {
  return date.toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
  });
};
</script>
```

## API

### Props

| 参数   | 说明     | 类型                                       | 必填 |
| ------ | -------- | ------------------------------------------ | ---- |
| status | 消息状态 | `'pending' \| 'sent' \| 'read' \| 'error'` | ✓    |

### Status 类型说明

| 状态值    | 说明     | 图标颜色 | 语义                   |
| --------- | -------- | -------- | ---------------------- |
| `pending` | 发送中   | 灰色     | 消息正在处理中         |
| `sent`    | 已发送   | 蓝色     | 消息已成功发送到服务器 |
| `read`    | 已读     | 蓝色     | 接收方已查看消息       |
| `error`   | 发送失败 | 红色     | 消息发送失败，需要重试 |

### 图标说明

每个状态都有对应的 SVG 图标：

- **pending**: 时钟图标，表示处理中
- **sent**: 单个勾号，表示发送成功
- **read**: 双勾信封图标，表示已读
- **error**: 错误圆圈图标，表示失败

## 使用场景

- **即时通讯**: 显示消息的投递和阅读状态
- **邮件客户端**: 显示邮件的发送状态
- **通知系统**: 显示通知的发送结果
- **文件上传**: 显示文件传输的进度状态
- **表单提交**: 显示表单数据的提交状态

## 最佳实践

### 状态转换

建议按照以下顺序进行状态转换：

1. `pending` → `sent` → `read`
2. `pending` → `error` → `pending` → `sent` → `read`

### 用户体验

- 对于发送失败的消息，建议提供重试功能
- 发送中的状态不应持续时间过长，避免用户焦虑
- 已读状态只在确认对方确实已查看时显示

### 样式定制

组件使用 Tailwind CSS 类名，可以通过 CSS 覆盖来自定义样式：

```css
/* 自定义状态图标大小 */
.message-status svg {
  @apply w-5 h-5;
}

/* 自定义错误状态颜色 */
.message-status:has(.status-icon[error]) {
  @apply text-orange-500;
}
```

## 可访问性

- 所有图标都使用 SVG，支持屏幕阅读器
- 图标颜色对比度符合 WCAG 标准
- 建议为状态变化提供适当的 ARIA 标签
