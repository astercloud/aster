# Flex 弹性布局

用于快速创建弹性布局的容器组件。

## 基础用法

基本的弹性布局。

```vue
<template>
  <Flex gap="md">
    <div>项目 1</div>
    <div>项目 2</div>
    <div>项目 3</div>
  </Flex>
</template>
```

## 方向

设置主轴方向。

```vue
<template>
  <div>
    <h3>水平方向（默认）</h3>
    <Flex direction="row" gap="md">
      <div>项目 1</div>
      <div>项目 2</div>
      <div>项目 3</div>
    </Flex>

    <h3 class="mt-4">垂直方向</h3>
    <Flex direction="column" gap="md">
      <div>项目 1</div>
      <div>项目 2</div>
      <div>项目 3</div>
    </Flex>
  </div>
</template>
```

## 对齐方式

设置主轴和交叉轴对齐。

```vue
<template>
  <div>
    <h3>主轴对齐</h3>
    <Flex justify="start" gap="md">开始对齐</Flex>
    <Flex justify="center" gap="md">居中对齐</Flex>
    <Flex justify="end" gap="md">结束对齐</Flex>
    <Flex justify="between" gap="md">两端对齐</Flex>
    <Flex justify="around" gap="md">分散对齐</Flex>

    <h3 class="mt-4">交叉轴对齐</h3>
    <Flex align="start" gap="md">开始对齐</Flex>
    <Flex align="center" gap="md">居中对齐</Flex>
    <Flex align="end" gap="md">结束对齐</Flex>
  </div>
</template>
```

## 间距

设置项目之间的间距。

```vue
<template>
  <div>
    <Flex gap="none">无间距</Flex>
    <Flex gap="sm">小间距</Flex>
    <Flex gap="md">中间距</Flex>
    <Flex gap="lg">大间距</Flex>
  </div>
</template>
```

## 换行

允许项目换行。

```vue
<template>
  <Flex wrap gap="md">
    <div v-for="i in 10" :key="i">项目 {{ i }}</div>
  </Flex>
</template>
```

## API

### Props

| 参数      | 说明           | 类型                                                    | 默认值    |
| --------- | -------------- | ------------------------------------------------------- | --------- |
| direction | 主轴方向       | `'row' \| 'column'`                                     | `'row'`   |
| wrap      | 是否换行       | `boolean`                                               | `false`   |
| justify   | 主轴对齐方式   | `'start' \| 'end' \| 'center' \| 'between' \| 'around'` | `'start'` |
| align     | 交叉轴对齐方式 | `'start' \| 'end' \| 'center' \| 'stretch'`             | `'start'` |
| gap       | 间距大小       | `'none' \| 'sm' \| 'md' \| 'lg'`                        | `'md'`    |

### Slots

| 名称    | 说明         |
| ------- | ------------ |
| default | 弹性布局内容 |

## 示例

### 卡片布局

```vue
<template>
  <Flex gap="md" wrap>
    <div class="card" v-for="i in 6" :key="i">卡片 {{ i }}</div>
  </Flex>
</template>

<style scoped>
.card {
  width: 200px;
  padding: 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
}
</style>
```

### 表单布局

```vue
<template>
  <Flex direction="column" gap="md">
    <Input label="用户名" />
    <Input label="密码" type="password" />
    <Flex justify="end" gap="md">
      <Button variant="secondary">取消</Button>
      <Button variant="primary">提交</Button>
    </Flex>
  </Flex>
</template>
```
