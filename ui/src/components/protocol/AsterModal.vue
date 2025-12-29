<template>
  <Teleport to="body">
    <Transition name="aster-modal">
      <div
        v-if="openValue"
        class="aster-modal-overlay"
        @click.self="handleOverlayClick"
      >
        <div class="aster-modal" role="dialog" aria-modal="true">
          <div v-if="titleValue || closable" class="aster-modal-header">
            <h2 v-if="titleValue" class="aster-modal-title">{{ titleValue }}</h2>
            <button
              v-if="closable"
              class="aster-modal-close"
              aria-label="Close"
              @click="handleClose"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="aster-modal-content">
            <slot />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { DataValue, UIActionEvent, PropertyValue } from '@/types/ui-protocol';
import { createUIActionEvent } from '@/types/ui-protocol';
import { useAsterDataBinding } from '@/composables/useAsterDataBinding';
import { useUIAction } from '@/composables/useUIAction';

/**
 * AsterModal Component
 *
 * Modal dialog with overlay, title, and close functionality.
 * Uses useAsterDataBinding for reactive data binding with the data model.
 * Emits action events both as Vue events and to the Control channel.
 */

interface Props {
  componentId?: string;
  surfaceId?: string;
  /** Open property - can be literal or path reference */
  open?: PropertyValue | boolean;
  /** Title property - can be literal or path reference */
  title?: PropertyValue | string;
  closable?: boolean;
  closeAction?: string;
  /** Path for two-way binding (legacy support) */
  openPath?: string;
}

const props = withDefaults(defineProps<Props>(), {
  closable: true,
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
function toPropertyValue(value: PropertyValue | string | boolean | undefined): PropertyValue | undefined {
  if (value === undefined) return undefined;
  if (typeof value === 'string') return { literalString: value };
  if (typeof value === 'boolean') return { literalBoolean: value };
  return value as PropertyValue;
}

// Use data binding for open
const { value: boundOpen, updateValue, path: openPath } = useAsterDataBinding({
  propertyValue: toPropertyValue(props.open),
  defaultValue: false,
});

// Use data binding for title
const { value: boundTitle } = useAsterDataBinding({
  propertyValue: toPropertyValue(props.title),
  defaultValue: '',
});

// Computed values for template
const openValue = computed(() => Boolean(boundOpen.value));
const titleValue = computed(() => boundTitle.value ? String(boundTitle.value) : undefined);

function handleClose() {
  // Update via data binding to close the modal
  updateValue(false);

  // Also emit for legacy support
  const path = openPath ?? props.openPath;
  if (path) {
    emit('update:value', path, false);
  }

  // Emit action event
  if (props.componentId && props.surfaceId) {
    const action = props.closeAction ?? 'close';
    const event = createUIActionEvent(
      props.surfaceId,
      props.componentId,
      action,
      {},
    );
    emit('action', event);
    emitFullAction(event);
  }
}

function handleOverlayClick() {
  if (props.closable) {
    handleClose();
  }
}
</script>

<style scoped>
.aster-modal-overlay {
  @apply fixed inset-0 flex items-center justify-center;
  z-index: var(--aster-z-modal-backdrop, 1040);
  background-color: var(--aster-modal-backdrop, rgba(0, 0, 0, 0.5));
}

.aster-modal {
  @apply max-w-lg w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col;
  background-color: var(--aster-surface, #ffffff);
  border-radius: var(--aster-radius-lg, 0.5rem);
  box-shadow: var(--aster-shadow-xl, 0 20px 25px -5px rgb(0 0 0 / 0.1));
}

.dark .aster-modal {
  background-color: var(--aster-surface, #334155);
}

.aster-modal-header {
  @apply flex items-center justify-between px-4 py-3;
  border-bottom: 1px solid var(--aster-border, #e5e7eb);
}

.dark .aster-modal-header {
  border-bottom-color: var(--aster-border, #475569);
}

.aster-modal-title {
  @apply text-lg font-semibold m-0;
  color: var(--aster-text, #111827);
}

.dark .aster-modal-title {
  color: var(--aster-text, #f1f5f9);
}

.aster-modal-close {
  @apply p-1 rounded transition-colors;
  color: var(--aster-text-secondary, #6b7280);
}

.aster-modal-close:hover {
  color: var(--aster-text, #111827);
}

.dark .aster-modal-close {
  color: var(--aster-text-secondary, #cbd5e1);
}

.dark .aster-modal-close:hover {
  color: var(--aster-text, #f1f5f9);
}

.aster-modal-content {
  @apply overflow-y-auto;
  padding: var(--aster-spacing-md, 1rem);
}

/* Transition styles */
.aster-modal-enter-active,
.aster-modal-leave-active {
  transition-duration: var(--aster-transition-normal, 200ms);
  transition-timing-function: var(--aster-transition-easing, cubic-bezier(0.4, 0, 0.2, 1));
  transition-property: opacity;
}

.aster-modal-enter-from,
.aster-modal-leave-to {
  @apply opacity-0;
}
</style>
