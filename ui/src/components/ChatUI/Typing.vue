<template>
  <div class="typing-indicator">
    <span class="typing-text">{{ text }}</span>
    <span class="typing-dots">
      <span class="dot"></span>
      <span class="dot"></span>
      <span class="dot"></span>
    </span>
  </div>
</template>

<script setup lang="ts">
interface Props {
  text?: string;
}

withDefaults(defineProps<Props>(), {
  text: "正在输入",
});
</script>

<style scoped>
.typing-indicator {
  @apply inline-flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 rounded-2xl;
}

.typing-text {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.typing-dots {
  @apply flex gap-1;
}

.dot {
  @apply w-1.5 h-1.5 bg-gray-400 dark:bg-gray-500 rounded-full;
  animation: typing 1.4s infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%,
  60%,
  100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-4px);
  }
}
</style>
