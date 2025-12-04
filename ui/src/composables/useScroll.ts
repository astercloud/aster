import { ref, onMounted, onUnmounted } from "vue";

/**
 * useScroll
 * 监听元素滚动并提供滚动控制方法
 */
export function useScroll<T extends HTMLElement>() {
  const target = ref<T | null>(null);
  const isAtBottom = ref(true);
  const isAtTop = ref(true);

  const updateState = () => {
    if (!target.value) return;
    const { scrollTop, scrollHeight, clientHeight } = target.value;
    isAtTop.value = scrollTop <= 0;
    isAtBottom.value = scrollTop + clientHeight >= scrollHeight - 10;
  };

  const scrollToTop = (smooth = true) => {
    if (!target.value) return;
    target.value.scrollTo({ top: 0, behavior: smooth ? "smooth" : "auto" });
  };

  const scrollToBottom = (smooth = true) => {
    if (!target.value) return;
    target.value.scrollTo({
      top: target.value.scrollHeight,
      behavior: smooth ? "smooth" : "auto",
    });
  };

  onMounted(() => {
    if (!target.value) return;
    target.value.addEventListener("scroll", updateState);
  });

  onUnmounted(() => {
    if (!target.value) return;
    target.value.removeEventListener("scroll", updateState);
  });

  return {
    target,
    isAtBottom,
    isAtTop,
    scrollToTop,
    scrollToBottom,
    refresh: updateState,
  };
}
