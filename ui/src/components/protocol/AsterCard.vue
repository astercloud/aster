<template>
  <div
    class="aster-card"
    :class="{ 'aster-card-clickable': clickable }"
    @click="handleClick"
  >
    <div v-if="title || subtitle" class="aster-card-header">
      <h3 v-if="title" class="aster-card-title">{{ title }}</h3>
      <p v-if="subtitle" class="aster-card-subtitle">{{ subtitle }}</p>
    </div>
    <div class="aster-card-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { UIActionEvent } from '@/types/ui-protocol';
import { createUIActionEvent } from '@/types/ui-protocol';
import { useUIAction } from '@/composables/useUIAction';

/**
 * AsterCard Component
 *
 * Card container with optional title, subtitle, and click action.
 * Emits action events both as Vue events and to the Control channel.
 */

interface Props {
  componentId?: string;
  surfaceId?: string;
  title?: string;
  subtitle?: string;
  clickable?: boolean;
  action?: string;
}

const props = withDefaults(defineProps<Props>(), {
  clickable: false,
});

const emit = defineEmits<{
  action: [event: UIActionEvent];
}>();

// Setup UI action emitter
const { emitFullAction } = useUIAction({
  surfaceId: props.surfaceId,
  componentId: props.componentId,
});

function handleClick() {
  if (props.clickable && props.action && props.componentId && props.surfaceId) {
    const event = createUIActionEvent(
      props.surfaceId,
      props.componentId,
      props.action,
      {},
    );
    emit('action', event);
    emitFullAction(event);
  }
}
</script>

<style scoped>
.aster-card {
  @apply overflow-hidden;
  background-color: var(--aster-surface, #ffffff);
  border: 1px solid var(--aster-border, #e5e7eb);
  border-radius: var(--aster-radius-lg, 0.5rem);
  box-shadow: var(--aster-shadow-sm, 0 1px 2px 0 rgb(0 0 0 / 0.05));
}

.dark .aster-card {
  background-color: var(--aster-surface, #334155);
  border-color: var(--aster-border, #475569);
}

.aster-card-clickable {
  @apply cursor-pointer transition-shadow;
}

.aster-card-clickable:hover {
  box-shadow: var(--aster-shadow-md, 0 4px 6px -1px rgb(0 0 0 / 0.1));
}

.aster-card-header {
  @apply px-4 py-3;
  border-bottom: 1px solid var(--aster-border, #e5e7eb);
}

.dark .aster-card-header {
  border-bottom-color: var(--aster-border, #475569);
}

.aster-card-title {
  @apply text-base font-semibold m-0;
  color: var(--aster-text, #111827);
}

.dark .aster-card-title {
  color: var(--aster-text, #f1f5f9);
}

.aster-card-subtitle {
  @apply text-sm mt-1 m-0;
  color: var(--aster-text-secondary, #6b7280);
}

.dark .aster-card-subtitle {
  color: var(--aster-text-secondary, #cbd5e1);
}

.aster-card-content {
  padding: var(--aster-spacing-md, 1rem);
}
</style>
