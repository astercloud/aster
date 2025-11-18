<template>
  <header class="app-header">
    <div class="container mx-auto px-4">
      <div class="flex items-center justify-between h-16">
        <NuxtLink to="/" class="flex items-center gap-3">
          <img src="/images/logo-icon.svg" alt="Aster Logo" class="h-10 w-10" />
          <div class="flex flex-col">
            <span class="text-xl font-bold bg-gradient-to-r from-primary-600 to-blue-600 bg-clip-text text-transparent">
              Aster · 星尘云枢
            </span>
            <span class="text-xs text-gray-500">星尘汇聚，智能成枢</span>
          </div>
        </NuxtLink>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
}

.dark .app-header {
  border-bottom-color: rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.95);
}
</style>
