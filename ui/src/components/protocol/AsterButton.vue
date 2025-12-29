<template>
  <button
    class="aster-button"
    :class="[variantClass, { 'aster-button-disabled': disabled }]"
    :disabled="disabled"
    @click="handleClick"
  >
    <span v-if="icon" class="aster-button-icon">{{ icon }}</span>
    <span class="aster-button-label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue';
import type { ButtonVariant, UIActionEvent, PropertyValue, DataMap } from '@/types/ui-protocol';
import { createUIActionEvent } from '@/types/ui-protocol';
import { useUIAction } from '@/composables/useUIAction';
import { resolveActionContext } from '@/protocol/client-message';

/**
 * AsterButton Component
 *
 * Button with variants and action emission.
 * Emits action events both as Vue events and to the Control channel.
 * Supports actionContext for automatic form data collection.
 */

interface Props {
  componentId?: string;
  surfaceId?: string;
  label?: string;
  action?: string;
  variant?: ButtonVariant;
  disabled?: boolean;
  icon?: string;
  actionContext?: Record<string, PropertyValue>;
}

const props = withDefaults(defineProps<Props>(), {
  label: '',
  action: 'click',
  variant: 'primary',
  disabled: false,
});

const emit = defineEmits<{
  action: [event: UIActionEvent];
}>();

// Inject data model from parent surface (if available)
const dataModel = inject<DataMap>('surfaceDataModel', {});

// Setup UI action emitter
const { emitFullAction } = useUIAction({
  surfaceId: props.surfaceId,
  componentId: props.componentId,
});

const variantClass = computed(() => {
  const variantMap: Record<ButtonVariant, string> = {
    primary: 'aster-button-primary',
    secondary: 'aster-button-secondary',
    text: 'aster-button-text',
  };
  return variantMap[props.variant] ?? 'aster-button-primary';
});

function handleClick() {
  if (!props.disabled && props.componentId && props.surfaceId && props.action) {
    // Resolve action context if provided
    const context = props.actionContext
      ? resolveActionContext(props.actionContext, dataModel)
      : {};

    const event = createUIActionEvent(
      props.surfaceId,
      props.componentId,
      props.action,
      context,
    );

    emit('action', event);
    emitFullAction(event);
  }
}
</script>

<style scoped>
.aster-button {
  @apply inline-flex items-center justify-center text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2;
  padding: var(--aster-spacing-sm, 0.5rem) var(--aster-spacing-md, 1rem);
  border-radius: var(--aster-radius-lg, 0.5rem);
}

.aster-button-primary {
  background-color: var(--aster-primary, #3b82f6);
  color: var(--aster-primary-contrast, #ffffff);
}

.aster-button-primary:hover:not(:disabled) {
  background-color: var(--aster-primary-hover, #2563eb);
}

.aster-button-primary:focus {
  --tw-ring-color: var(--aster-primary, #3b82f6);
}

.aster-button-secondary {
  background-color: var(--aster-secondary-light, #f3f4f6);
  color: var(--aster-text, #111827);
}

.aster-button-secondary:hover:not(:disabled) {
  background-color: var(--aster-border, #e5e7eb);
}

.aster-button-secondary:focus {
  --tw-ring-color: var(--aster-secondary, #6b7280);
}

.dark .aster-button-secondary {
  background-color: var(--aster-surface, #334155);
  color: var(--aster-text, #f1f5f9);
}

.dark .aster-button-secondary:hover:not(:disabled) {
  background-color: var(--aster-surface-hover, #475569);
}

.aster-button-text {
  background-color: transparent;
  color: var(--aster-primary, #3b82f6);
}

.aster-button-text:hover:not(:disabled) {
  background-color: var(--aster-primary-light, #dbeafe);
}

.aster-button-text:focus {
  --tw-ring-color: var(--aster-primary, #3b82f6);
}

.dark .aster-button-text {
  color: var(--aster-primary, #60a5fa);
}

.dark .aster-button-text:hover:not(:disabled) {
  background-color: var(--aster-surface, #334155);
}

.aster-button-disabled {
  @apply opacity-50 cursor-not-allowed;
}

.aster-button-icon {
  @apply mr-2;
}
</style>
