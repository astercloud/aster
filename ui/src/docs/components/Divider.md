# Divider 分割线

用于分隔内容的分割线组件。

## 基础用法

基本的分割线。

```vue
<template>
  <div>
    <p>内容上方</p>
    <Divider />
    <p>内容下方</p>
  </div>
</template>
```

## 带文字

分割线可以包含文字。

```vue
<template>
  <div>
    <p>内容上方</p>
    <Divider>分割线文字</Divider>
    <p>内容下方</p>
  </div>
</template>
```

## 垂直分割线

垂直方向的分割线。

```vue
<template>
  <Flex align="center">
    <span>选项 1</span>
    <Divider direction="vertical" />
    <span>选项 2</span>
    <Divider direction="vertical" />
    <span>选项 3</span>
  </Flex>
</template>
```

## API

### Props

| 参数      | 说明       | 类型                         | 默认值         |
| --------- | ---------- | ---------------------------- | -------------- |
| direction | 分割线方向 | `'horizontal' \| 'vertical'` | `'horizontal'` |

### Slots

| 名称    | 说明           |
| ------- | -------------- |
| default | 分割线中的文字 |

## 示例

### 章节分隔

```vue
<template>
  <div>
    <h2>第一章</h2>
    <p>第一章的内容...</p>

    <Divider>第二章</Divider>

    <h2>第二章</h2>
    <p>第二章的内容...</p>
  </div>
</template>
```

### 工具栏分隔

```vue
<template>
  <Flex align="center" gap="md">
    <Button icon="save">保存</Button>
    <Divider direction="vertical" />
    <Button icon="copy">复制</Button>
    <Divider direction="vertical" />
    <Button icon="delete">删除</Button>
  </Flex>
</template>
```
