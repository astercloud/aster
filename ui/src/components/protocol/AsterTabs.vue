<template>
  <div class="aster-tabs">
    <div class="aster-tabs-header" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="aster-tab-button"
        :class="{ 'aster-tab-active': activeTabValue === tab.id }"
        :disabled="tab.disabled"
        role="tab"
        :aria-selected="activeTabValue === tab.id"
        @click="handleTabClick(tab.id)"
      >
        <span v-if="tab.icon" class="aster-tab-icon">{{ tab.icon }}</span>
        <span class="aster-tab-label">{{ tab.label }}</span>
      </button>
    </div>
    <div class="aster-tabs-content" role="tabpanel">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { DataValue, UIActionEvent, PropertyValue } from '@/types/ui-protocol';
import { createUIActionEvent } from '@/types/ui-protocol';
import { useAsterDataBinding } from '@/composables/useAsterDataBinding';
import { useUIAction } from '@/composables/useUIAction';

/**
 * AsterTabs Component
 *
 * Tabbed interface with header buttons and content panel.
 * Uses useAsterDataBinding for reactive data binding with the data model.
 * Emits action events both as Vue events and to the Control channel.
 */

interface TabItem {
  id: string;
  label: string;
  icon?: string;
  disabled?: boolean;
}

interface Props {
  componentId?: string;
  surfaceId?: string;
  /** Active tab property - can be literal or path reference */
  activeTab?: PropertyValue | string;
  tabs?: TabItem[];
  /** Path for two-way binding (legacy support) */
  activeTabPath?: string;
}

const props = withDefaults(defineProps<Props>(), {
  tabs: () => [],
});

const emit = defineEmits<{
  action: [event: UIActionEvent];
  'update:value': [path: string, value: DataValue];
}>();

// Setup UI action emitter
const { emitFullAction } = useUIAction({
  surfaceId: props.surfaceId,
  componentId: props.componentId,
});

// Convert props to PropertyValue format for binding
function toPropertyValue(value: PropertyValue | string | undefined): PropertyValue | undefined {
  if (value === undefined) return undefined;
  if (typeof value === 'string') return { literalString: value };
  return value as PropertyValue;
}

// Use data binding for activeTab
const { value: boundActiveTab, updateValue, path: activeTabPath } = useAsterDataBinding({
  propertyValue: toPropertyValue(props.activeTab),
  defaultValue: '',
});

// Computed value for template
const activeTabValue = computed(() => String(boundActiveTab.value ?? ''));

function handleTabClick(tabId: string) {
  // Update via data binding
  updateValue(tabId);

  // Also emit for legacy support
  const path = activeTabPath ?? props.activeTabPath;
  if (path) {
    emit('update:value', path, tabId);
  }

  // Emit action event
  if (props.componentId && props.surfaceId) {
    const event = createUIActionEvent(
      props.surfaceId,
      props.componentId,
      'tab-change',
      { tabId },
    );
    emit('action', event);
    emitFullAction(event);
  }
}
</script>

<style scoped>
.aster-tabs {
  @apply flex flex-col;
}

.aster-tabs-header {
  @apply flex border-b border-gray-200 dark:border-gray-700;
}

.aster-tab-button {
  @apply px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 border-b-2 border-transparent transition-colors;
  @apply hover:text-gray-900 dark:hover:text-gray-100;
  @apply disabled:opacity-50 disabled:cursor-not-allowed;
}

.aster-tab-active {
  @apply text-blue-600 dark:text-blue-400 border-blue-600 dark:border-blue-400;
}

.aster-tab-icon {
  @apply mr-2;
}

.aster-tabs-content {
  @apply p-4;
}
</style>
