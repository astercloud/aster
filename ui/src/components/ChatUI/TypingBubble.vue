<template>
  <div class="typing-bubble">
    <div class="typing-avatar">
      <div class="avatar-placeholder"></div>
    </div>
    <div class="typing-indicator">
      <span class="typing-dot"></span>
      <span class="typing-dot"></span>
      <span class="typing-dot"></span>
    </div>
  </div>
</template>

<style scoped>
.typing-bubble {
  @apply flex items-center gap-2;
}

.typing-avatar {
  @apply w-8 h-8 rounded-full overflow-hidden flex-shrink-0;
}

.avatar-placeholder {
  @apply w-full h-full bg-gray-200 dark:bg-gray-700;
}

.typing-indicator {
  @apply flex items-center gap-1 px-4 py-3 bg-gray-100 dark:bg-gray-800 rounded-2xl;
  border-bottom-left-radius: 4px;
}

.typing-dot {
  @apply w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full;
  animation: typing 1.4s infinite;
}

.typing-dot:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes typing {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.7;
  }
  30% {
    transform: translateY(-8px);
    opacity: 1;
  }
}
</style>
