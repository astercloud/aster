<template>
  <div ref="scrollRef" class="scroll-view" @scroll="handleScroll">
    <slot></slot>

    <button v-if="showBackTop && showButton" class="back-top-btn" @click="scrollToTop">
      <Icon type="arrow-up" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import Icon from "./Icon.vue";

interface Props {
  showBackTop?: boolean;
  threshold?: number;
}

const props = withDefaults(defineProps<Props>(), {
  showBackTop: true,
  threshold: 300,
});

const emit = defineEmits<{
  scroll: [event: Event];
  reachBottom: [];
}>();

const scrollRef = ref<HTMLDivElement>();
const showButton = ref(false);

const handleScroll = (e: Event) => {
  const target = e.target as HTMLDivElement;

  // 显示/隐藏回到顶部按钮
  showButton.value = target.scrollTop > props.threshold;

  // 检测是否到达底部
  const isBottom = target.scrollHeight - target.scrollTop <= target.clientHeight + 50;
  if (isBottom) {
    emit("reachBottom");
  }

  emit("scroll", e);
};

const scrollToTop = () => {
  scrollRef.value?.scrollTo({
    top: 0,
    behavior: "smooth",
  });
};

const scrollToBottom = () => {
  scrollRef.value?.scrollTo({
    top: scrollRef.value.scrollHeight,
    behavior: "smooth",
  });
};

defineExpose({
  scrollToTop,
  scrollToBottom,
});
</script>

<style scoped>
.scroll-view {
  @apply relative h-full overflow-y-auto;
}

.back-top-btn {
  @apply fixed bottom-8 right-8 z-40 p-3 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 border border-gray-200 dark:border-gray-700 rounded-full shadow-lg transition-all;
  animation: fadeIn 0.3s;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
