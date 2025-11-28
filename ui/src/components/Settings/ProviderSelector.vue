<template>
  <div class="provider-selector">
    <div class="selector-group">
      <label class="label">Provider</label>
      <select v-model="selectedProvider" class="select-provider">
        <option value="deepseek">DeepSeek</option>
        <option value="anthropic">Anthropic (Claude)</option>
        <option value="openai">OpenAI (GPT)</option>
        <option value="glm">GLM (智谱AI)</option>
      </select>
    </div>

    <div class="selector-group">
      <label class="label">Model</label>
      <select v-model="selectedModel" class="select-model">
        <option v-for="model in availableModels" :key="model" :value="model">
          {{ model }}
        </option>
      </select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';

const selectedProvider = ref('deepseek');
const selectedModel = ref('deepseek-chat');

const emit = defineEmits<{
  change: [{ provider: string; model: string }];
}>();

// 最新模型列表 (2025-01更新)
const modelMap: Record<string, string[]> = {
  deepseek: [
    'deepseek-chat',      // DeepSeek-V3.1 (Non-thinking Mode)
    'deepseek-reasoner',  // DeepSeek-V3.1 (Thinking Mode)
  ],
  anthropic: [
    'claude-opus-4-0',              // Most capable model
    'claude-opus-4-1',              // Latest opus with strict tool use
    'claude-3-7-sonnet-latest',     // High-performance with extended thinking
    'claude-3-5-sonnet-latest',     // Balanced performance
    'claude-3-sonnet-20250219',     // Specific snapshot
    'claude-3-5-haiku-latest',      // Fastest and most compact
    'claude-3-5-sonnet-20241022',   // Previous sonnet
    'claude-3-opus-20240229',       // Previous opus
  ],
  openai: [
    'gpt-5.1',        // Latest flagship (complex reasoning, broad knowledge)
    'gpt-5',          // Previous flagship
    'gpt-5-mini',     // Cost-optimized reasoning (balance speed/cost/capability)
    'gpt-5-nano',     // High-throughput (simple tasks, classification)
    'gpt-4.1',        // Solid combination of intelligence/speed/cost
    'gpt-4o-mini',    // Previous mini model
    'gpt-4-turbo',    // Previous turbo model
  ],
  glm: [
    'glm-4',
    'glm-4-plus',
    'glm-3-turbo',
  ],
};

const availableModels = computed(() => modelMap[selectedProvider.value] || []);

// 当 provider 改变时，自动选择第一个 model
watch(selectedProvider, () => {
  selectedModel.value = availableModels.value[0] || '';
});

// 当 provider 或 model 改变时，通知父组件
watch([selectedProvider, selectedModel], () => {
  emit('change', {
    provider: selectedProvider.value,
    model: selectedModel.value,
  });
});
</script>

<style scoped>
.provider-selector {
  @apply space-y-3 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700;
}

.selector-group {
  @apply space-y-2;
}

.label {
  @apply block text-sm font-semibold text-gray-700 dark:text-gray-300;
}

.select-provider,
.select-model {
  @apply w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
         bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100
         focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
         transition-colors;
}

.select-provider:hover,
.select-model:hover {
  @apply border-gray-400 dark:border-gray-500;
}
</style>
