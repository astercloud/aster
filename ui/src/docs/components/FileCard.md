# FileCard 文件卡片

用于展示文件信息的卡片组件，支持文件名、大小显示和下载链接。

## 基础用法

基本的文件卡片用法。

```vue
<template>
  <FileCard :file="fileInfo" />
</template>

<script setup>
import { FileCard } from "@/components/ChatUI";

const fileInfo = {
  name: "document.pdf",
  size: 1024000,
  url: "/files/document.pdf",
};
</script>
```

## 不带下载链接

当没有提供下载链接时，下载按钮不会显示。

```vue
<template>
  <FileCard :file="readOnlyFile" />
</template>

<script setup>
import { FileCard } from "@/components/ChatUI";

const readOnlyFile = {
  name: "readonly.txt",
  size: 2048,
};
</script>
```

## 多个文件

在聊天场景中展示多个文件。

```vue
<template>
  <Flex direction="column" gap="md">
    <FileCard :file="file1" />
    <FileCard :file="file2" />
    <FileCard :file="file3" />
  </Flex>
</template>

<script setup>
import { FileCard, Flex } from "@/components/ChatUI";

const file1 = {
  name: "presentation.pptx",
  size: 5242880,
  url: "/files/presentation.pptx",
};

const file2 = {
  name: "spreadsheet.xlsx",
  size: 1048576,
  url: "/files/spreadsheet.xlsx",
};

const file3 = {
  name: "image.jpg",
  size: 512000,
  url: "/files/image.jpg",
};
</script>
```

## 在消息中使用

在聊天消息中展示文件卡片。

```vue
<template>
  <Bubble position="left">
    <Flex direction="column" gap="sm">
      <div>我已经上传了以下文件：</div>
      <FileCard :file="attachedFile" />
    </Flex>
  </Bubble>
</template>

<script setup>
import { FileCard, Bubble, Flex } from "@/components/ChatUI";

const attachedFile = {
  name: "project-plan.pdf",
  size: 2097152,
  url: "/files/project-plan.pdf",
};
</script>
```

## API

### Props

| 参数 | 说明         | 类型       | 必填 |
| ---- | ------------ | ---------- | ---- |
| file | 文件信息对象 | `FileInfo` | ✓    |

### FileInfo 类型

| 属性 | 说明             | 类型     | 必填 |
| ---- | ---------------- | -------- | ---- |
| name | 文件名           | `string` | ✓    |
| size | 文件大小（字节） | `number` | ✓    |
| url  | 下载链接         | `string` | -    |

### 特性说明

- **自动格式化文件大小**：组件会自动将字节转换为合适的单位（B、KB、MB、GB）
- **文件名截断**：当文件名过长时会自动截断，确保布局美观
- **响应式设计**：支持深色模式和不同屏幕尺寸
- **无障碍支持**：包含适当的语义化标签和可访问性属性

### 样式定制

FileCard 使用 Tailwind CSS 类名，你可以通过覆盖以下类名来定制样式：

```css
/* 自定义文件卡片背景 */
.file-card {
  @apply bg-white border-gray-300;
}

/* 自定义文件图标颜色 */
.file-icon {
  @apply text-green-500;
}

/* 自定义文件名字体 */
.file-name {
  @apply font-bold;
}
```

## 使用场景

- **聊天应用**：展示用户上传的文件
- **文档管理**：显示文件列表和下载链接
- **邮件客户端**：展示邮件附件
- **云存储**：显示云端文件信息
