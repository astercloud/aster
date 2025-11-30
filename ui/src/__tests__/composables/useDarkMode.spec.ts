import { describe, expect, it, vi } from 'vitest';
import { useDarkMode } from '@/composables/useDarkMode';

describe('useDarkMode', () => {
  it('should initialize with auto theme', () => {
    const { theme, isDark } = useDarkMode();

    expect(theme.value).toBe('auto');
    expect(typeof isDark.value).toBe('boolean');
  });

  it('should toggle theme', () => {
    const { isDark, toggleTheme } = useDarkMode();

    const initialDark = isDark.value;
    toggleTheme();

    expect(isDark.value).toBe(!initialDark);
  });

  it('should set specific theme', () => {
    const { theme, isDark, setTheme } = useDarkMode();

    setTheme('dark');
    expect(theme.value).toBe('dark');
    expect(isDark.value).toBe(true);

    setTheme('light');
    expect(theme.value).toBe('light');
    expect(isDark.value).toBe(false);
  });
});
