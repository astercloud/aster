<template>
  <div :class="['avatar', sizeClass, shapeClass]">
    <img v-if="src && !error" :src="src" :alt="alt" class="avatar-image" @error="handleError" />
    <div v-else class="avatar-placeholder">
      {{ placeholder }}
    </div>
    <span v-if="status" :class="['avatar-status', `status-${status}`]"></span>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

interface Props {
  src?: string;
  alt?: string;
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  shape?: "circle" | "square";
  status?: "online" | "offline" | "busy";
}

const props = withDefaults(defineProps<Props>(), {
  alt: "",
  size: "md",
  shape: "circle",
});

const error = ref(false);

const placeholder = computed(() => {
  return props.alt && props.alt.length > 0 ? props.alt[0]!.toUpperCase() : "?";
});

const sizeClass = computed(() => {
  const map = {
    xs: "w-6 h-6 text-xs",
    sm: "w-8 h-8 text-sm",
    md: "w-10 h-10 text-base",
    lg: "w-12 h-12 text-lg",
    xl: "w-16 h-16 text-xl",
  };
  return map[props.size];
});

const shapeClass = computed(() => {
  return props.shape === "circle" ? "rounded-full" : "rounded-lg";
});

const handleError = () => {
  error.value = true;
};
</script>

<style scoped>
.avatar {
  @apply relative inline-flex items-center justify-center overflow-hidden flex-shrink-0;
}

.avatar-image {
  @apply w-full h-full object-cover;
}

.avatar-placeholder {
  @apply w-full h-full flex items-center justify-center bg-gradient-to-br from-blue-400 to-blue-600 text-white font-semibold;
}

.avatar-status {
  @apply absolute bottom-0 right-0 w-3 h-3 border-2 border-white dark:border-gray-900 rounded-full;
}

.status-online {
  @apply bg-green-500;
}

.status-offline {
  @apply bg-gray-400;
}

.status-busy {
  @apply bg-red-500;
}
</style>
