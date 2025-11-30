<template>
  <div class="room-list">
    <!-- Header -->
    <div class="list-header">
      <div class="header-left">
        <h2 class="list-title">Room 列表</h2>
        <span class="room-count">{{ rooms.length }} 个</span>
      </div>

      <div class="header-right">
        <button @click="$emit('create')" class="btn-create">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
          </svg>
          创建 Room
        </button>
      </div>
    </div>

    <!-- Search -->
    <div class="list-search">
      <svg class="search-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
      </svg>
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索 Room..."
        class="search-input"
      />
    </div>

    <!-- Loading -->
    <LoadingSpinner v-if="loading" class="my-8" />

    <!-- Empty State -->
    <EmptyState
      v-else-if="filteredRooms.length === 0"
      title="暂无 Room"
      description="还没有创建任何 Room，点击上方按钮创建第一个"
    >
      <template #action>
        <button @click="$emit('create')" class="btn-create">
          创建 Room
        </button>
      </template>
    </EmptyState>

    <!-- Room List -->
    <div v-else class="room-grid">
      <div
        v-for="room in filteredRooms"
        :key="room.id"
        class="room-card"
        @click="$emit('select', room)"
      >
        <div class="room-header">
          <div class="room-icon">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
            </svg>
          </div>

          <div class="room-info">
            <h3 class="room-name">{{ room.name }}</h3>
            <p class="room-members">{{ room.members.length }} 个成员</p>
          </div>
        </div>

        <div v-if="room.metadata?.description" class="room-description">
          {{ room.metadata.description }}
        </div>

        <div class="room-footer">
          <span class="room-time">
            {{ formatTime(room.createdAt) }}
          </span>
          <button
            @click.stop="$emit('delete', room)"
            class="btn-delete"
            title="删除"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, type PropType } from 'vue';
import LoadingSpinner from '../Common/LoadingSpinner.vue';
import EmptyState from '../Common/EmptyState.vue';
import type { Room } from '@/types';

export default defineComponent({
  name: 'RoomList',
  components: {
    LoadingSpinner,
    EmptyState,
  },
  props: {
    rooms: {
      type: Array as PropType<Room[]>,
      required: true,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: {
    create: () => true,
    select: (room: Room) => true,
    delete: (room: Room) => true,
  },
  setup(props) {
    const searchQuery = ref('');

    const filteredRooms = computed(() => {
      if (!searchQuery.value) return props.rooms;

      const query = searchQuery.value.toLowerCase();
      return props.rooms.filter(room =>
        room.name.toLowerCase().includes(query)
      );
    });

    function formatTime(timestamp: number): string {
      const date = new Date(timestamp);
      const now = new Date();
      const diff = now.getTime() - date.getTime();
      const days = Math.floor(diff / (1000 * 60 * 60 * 24));

      if (days === 0) return '今天';
      if (days === 1) return '昨天';
      if (days < 7) return `${days} 天前`;

      return date.toLocaleDateString('zh-CN');
    }

    return {
      searchQuery,
      filteredRooms,
      formatTime,
    };
  },
});
</script>

<style scoped>
.room-list {
  @apply space-y-4;
}

.list-header {
  @apply flex items-center justify-between;
}

.header-left {
  @apply flex items-center gap-3;
}

.list-title {
  @apply text-2xl font-bold text-text dark:text-text-dark;
}

.room-count {
  @apply px-2 py-1 bg-primary/10 dark:bg-primary/20 text-primary dark:text-primary-light text-sm font-medium rounded-full;
}

.btn-create {
  @apply flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg font-medium transition-colors;
}

.list-search {
  @apply relative;
}

.search-icon {
  @apply absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-secondary dark:text-secondary-dark;
}

.search-input {
  @apply w-full pl-10 pr-4 py-2 bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary/20;
}

.room-grid {
  @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4;
}

.room-card {
  @apply bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg p-4 hover:shadow-md transition-all cursor-pointer;
}

.room-header {
  @apply flex items-start gap-3 mb-3;
}

.room-icon {
  @apply w-12 h-12 rounded-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center text-primary dark:text-primary-light flex-shrink-0;
}

.room-info {
  @apply flex-1 min-w-0;
}

.room-name {
  @apply text-base font-semibold text-text dark:text-text-dark truncate;
}

.room-members {
  @apply text-sm text-secondary dark:text-secondary-dark mt-1;
}

.room-description {
  @apply text-sm text-secondary dark:text-secondary-dark mb-3 line-clamp-2;
}

.room-footer {
  @apply flex items-center justify-between pt-3 border-t border-border dark:border-border-dark;
}

.room-time {
  @apply text-xs text-secondary dark:text-secondary-dark;
}

.btn-delete {
  @apply p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 dark:text-red-400 rounded transition-colors;
}
</style>
