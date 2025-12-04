<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
    <!-- Header -->
    <header class="border-b border-gray-200 dark:border-gray-700 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm sticky top-0 z-10">
      <div class="max-w-7xl mx-auto px-6 py-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
              <span class="text-white font-bold text-xl">A</span>
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Aster Agent UI</h1>
              <p class="text-sm text-gray-600 dark:text-gray-400">专为 AI Agent 设计的组件库</p>
            </div>
          </div>
          <div class="flex gap-3">
            <a href="https://github.com/astercloud/aster" target="_blank" class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"> GitHub </a>
          </div>
        </div>
      </div>
    </header>

    <!-- Hero Section -->
    <section class="max-w-7xl mx-auto px-6 py-16">
      <div class="text-center mb-12">
        <h2 class="text-5xl font-bold text-gray-900 dark:text-white mb-4">构建强大的 AI Agent 应用</h2>
        <p class="text-xl text-gray-600 dark:text-gray-400 max-w-3xl mx-auto">完整的 Vue 3 组件库，包含 Agent 管理、对话界面、工作流可视化等功能</p>
      </div>

      <!-- Quick Stats -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-6 mb-16">
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div class="text-3xl font-bold text-blue-600 dark:text-blue-400 mb-2">33+</div>
          <div class="text-gray-600 dark:text-gray-400">组件</div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div class="text-3xl font-bold text-purple-600 dark:text-purple-400 mb-2">6</div>
          <div class="text-gray-600 dark:text-gray-400">Agent 专属</div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div class="text-3xl font-bold text-green-600 dark:text-green-400 mb-2">100%</div>
          <div class="text-gray-600 dark:text-gray-400">TypeScript</div>
        </div>
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div class="text-3xl font-bold text-orange-600 dark:text-orange-400 mb-2">Vue 3</div>
          <div class="text-gray-600 dark:text-gray-400">Composition API</div>
        </div>
      </div>
    </section>

    <!-- Navigation Cards -->
    <section class="max-w-7xl mx-auto px-6 pb-16">
      <h3 class="text-3xl font-bold text-gray-900 dark:text-white mb-8">快速导航</h3>

      <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        <!-- Agent Demo -->
        <router-link to="/agent-demo" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-blue-500 dark:hover:border-blue-400 transition-all">
          <div class="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">🤖</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">Agent 聊天演示</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">体验完整的 Agent 对话功能，包括思考过程、工具调用等</p>
          <div class="text-blue-600 dark:text-blue-400 font-medium group-hover:translate-x-1 transition-transform inline-block">立即体验 →</div>
        </router-link>

        <!-- Component Docs -->
        <router-link to="/docs" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-purple-500 dark:hover:border-purple-400 transition-all">
          <div class="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">📖</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">交互式文档</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">查看所有组件的文档和实时演示</p>
          <div class="text-purple-600 dark:text-purple-400 font-medium group-hover:translate-x-1 transition-transform inline-block">查看文档 →</div>
        </router-link>

        <!-- Components Showcase -->
        <router-link to="/components" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-green-500 dark:hover:border-green-400 transition-all">
          <div class="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">🎨</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">组件展示</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">浏览所有可用组件的视觉效果</p>
          <div class="text-green-600 dark:text-green-400 font-medium group-hover:translate-x-1 transition-transform inline-block">浏览组件 →</div>
        </router-link>

        <!-- Agent Management -->
        <router-link to="/agents" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-orange-500 dark:hover:border-orange-400 transition-all">
          <div class="w-12 h-12 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">⚙️</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">Agent 管理</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">创建、配置和管理你的 AI Agents</p>
          <div class="text-orange-600 dark:text-orange-400 font-medium group-hover:translate-x-1 transition-transform inline-block">管理 Agents →</div>
        </router-link>

        <!-- Workflows -->
        <router-link to="/workflows" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-pink-500 dark:hover:border-pink-400 transition-all">
          <div class="w-12 h-12 bg-pink-100 dark:bg-pink-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">🔄</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">工作流</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">可视化 Agent 工作流和执行步骤</p>
          <div class="text-pink-600 dark:text-pink-400 font-medium group-hover:translate-x-1 transition-transform inline-block">查看工作流 →</div>
        </router-link>

        <!-- Rooms -->
        <router-link to="/rooms" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-indigo-500 dark:hover:border-indigo-400 transition-all">
          <div class="w-12 h-12 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">👥</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">协作房间</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">多 Agent 协作和团队工作空间</p>
          <div class="text-indigo-600 dark:text-indigo-400 font-medium group-hover:translate-x-1 transition-transform inline-block">进入房间 →</div>
        </router-link>

        <!-- Projects -->
        <router-link to="/projects" class="group bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-lg hover:border-yellow-500 dark:hover:border-yellow-400 transition-all">
          <div class="w-12 h-12 bg-yellow-100 dark:bg-yellow-900/30 rounded-lg flex items-center justify-center mb-4 group-hover:scale-110 transition-transform">
            <span class="text-2xl">📁</span>
          </div>
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">项目管理</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">AI 写作项目和会话管理</p>
          <div class="text-yellow-600 dark:text-yellow-400 font-medium group-hover:translate-x-1 transition-transform inline-block">管理项目 →</div>
        </router-link>
      </div>
    </section>

    <!-- Features -->
    <section class="max-w-7xl mx-auto px-6 pb-16">
      <h3 class="text-3xl font-bold text-gray-900 dark:text-white mb-8">核心特性</h3>

      <div class="grid md:grid-cols-2 gap-8">
        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">🤖</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Agent 专属组件</h4>
            <p class="text-gray-600 dark:text-gray-400">专为 AI Agent 场景设计，包括 AgentCard、ThinkingBlock、WorkflowTimeline 等</p>
          </div>
        </div>

        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">💬</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">完整对话体验</h4>
            <p class="text-gray-600 dark:text-gray-400">支持文本、图片、文件等多模态消息，Markdown 渲染，代码高亮</p>
          </div>
        </div>

        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">🔄</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">工作流可视化</h4>
            <p class="text-gray-600 dark:text-gray-400">实时展示 Agent 执行步骤，支持状态追踪和错误处理</p>
          </div>
        </div>

        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">🎨</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">现代设计</h4>
            <p class="text-gray-600 dark:text-gray-400">基于 Tailwind CSS，支持深色模式，响应式布局</p>
          </div>
        </div>

        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-pink-100 dark:bg-pink-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">💪</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">TypeScript 支持</h4>
            <p class="text-gray-600 dark:text-gray-400">完整的类型定义，提供优秀的开发体验</p>
          </div>
        </div>

        <div class="flex gap-4">
          <div class="flex-shrink-0 w-10 h-10 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg flex items-center justify-center">
            <span class="text-xl">⚡</span>
          </div>
          <div>
            <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">高性能</h4>
            <p class="text-gray-600 dark:text-gray-400">基于 Vue 3 Composition API，优化的渲染性能</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Quick Start -->
    <section class="max-w-7xl mx-auto px-6 pb-16">
      <div class="bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl p-8 text-white">
        <h3 class="text-3xl font-bold mb-4">快速开始</h3>
        <p class="text-lg mb-6 opacity-90">5 分钟创建你的第一个 Agent 应用</p>
        <div class="bg-black/20 rounded-lg p-4 mb-6 font-mono text-sm">
          <div class="text-gray-300"># 安装依赖</div>
          <div>npm install marked</div>
          <div class="mt-2 text-gray-300"># 启动开发服务器</div>
          <div>npm run dev</div>
        </div>
        <div class="flex gap-4">
          <router-link to="/agent-demo" class="px-6 py-3 bg-white text-blue-600 rounded-lg font-semibold hover:bg-gray-100 transition-colors"> 查看演示 </router-link>
          <router-link to="/docs" class="px-6 py-3 bg-white/10 backdrop-blur-sm border border-white/20 rounded-lg font-semibold hover:bg-white/20 transition-colors"> 阅读文档 </router-link>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="border-t border-gray-200 dark:border-gray-700 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm">
      <div class="max-w-7xl mx-auto px-6 py-8">
        <div class="flex flex-col md:flex-row justify-between items-center gap-4">
          <div class="text-gray-600 dark:text-gray-400">© 2024 Aster Cloud. MIT License.</div>
          <div class="flex gap-6">
            <a href="https://github.com/astercloud/aster" target="_blank" class="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"> GitHub </a>
            <router-link to="/docs" class="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"> 文档 </router-link>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
// Home page - Navigation hub for Aster Agent UI
</script>
