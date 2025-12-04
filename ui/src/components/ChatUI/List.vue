<template>
  <div class="list-container">
    <div v-for="(item, index) in items" :key="index" class="list-item" @click="$emit('select', item)">
      <slot :item="item" :index="index">
        {{ item }}
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  items: any[];
}

defineProps<Props>();

defineEmits<{
  select: [item: any];
}>();
</script>

<style scoped>
.list-container {
  @apply divide-y divide-gray-200 dark:divide-gray-700;
}

.list-item {
  @apply px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer transition-colors;
}
</style>
