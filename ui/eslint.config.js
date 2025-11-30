import antfu from '@antfu/eslint-config';

export default antfu({
  // 启用 Vue 支持
  vue: true,

  // 启用 TypeScript 支持
  typescript: true,

  // 格式化配置 (替代 Prettier)
  formatters: {
    css: true,
    html: true,
    markdown: true,
  },

  // 忽略的文件
  ignores: [
    'dist',
    'node_modules',
    '*.min.js',
    'coverage',
  ],

  // 自定义规则
  rules: {
    // Vue 相关
    'vue/block-order': ['error', { order: ['template', 'script', 'style'] }],
    'vue/define-macros-order': ['error', { order: ['defineProps', 'defineEmits'] }],
    'vue/no-unused-refs': 'warn',

    // TypeScript 相关
    'ts/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    'ts/consistent-type-imports': ['error', { prefer: 'type-imports' }],

    // 通用规则
    'no-console': ['warn', { allow: ['warn', 'error'] }],
    'curly': ['error', 'multi-line'],
    'antfu/if-newline': 'off',

    // 样式规则
    'style/semi': ['error', 'always'],
    'style/quotes': ['error', 'single'],
    'style/comma-dangle': ['error', 'always-multiline'],
  },
});
