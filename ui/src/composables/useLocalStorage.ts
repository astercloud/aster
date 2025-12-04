/**
 * useLocalStorage Composable
 * LocalStorage 响应式封装
 */

import { ref, watch, type Ref } from "vue";

export function useLocalStorage<T>(key: string, defaultValue: T): Ref<T> {
  // 读取初始值
  const storedValue = localStorage.getItem(key);
  const initialValue = storedValue ? JSON.parse(storedValue) : defaultValue;

  const value = ref<T>(initialValue) as Ref<T>;

  // 监听变化并保存
  watch(
    value,
    (newValue) => {
      try {
        localStorage.setItem(key, JSON.stringify(newValue));
      } catch (error) {
        console.error(`Error saving to localStorage: ${error}`);
      }
    },
    { deep: true },
  );

  // 监听其他标签页的变化
  if (typeof window !== "undefined") {
    window.addEventListener("storage", (e) => {
      if (e.key === key && e.newValue) {
        try {
          value.value = JSON.parse(e.newValue);
        } catch (error) {
          console.error(`Error parsing localStorage value: ${error}`);
        }
      }
    });
  }

  return value;
}
