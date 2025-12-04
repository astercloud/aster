# SystemMessage 系统消息

用于显示系统通知、提示信息的消息组件。

## 基础用法

显示简单的系统通知消息。

```vue
<template>
  <div class="space-y-4">
    <SystemMessage> 欢迎来到聊天系统！ </SystemMessage>

    <SystemMessage> 用户 张三 加入了聊天室 </SystemMessage>

    <SystemMessage> 消息历史记录已加载完成 </SystemMessage>
  </div>
</template>

<script setup>
import { SystemMessage } from "@/components/ChatUI";
</script>
```

## 不同类型的消息

虽然组件本身不区分类型，但可以通过样式和图标来区分不同用途。

```vue
<template>
  <div class="space-y-4">
    <!-- 欢迎消息 -->
    <SystemMessage class="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
      <div class="flex items-center gap-2">
        <Icon type="info-circle" class="text-blue-500" />
        <span>欢迎使用 AI 助手！我是您的智能助手。</span>
      </div>
    </SystemMessage>

    <!-- 成功消息 -->
    <SystemMessage class="bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800">
      <div class="flex items-center gap-2">
        <Icon type="check-circle" class="text-green-500" />
        <span>文件上传成功</span>
      </div>
    </SystemMessage>

    <!-- 警告消息 -->
    <SystemMessage class="bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800">
      <div class="flex items-center gap-2">
        <Icon type="warning" class="text-yellow-500" />
        <span>检测到网络连接不稳定</span>
      </div>
    </SystemMessage>

    <!-- 错误消息 -->
    <SystemMessage class="bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800">
      <div class="flex items-center gap-2">
        <Icon type="error-circle" class="text-red-500" />
        <span>消息发送失败，请重试</span>
      </div>
    </SystemMessage>
  </div>
</template>

<script setup>
import { SystemMessage, Icon } from "@/components/ChatUI";
</script>
```

## 时间戳

在系统消息中添加时间戳信息。

```vue
<template>
  <div class="space-y-4">
    <SystemMessage>
      <div class="flex items-center justify-between">
        <span>用户 李四 加入了聊天</span>
        <span class="text-xs text-gray-500 ml-4">10:30 AM</span>
      </div>
    </SystemMessage>

    <SystemMessage>
      <div class="flex items-center justify-between">
        <span>会话已开始</span>
        <span class="text-xs text-gray-500 ml-4">10:31 AM</span>
      </div>
    </SystemMessage>
  </div>
</template>

<script setup>
import { SystemMessage } from "@/components/ChatUI";
</script>
```

## 交互式系统消息

包含操作按钮的系统消息。

```vue
<template>
  <div class="space-y-4">
    <SystemMessage>
      <div class="space-y-2">
        <p>检测到新版本可用，是否立即更新？</p>
        <div class="flex gap-2">
          <Button size="sm" variant="primary">立即更新</Button>
          <Button size="sm" variant="text">稍后提醒</Button>
        </div>
      </div>
    </SystemMessage>

    <SystemMessage>
      <div class="space-y-2">
        <p>您有未保存的更改，是否要保存？</p>
        <div class="flex gap-2">
          <Button size="sm" variant="primary">保存</Button>
          <Button size="sm" variant="secondary">不保存</Button>
          <Button size="sm" variant="text">取消</Button>
        </div>
      </div>
    </SystemMessage>
  </div>
</template>

<script setup>
import { SystemMessage, Button } from "@/components/ChatUI";
</script>
```

## 链接和操作

在系统消息中包含链接和可点击元素。

```vue
<template>
  <SystemMessage>
    <div class="space-y-2">
      <p>您的账户需要验证，请检查邮箱并点击验证链接。</p>
      <a href="#" class="text-blue-600 hover:underline text-sm">重新发送验证邮件</a>
    </div>
  </SystemMessage>

  <SystemMessage>
    <div class="space-y-2">
      <p>文件处理完成，点击下载结果：</p>
      <div class="flex items-center gap-2">
        <Icon type="download" class="text-blue-500" />
        <a href="#" class="text-blue-600 hover:underline text-sm">download-result.pdf</a>
      </div>
    </div>
  </SystemMessage>
</template>

<script setup>
import { SystemMessage, Icon } from "@/components/ChatUI";
</script>
```

## 在聊天流中使用

与其他消息组件混合使用，形成完整的对话流。

```vue
<template>
  <div class="space-y-4">
    <!-- 系统欢迎消息 -->
    <SystemMessage class="text-center"> 开始新的对话会话 </SystemMessage>

    <!-- 用户消息 -->
    <Bubble position="right"> 你好，我想了解产品功能 </Bubble>

    <!-- AI 回复 -->
    <Bubble position="left"> 您好！我很乐意为您介绍我们的产品功能... </Bubble>

    <!-- 系统提示 -->
    <SystemMessage>
      <div class="flex items-center gap-2">
        <Icon type="lightbulb" class="text-yellow-500" />
        <span>提示：您可以随时输入 "帮助" 获取更多信息</span>
      </div>
    </SystemMessage>

    <!-- 更多对话... -->
  </div>
</template>

<script setup>
import { SystemMessage, Bubble, Icon } from "@/components/ChatUI";
</script>
```

## API

SystemMessage 是一个简单的展示组件，主要通过插槽传递内容。

### Slots

| 名称    | 说明           |
| ------- | -------------- |
| default | 系统消息的内容 |

### 默认样式

组件内置了以下样式：

- 居中显示
- 浅色背景和边框
- 较小的字体尺寸
- 适当的内边距和圆角

### 使用说明

#### 内容设计

- 保持系统消息简洁明了
- 使用中性或友好的语言
- 提供必要的操作指引

#### 视觉区分

- 可以通过添加背景色来区分不同类型的消息
- 使用图标增强视觉传达效果
- 保持与整体设计风格一致

#### 交互设计

- 避免在系统消息中放置过多交互元素
- 为重要操作提供明确的按钮
- 考虑移动设备的触摸体验

## 使用场景

- **聊天欢迎**: 显示欢迎信息和使用指引
- **状态通知**: 系统状态变化的通知
- **操作确认**: 重要操作的确认提示
- **错误提示**: 错误信息的友好展示
- **帮助信息**: 使用帮助和操作指引
- **版本更新**: 新版本或功能更新的通知

## 最佳实践

- 保持消息简短且有意义
- 使用适当的图标增强表达
- 为重要的系统消息提供视觉突出
- 避免频繁发送系统消息
- 考虑消息的优先级和重要性
