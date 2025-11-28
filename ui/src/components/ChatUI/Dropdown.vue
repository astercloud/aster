<template>
  <div class="dropdown" ref="dropdownRef">
    <div @click="toggle">
      <slot name="trigger">
        <Button>{{ label }}</Button>
      </slot>
    </div>
    
    <Teleport to="body">
      <div
        v-if="visible"
        class="dropdown-menu"
        :style="menuStyle"
      >
        <div
          v-for="item in items"
          :key="item.key"
          class="dropdown-item"
          @click="handleSelect(item)"
        >
          <Icon v-if="item.icon" :type="(item.icon as any)" />
          <span>{{ item.label }}</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import Button from './Button.vue';
import Icon from './Icon.vue';

interface DropdownItem {
  key: string;
  label: string;
  icon?: string; // IconType compatible string
}

interface Props {
  items: DropdownItem[];
  label?: string;
}

withDefaults(defineProps<Props>(), {
  label: '选择',
});

const emit = defineEmits<{
  select: [item: DropdownItem];
}>();

const visible = ref(false);
const dropdownRef = ref<HTMLDivElement>();
const menuStyle = ref({});

const toggle = () => {
  visible.value = !visible.value;
  if (visible.value) {
    updateMenuPosition();
  }
};

const handleSelect = (item: DropdownItem) => {
  emit('select', item);
  visible.value = false;
};

const updateMenuPosition = () => {
  if (!dropdownRef.value) return;
  const rect = dropdownRef.value.getBoundingClientRect();
  menuStyle.value = {
    position: 'fixed',
    top: `${rect.bottom + 8}px`,
    left: `${rect.left}px`,
    minWidth: `${rect.width}px`,
  };
};

const handleClickOutside = (e: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    visible.value = false;
  }
};

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});
</script>

<style scoped>
.dropdown {
  @apply relative inline-block;
}

.dropdown-menu {
  @apply z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 min-w-[160px];
  animation: slideDown 0.2s;
}

.dropdown-item {
  @apply flex items-center gap-2 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
