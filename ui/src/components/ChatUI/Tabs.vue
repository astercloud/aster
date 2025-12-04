<template>
  <div class="tabs">
    <div class="tabs-header">
      <button v-for="tab in tabs" :key="tab.key" :class="['tab-item', { active: activeKey === tab.key }]" @click="handleTabClick(tab.key)">
        {{ tab.label }}
      </button>
    </div>
    <div class="tabs-content">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

interface Tab {
  key: string;
  label: string;
}

interface Props {
  tabs: Tab[];
  modelValue?: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:modelValue": [key: string];
  change: [key: string];
}>();

const activeKey = ref(props.modelValue || props.tabs[0]?.key);

watch(
  () => props.modelValue,
  (val) => {
    if (val) activeKey.value = val;
  },
);

const handleTabClick = (key: string) => {
  activeKey.value = key;
  emit("update:modelValue", key);
  emit("change", key);
};
</script>

<style scoped>
.tabs {
  @apply space-y-4;
}

.tabs-header {
  @apply flex gap-2 border-b border-gray-200 dark:border-gray-700;
}

.tab-item {
  @apply px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white border-b-2 border-transparent transition-colors;
}

.tab-item.active {
  @apply text-blue-600 dark:text-blue-400 border-blue-600 dark:border-blue-400;
}

.tabs-content {
  @apply py-4;
}
</style>
