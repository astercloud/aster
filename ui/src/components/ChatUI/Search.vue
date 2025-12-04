<template>
  <div class="search-container">
    <div class="search-input-wrapper">
      <Icon type="search" class="search-icon" />
      <input v-model="searchValue" type="text" :placeholder="placeholder" class="search-input" @input="handleInput" @keydown.enter="handleSearch" />
      <button v-if="searchValue" class="clear-button" @click="handleClear">
        <Icon type="close" size="sm" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import Icon from "./Icon.vue";

interface Props {
  modelValue?: string;
  placeholder?: string;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: "",
  placeholder: "搜索...",
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  search: [value: string];
}>();

const searchValue = ref(props.modelValue);

watch(
  () => props.modelValue,
  (val) => {
    searchValue.value = val;
  },
);

const handleInput = () => {
  emit("update:modelValue", searchValue.value);
};

const handleSearch = () => {
  emit("search", searchValue.value);
};

const handleClear = () => {
  searchValue.value = "";
  emit("update:modelValue", "");
  emit("search", "");
};
</script>

<style scoped>
.search-container {
  @apply w-full;
}

.search-input-wrapper {
  @apply relative flex items-center;
}

.search-icon {
  @apply absolute left-3 text-gray-400 dark:text-gray-500;
}

.search-input {
  @apply w-full pl-10 pr-10 py-2 bg-gray-100 dark:bg-gray-800 border border-transparent rounded-lg text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-600 focus:bg-white dark:focus:bg-gray-700 transition-colors;
}

.clear-button {
  @apply absolute right-3 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors;
}
</style>
