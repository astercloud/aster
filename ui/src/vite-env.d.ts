/// <reference types="vite/client" />

/**
 * Vite 环境变量类型声明
 * @see https://vitejs.dev/guide/env-and-mode.html#intellisense-for-typescript
 */
interface ImportMetaEnv {
  /** 演示模式开关 */
  readonly VITE_DEMO_MODE: string;
  /** 后端 API 地址 */
  readonly VITE_API_URL: string;
  /** WebSocket 地址 */
  readonly VITE_WS_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
