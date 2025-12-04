/**
 * useClipboard Composable
 * 剪贴板操作
 */

import { ref } from "vue";

export function useClipboard() {
  const isSupported = ref(typeof navigator !== "undefined" && "clipboard" in navigator);
  const copied = ref(false);
  const error = ref<Error | null>(null);

  async function copy(text: string): Promise<boolean> {
    if (!isSupported.value) {
      error.value = new Error("Clipboard API not supported");
      return false;
    }

    try {
      await navigator.clipboard.writeText(text);
      copied.value = true;
      error.value = null;

      // 2秒后重置状态
      setTimeout(() => {
        copied.value = false;
      }, 2000);

      return true;
    } catch (err) {
      error.value = err as Error;
      copied.value = false;
      return false;
    }
  }

  async function read(): Promise<string | null> {
    if (!isSupported.value) {
      error.value = new Error("Clipboard API not supported");
      return null;
    }

    try {
      const text = await navigator.clipboard.readText();
      error.value = null;
      return text;
    } catch (err) {
      error.value = err as Error;
      return null;
    }
  }

  return {
    isSupported,
    copied,
    error,
    copy,
    read,
  };
}
