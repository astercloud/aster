import { fileURLToPath } from 'node:url';
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      // 测试环境
      environment: 'happy-dom',

      // 全局 API (describe, it, expect 等)
      globals: true,

      // 包含的测试文件
      include: ['src/**/*.{test,spec}.{js,ts,vue}'],

      // 排除的文件
      exclude: ['node_modules', 'dist'],

      // 根目录
      root: fileURLToPath(new URL('./', import.meta.url)),

      // 覆盖率配置
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json', 'html'],
        include: ['src/**/*.{ts,vue}'],
        exclude: [
          'src/**/*.d.ts',
          'src/**/*.{test,spec}.ts',
          'src/main.ts',
          'src/router.ts',
        ],
      },

      // 设置文件
      setupFiles: ['./vitest.setup.ts'],
    },
  }),
);
