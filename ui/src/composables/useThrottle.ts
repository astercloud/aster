/**
 * useThrottle Composable
 * 节流处理
 */

import { ref, watch, type Ref } from "vue";

export function useThrottle<T>(value: Ref<T>, delay: number = 300): Ref<T> {
  const throttledValue = ref<T>(value.value) as Ref<T>;
  let lastUpdate = 0;

  watch(value, (newValue) => {
    const now = Date.now();

    if (now - lastUpdate >= delay) {
      throttledValue.value = newValue;
      lastUpdate = now;
    }
  });

  return throttledValue;
}

export function useThrottleFn<T extends (...args: any[]) => any>(fn: T, delay: number = 300): (...args: Parameters<T>) => void {
  let lastCall = 0;

  return function (...args: Parameters<T>) {
    const now = Date.now();

    if (now - lastCall >= delay) {
      fn(...args);
      lastCall = now;
    }
  };
}
