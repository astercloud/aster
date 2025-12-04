<template>
  <nav class="navbar">
    <div class="navbar-brand">
      <router-link to="/" class="brand-link" v-if="shouldShowHomeLink">
        <span class="brand-text">{{ title }}</span>
      </router-link>
      <slot name="brand" v-else>
        <span class="brand-text">{{ title }}</span>
      </slot>
    </div>

    <div class="navbar-menu">
      <slot name="menu"></slot>
    </div>

    <div class="navbar-actions">
      <slot name="actions"></slot>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useRoute } from "vue-router";
import { computed } from "vue";

interface Props {
  title?: string;
  showHomeLink?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: "Aster Agent",
  showHomeLink: true,
});

const route = useRoute();

// 默认显示首页链接，除非明确禁用或者已经在首页
const shouldShowHomeLink = computed(() => {
  if (props.showHomeLink !== undefined) {
    return props.showHomeLink;
  }
  return route.path !== "/";
});
</script>

<style scoped>
.navbar {
  @apply flex items-center justify-between px-6 py-4 bg-surface dark:bg-surface-dark border-b border-border dark:border-border-dark;
}

.navbar-brand {
  @apply flex items-center gap-3;
}

.brand-link {
  @apply hover:opacity-80 transition-opacity duration-200;
  text-decoration: none;
}

.brand-link:hover .brand-text {
  @apply text-blue-600 dark:text-blue-400;
}

.brand-text {
  @apply text-xl font-bold text-text dark:text-text-dark transition-colors duration-200;
}

.navbar-menu {
  @apply flex-1 flex items-center justify-center gap-6;
}

.navbar-actions {
  @apply flex items-center gap-3;
}
</style>
