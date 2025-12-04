<template>
  <div :class="['message-item flex animate-slide-in', message.role === 'user' ? 'justify-end' : 'justify-start']">
    <div :class="['max-w-[80%]', message.role === 'user' ? 'items-end' : 'items-start']">
      <!-- Thinking Block -->
      <ThinkingBlock v-if="message.thoughts && message.thoughts.length > 0 && showThinking" :thoughts="message.thoughts" :is-finished="!!message.content" @approve="$emit('approve', $event)" @reject="$emit('reject', $event)" class="mb-2" />

      <!-- Message Bubble -->
      <div :class="['rounded-lg px-4 py-3 shadow-sm', message.role === 'user' ? 'bg-primary text-white' : 'bg-surface dark:bg-surface-dark text-text dark:text-text-dark border border-border dark:border-border-dark']">
        <!-- Attachments -->
        <div v-if="message.attachments && message.attachments.length > 0" class="mb-2">
          <div v-for="attachment in message.attachments" :key="attachment.id" class="mb-2">
            <img v-if="attachment.type === 'image'" :src="attachment.url || attachment.preview" :alt="attachment.name" class="max-w-full rounded" style="max-height: 200px" />
          </div>
        </div>

        <!-- Content -->
        <div v-if="message.content" class="markdown-body" v-html="renderedContent"></div>

        <!-- Loading -->
        <div v-else-if="!message.content && message.role === 'assistant'" class="flex items-center gap-1">
          <span class="w-2 h-2 bg-secondary rounded-full animate-bounce"></span>
          <span class="w-2 h-2 bg-secondary rounded-full animate-bounce" style="animation-delay: 0.1s"></span>
          <span class="w-2 h-2 bg-secondary rounded-full animate-bounce" style="animation-delay: 0.2s"></span>
        </div>

        <!-- Timestamp -->
        <div :class="['text-xs mt-2', message.role === 'user' ? 'text-white/70' : 'text-secondary']">
          {{ formatTime(message.createdAt) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { marked } from "marked";
import ThinkingBlock from "./ThinkingBlock.vue";
import type { Message } from "@/types";

interface Props {
  message: Message;
  showThinking?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showThinking: true,
});

defineEmits<{
  approve: [requestId: string];
  reject: [requestId: string];
}>();

const renderedContent = computed(() => {
  if (!props.message.content) return "";
  let raw = "";
  if (typeof props.message.content === "string") {
    raw = props.message.content;
  } else if ("text" in props.message.content && typeof props.message.content.text === "string") {
    raw = props.message.content.text;
  } else {
    raw = JSON.stringify(props.message.content, null, 2);
  }
  try {
    return marked.parse(raw);
  } catch (error) {
    return raw;
  }
});

const formatTime = (timestamp?: number) => {
  const date = new Date(timestamp || Date.now());
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
};
</script>

<style scoped>
.message-item {
  margin-bottom: 1.5rem;
}
</style>
