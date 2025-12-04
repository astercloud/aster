# RoomCard 房间卡片

用于展示多 Agent 协作房间的卡片组件。

## 基础用法

基本的房间卡片。

```vue
<template>
  <RoomCard :room="room" @join="handleJoin" @leave="handleLeave" />
</template>

<script setup>
const room = {
  id: "1",
  name: "产品设计讨论",
  members: [
    { name: "产品经理 Agent", agentId: "pm-1", status: "online" },
    { name: "设计师 Agent", agentId: "designer-1", status: "online" },
    { name: "开发 Agent", agentId: "dev-1", status: "busy" },
  ],
  createdAt: Date.now(),
};

const handleJoin = (room) => {
  console.log("Join room:", room.name);
};

const handleLeave = (room) => {
  console.log("Leave room:", room.id);
};
</script>
```

## 显示成员

显示房间内的 Agent 成员。

```vue
<template>
  <RoomCard
    :room="{
      id: '1',
      name: '技术讨论室',
      members: [
        {
          name: '架构师 Agent',
          agentId: 'arch-1',
          avatar: '/avatars/arch.jpg',
          status: 'online',
        },
        {
          name: '前端 Agent',
          agentId: 'fe-1',
          status: 'online',
        },
        {
          name: '后端 Agent',
          agentId: 'be-1',
          status: 'busy',
        },
      ],
      createdAt: Date.now(),
    }"
  />
</template>
```

## API

### Props

| 参数 | 说明     | 类型   | 默认值 |
| ---- | -------- | ------ | ------ |
| room | 房间对象 | `Room` | -      |

### Room 类型

```typescript
interface Room {
  id: string;
  name: string;
  members: RoomMember[];
  createdAt: number;
  metadata?: Record<string, any>;
}

interface RoomMember {
  name: string;
  agentId: string;
  avatar?: string;
  status?: "online" | "offline" | "busy";
}
```

### Events

| 事件名 | 说明               | 回调参数     |
| ------ | ------------------ | ------------ |
| join   | 点击加入按钮时触发 | `room: Room` |
| leave  | 点击离开按钮时触发 | `room: Room` |
| edit   | 点击编辑按钮时触发 | `room: Room` |

## 使用场景

- 多 Agent 协作
- Agent 团队管理
- 协作空间展示
- Agent 社交

## 示例

### 房间列表

```vue
<template>
  <div class="grid grid-cols-2 gap-4">
    <RoomCard v-for="room in rooms" :key="room.id" :room="room" @join="joinRoom" />
  </div>
</template>

<script setup>
import { ref } from "vue";

const rooms = ref([
  {
    id: "1",
    name: "产品策划室",
    members: [
      { name: "产品 Agent", agentId: "pm-1", status: "online" },
      { name: "市场 Agent", agentId: "mkt-1", status: "online" },
    ],
    createdAt: Date.now(),
  },
  {
    id: "2",
    name: "技术攻关组",
    members: [
      { name: "架构 Agent", agentId: "arch-1", status: "busy" },
      { name: "开发 Agent", agentId: "dev-1", status: "online" },
      { name: "测试 Agent", agentId: "qa-1", status: "online" },
    ],
    createdAt: Date.now(),
  },
]);

const joinRoom = (room) => {
  console.log("Joining room:", room.name);
};
</script>
```
