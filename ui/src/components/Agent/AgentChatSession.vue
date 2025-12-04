<template>
  <div class="agent-chat-session">
    <!-- Header -->
    <div class="session-header">
      <button @click="$emit('back')" class="back-button">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
        </svg>
        返回
      </button>

      <div class="agent-info">
        <div class="agent-avatar">
          <img v-if="agent.avatar" :src="agent.avatar" :alt="agent.name" />
          <div v-else class="avatar-placeholder">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
              ></path>
            </svg>
          </div>
        </div>
        <div>
          <h2 class="agent-name">{{ agent.name }}</h2>
          <p v-if="agent.description" class="agent-description">{{ agent.description }}</p>
        </div>
      </div>

      <div class="status-indicator">
        <span :class="['status-dot', statusClass]"></span>
        {{ statusText }}
      </div>
    </div>

    <!-- Chat Area -->
    <div class="chat-container">
      <AsterChat :config="chatConfig" :show-header="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import AsterChat from "../AsterChat.vue";
import type { Agent } from "@/types";

interface Props {
  agent: Agent;
}

const props = defineProps<Props>();

defineEmits<{
  back: [];
}>();

const statusClass = computed(() => {
  const classes: Record<string, string> = {
    idle: "status-idle",
    thinking: "status-thinking",
    busy: "status-busy",
    error: "status-error",
  };
  return classes[props.agent.status] || "status-idle";
});

const statusText = computed(() => {
  const texts: Record<string, string> = {
    idle: "空闲",
    thinking: "思考中",
    busy: "忙碌",
    error: "错误",
  };
  return texts[props.agent.status] || "未知";
});

const chatConfig = computed(() => ({
  agentId: props.agent.id,
  agentProfile: {
    name: props.agent.name,
    description: props.agent.description,
  },
  placeholder: `与 ${props.agent.name} 对话...`,
  enableThinking: true,
  enableApproval: true,
  enableQuickReplies: true,
  demoMode: false,
  apiUrl: "http://localhost:8080",
  modelConfig: {
    provider: props.agent.metadata?.provider || "anthropic",
    model: props.agent.metadata?.model || "claude-3-5-sonnet-20241022",
  },
}));
</script>

<style scoped>
.agent-chat-session {
  @apply h-screen flex flex-col bg-background dark:bg-background-dark;
}

.session-header {
  @apply flex items-center gap-4 px-6 py-4 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.back-button {
  @apply flex items-center gap-2 px-3 py-2 text-sm font-medium text-secondary dark:text-secondary-dark hover:text-text dark:hover:text-text-dark hover:bg-background dark:hover:bg-background-dark rounded-lg transition-colors;
}

.agent-info {
  @apply flex items-center gap-3 flex-1;
}

.agent-avatar {
  @apply w-10 h-10 rounded-full overflow-hidden flex-shrink-0;
}

.agent-avatar img {
  @apply w-full h-full object-cover;
}

.avatar-placeholder {
  @apply w-full h-full bg-primary/10 dark:bg-primary/20 flex items-center justify-center text-primary dark:text-primary-light;
}

.agent-name {
  @apply text-lg font-semibold text-text dark:text-text-dark;
}

.agent-description {
  @apply text-sm text-secondary dark:text-secondary-dark;
}

.status-indicator {
  @apply flex items-center gap-2 text-sm font-medium text-secondary dark:text-secondary-dark;
}

.status-dot {
  @apply w-2 h-2 rounded-full;
}

.status-dot.status-idle {
  @apply bg-gray-500 dark:bg-gray-400;
}

.status-dot.status-thinking {
  @apply bg-blue-500 dark:bg-blue-400 animate-pulse;
}

.status-dot.status-busy {
  @apply bg-amber-500 dark:bg-amber-400 animate-pulse;
}

.status-dot.status-error {
  @apply bg-red-500 dark:bg-red-400;
}

.chat-container {
  @apply flex-1 overflow-hidden;
}
</style>
