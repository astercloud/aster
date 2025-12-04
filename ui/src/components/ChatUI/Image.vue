<template>
  <div class="image-container">
    <img v-if="!error" :src="src" :alt="alt" :class="['image', sizeClass, shapeClass]" @load="handleLoad" @error="handleError" />
    <div v-if="loading" class="image-loading">
      <Icon type="loading" />
    </div>
    <div v-if="error" class="image-error">
      <Icon type="image" />
      <span class="error-text">加载失败</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import Icon from "./Icon.vue";

interface Props {
  src: string;
  alt?: string;
  size?: "sm" | "md" | "lg" | "full";
  shape?: "square" | "rounded" | "circle";
}

const props = withDefaults(defineProps<Props>(), {
  alt: "",
  size: "md",
  shape: "rounded",
});

const loading = ref(true);
const error = ref(false);

const sizeClass = computed(() => {
  const map = {
    sm: "w-16 h-16",
    md: "w-32 h-32",
    lg: "w-48 h-48",
    full: "w-full h-auto",
  };
  return map[props.size];
});

const shapeClass = computed(() => {
  const map = {
    square: "rounded-none",
    rounded: "rounded-lg",
    circle: "rounded-full",
  };
  return map[props.shape];
});

const handleLoad = () => {
  loading.value = false;
};

const handleError = () => {
  loading.value = false;
  error.value = true;
};
</script>

<style scoped>
.image-container {
  @apply relative inline-block;
}

.image {
  @apply object-cover;
}

.image-loading,
.image-error {
  @apply absolute inset-0 flex flex-col items-center justify-center bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500;
}

.error-text {
  @apply text-xs mt-2;
}
</style>
