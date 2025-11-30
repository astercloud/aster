<template>
  <div ref="containerRef" class="message-list" @scroll="handleScroll">
    <div class="message-list-inner">
      <!-- Load More Button -->
      <div v-if="hasMore && !isLoadingMore" class="load-more">
        <button
          @click="$emit('load-more')"
          class="load-more-button"
        >
          加载更多消息
        </button>
      </div>

      <!-- Loading Indicator -->
      <div v-if="isLoadingMore" class="loading-indicator">
        <svg class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
        </svg>
        <span>加载中...</span>
      </div>

      <!-- Empty State -->
      <div v-if="messages.length === 0 && !isTyping" class="empty-state">
        <svg class="w-16 h-16 text-secondary/30 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"></path>
        </svg>
        <p class="text-secondary">{{ config.emptyText || '开始对话...' }}</p>
      </div>

      <!-- Messages -->
      <div v-for="(group, index) in groupedMessages" :key="index" class="message-group">
        <!-- Date Divider -->
        <div v-if="group.date" class="date-divider">
          <span>{{ group.date }}</span>
        </div>

        <!-- Messages in Group -->
        <MessageBubble
          v-for="message in group.messages"
          :key="message.id"
          :message="message"
          :show-avatar="config.showAvatar"
          :show-timestamp="config.showTimestamp"
          @copy="handleCopy"
          @retry="handleRetry"
          @delete="handleDelete"
        />
      </div>

      <!-- Typing Indicator -->
      <div v-if="isTyping" class="typing-indicator">
        <div class="typing-avatar">
          <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
            <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
            </svg>
          </div>
        </div>
        <div class="typing-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>

      <!-- Scroll to Bottom Button -->
      <transition name="fade">
        <button
          v-if="showScrollButton"
          @click="scrollToBottom(true)"
          class="scroll-to-bottom"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3"></path>
          </svg>
        </button>
      </transition>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch, nextTick, onMounted } from 'vue';
import MessageBubble from './MessageBubble.vue';
import type { Message, ChatConfig } from '@/types';

export default defineComponent({
  name: 'MessageList',

  components: {
    MessageBubble,
  },

  props: {
    messages: {
      type: Array as () => Message[],
      required: true,
    },
    isTyping: {
      type: Boolean,
      default: false,
    },
    hasMore: {
      type: Boolean,
      default: false,
    },
    isLoadingMore: {
      type: Boolean,
      default: false,
    },
    config: {
      type: Object as () => ChatConfig,
      required: true,
    },
  },

  emits: {
    'load-more': () => true,
    copy: (message: Message) => true,
    retry: (message: Message) => true,
    delete: (message: Message) => true,
  },

  setup(props, { emit, expose }) {
    const containerRef = ref<HTMLElement>();
    const showScrollButton = ref(false);
    const isNearBottom = ref(true);

    // Group messages by date
    const groupedMessages = computed(() => {
      const groups: Array<{ date: string; messages: Message[] }> = [];
      let currentDate = '';
      let currentGroup: Message[] = [];

      props.messages.forEach((message) => {
        const messageDate = formatMessageDate(message.createdAt);

        if (messageDate !== currentDate) {
          if (currentGroup.length > 0) {
            groups.push({ date: currentDate, messages: currentGroup });
          }
          currentDate = messageDate;
          currentGroup = [message];
        } else {
          currentGroup.push(message);
        }
      });

      if (currentGroup.length > 0) {
        groups.push({ date: currentDate, messages: currentGroup });
      }

      return groups;
    });

    // Format date for divider
    function formatMessageDate(timestamp: number): string {
      const date = new Date(timestamp);
      const today = new Date();
      const yesterday = new Date(today);
      yesterday.setDate(yesterday.getDate() - 1);

      if (date.toDateString() === today.toDateString()) {
        return '今天';
      } else if (date.toDateString() === yesterday.toDateString()) {
        return '昨天';
      } else {
        return date.toLocaleDateString('zh-CN', {
          month: 'long',
          day: 'numeric',
        });
      }
    }

    // Handle scroll
    function handleScroll() {
      if (!containerRef.value) return;

      const { scrollTop, scrollHeight, clientHeight } = containerRef.value;
      const distanceFromBottom = scrollHeight - scrollTop - clientHeight;

      isNearBottom.value = distanceFromBottom < 100;
      showScrollButton.value = distanceFromBottom > 200;
    }

    // Scroll to bottom
    function scrollToBottom(smooth = true) {
      if (!containerRef.value) return;

      containerRef.value.scrollTo({
        top: containerRef.value.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto',
      });
    }

    // Handle message actions
    function handleCopy(message: Message) {
      if (message.type === 'text') {
        navigator.clipboard.writeText(message.content.text);
      }
      emit('copy', message);
    }

    function handleRetry(message: Message) {
      emit('retry', message);
    }

    function handleDelete(message: Message) {
      emit('delete', message);
    }

    // Auto scroll when new messages arrive
    watch(() => props.messages.length, async () => {
      if (isNearBottom.value || props.config.autoScroll !== false) {
        await nextTick();
        scrollToBottom();
      }
    });

    // Auto scroll when typing starts
    watch(() => props.isTyping, async (typing) => {
      if (typing && isNearBottom.value) {
        await nextTick();
        scrollToBottom();
      }
    });

    // Expose scroll method
    expose({
      scrollToBottom,
    });

    onMounted(() => {
      scrollToBottom(false);
    });

    return {
      containerRef,
      showScrollButton,
      groupedMessages,
      handleScroll,
      scrollToBottom,
      handleCopy,
      handleRetry,
      handleDelete,
    };
  },
});
</script>

<style scoped>
.message-list {
  @apply flex-1 overflow-y-auto px-4 py-6 relative;
}

.message-list-inner {
  @apply max-w-4xl mx-auto;
}

.load-more {
  @apply flex justify-center mb-4;
}

.load-more-button {
  @apply px-4 py-2 text-sm text-secondary hover:text-primary bg-surface hover:bg-background border border-border rounded-lg transition-colors;
}

.loading-indicator {
  @apply flex items-center justify-center gap-2 text-sm text-secondary mb-4;
}

.empty-state {
  @apply flex flex-col items-center justify-center h-full text-center;
}

.message-group {
  @apply mb-6;
}

.date-divider {
  @apply flex justify-center mb-4;
}

.date-divider span {
  @apply px-3 py-1 text-xs text-secondary bg-surface border border-border rounded-full;
}

.typing-indicator {
  @apply flex items-end gap-2 mb-4 animate-slide-in;
}

.typing-avatar {
  @apply flex-shrink-0;
}

.typing-dots {
  @apply flex items-center gap-1 bg-surface border border-border rounded-lg px-4 py-3;
}

.typing-dots span {
  @apply w-2 h-2 bg-secondary rounded-full animate-bounce;
}

.typing-dots span:nth-child(1) {
  animation-delay: 0ms;
}

.typing-dots span:nth-child(2) {
  animation-delay: 150ms;
}

.typing-dots span:nth-child(3) {
  animation-delay: 300ms;
}

.scroll-to-bottom {
  @apply fixed bottom-24 right-8 w-10 h-10 bg-primary text-white rounded-full shadow-lg hover:bg-primary-hover transition-all flex items-center justify-center;
}

.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity duration-200;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}
</style>
