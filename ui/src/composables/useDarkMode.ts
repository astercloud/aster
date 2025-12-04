/**
 * useDarkMode Composable
 * 暗色模式管理
 */

import { ref, onMounted } from "vue";

export type Theme = "light" | "dark" | "auto";

export function useDarkMode() {
  const theme = ref<Theme>("auto");
  const isDark = ref(false);

  // 检测系统主题
  const getSystemTheme = (): "light" | "dark" => {
    if (typeof window === "undefined") return "light";
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  };

  // 应用主题
  const applyTheme = (newTheme: Theme) => {
    const effectiveTheme = newTheme === "auto" ? getSystemTheme() : newTheme;
    isDark.value = effectiveTheme === "dark";

    if (typeof document !== "undefined") {
      if (isDark.value) {
        document.documentElement.classList.add("dark");
      } else {
        document.documentElement.classList.remove("dark");
      }
    }
  };

  // 设置主题
  const setTheme = (newTheme: Theme) => {
    theme.value = newTheme;
    applyTheme(newTheme);

    // 保存到 localStorage
    if (typeof localStorage !== "undefined") {
      localStorage.setItem("theme", newTheme);
    }
  };

  // 切换主题
  const toggleTheme = () => {
    const newTheme = isDark.value ? "light" : "dark";
    setTheme(newTheme);
  };

  // 监听系统主题变化
  const watchSystemTheme = () => {
    if (typeof window === "undefined") return;

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => {
      if (theme.value === "auto") {
        applyTheme("auto");
      }
    };

    mediaQuery.addEventListener("change", handler);
    return () => mediaQuery.removeEventListener("change", handler);
  };

  // 初始化
  onMounted(() => {
    // 从 localStorage 读取主题
    if (typeof localStorage !== "undefined") {
      const savedTheme = localStorage.getItem("theme") as Theme | null;
      if (savedTheme) {
        theme.value = savedTheme;
      }
    }

    // 应用主题
    applyTheme(theme.value);

    // 监听系统主题变化
    const cleanup = watchSystemTheme();
    return cleanup;
  });

  return {
    theme,
    isDark,
    setTheme,
    toggleTheme,
  };
}
