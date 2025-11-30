<template>
  <div v-if="replies.length > 0" class="quick-replies">
    <div class="quick-replies-inner">
      <div class="quick-replies-scroll">
        <button
          v-for="(reply, index) in replies"
          :key="index"
          @click="handleSelect(reply)"
          :class="['quick-reply-button', reply.isHighlight && 'highlight']"
        >
          <svg v-if="reply.icon" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="reply.icon"></path>
          </svg>
          <span>{{ reply.text }}</span>
          <span v-if="reply.isNew" class="new-badge">NEW</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import type { QuickReply } from '@/types';

export default defineComponent({
  name: 'QuickReplies',

  props: {
    replies: {
      type: Array as () => QuickReply[],
      required: true,
    },
  },

  emits: {
    select: (reply: QuickReply) => true,
  },

  setup(props, { emit }) {
    function handleSelect(reply: QuickReply) {
      emit('select', reply);
    }

    return {
      handleSelect,
    };
  },
});
</script>

<style scoped>
.quick-replies {
  @apply border-t border-border bg-surface;
}

.quick-replies-inner {
  @apply max-w-4xl mx-auto px-4 py-3;
}

.quick-replies-scroll {
  @apply flex gap-2 overflow-x-auto pb-2;
  scrollbar-width: thin;
}

.quick-replies-scroll::-webkit-scrollbar {
  @apply h-1;
}

.quick-replies-scroll::-webkit-scrollbar-track {
  @apply bg-transparent;
}

.quick-replies-scroll::-webkit-scrollbar-thumb {
  @apply bg-border rounded-full;
}

.quick-reply-button {
  @apply flex items-center gap-2 px-4 py-2 bg-background hover:bg-stone-100 border border-border rounded-full text-sm font-medium text-primary whitespace-nowrap transition-all flex-shrink-0;
}

.quick-reply-button:hover {
  @apply shadow-sm;
}

.quick-reply-button:active {
  @apply scale-95;
}

.quick-reply-button.highlight {
  @apply bg-primary text-white border-primary hover:bg-primary-hover;
}

.new-badge {
  @apply ml-1 px-1.5 py-0.5 bg-red-500 text-white text-xs rounded;
}
</style>
