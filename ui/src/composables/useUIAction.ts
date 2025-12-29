/**
 * UI Action Composable
 *
 * Provides functionality to emit UI action events to the Control channel.
 * This composable bridges the gap between Vue component events and the
 * Aster event system, allowing user interactions to be sent to the server.
 *
 * @module useUIAction
 */

import { inject, type Ref } from 'vue';
import type { UIActionEvent } from '@/types/ui-protocol';
import { createUIActionEvent } from '@/types/ui-protocol';

/**
 * UI Action emitter function type
 */
export type UIActionEmitter = (event: UIActionEvent) => void;

/**
 * UI Action context key for dependency injection
 */
export const UI_ACTION_CONTEXT_KEY = Symbol('aster-ui-action-context');

/**
 * UI Action context interface
 */
export interface UIActionContext {
  /** Emit a UI action event to the Control channel */
  emitAction: UIActionEmitter;
  /** Whether the action emitter is connected */
  isConnected: Ref<boolean>;
}

/**
 * Options for useUIAction composable
 */
export interface UseUIActionOptions {
  /** Surface ID for the action */
  surfaceId?: string;
  /** Component ID for the action */
  componentId?: string;
}

/**
 * Return type for useUIAction composable
 */
export interface UseUIActionReturn {
  /** Emit a UI action event */
  emitAction: (action: string, payload?: Record<string, unknown>) => void;
  /** Emit a full UI action event */
  emitFullAction: UIActionEmitter;
  /** Whether the action emitter is connected */
  isConnected: Ref<boolean>;
}

/**
 * Composable for emitting UI action events to the Control channel
 *
 * @param options - Configuration options
 * @returns UI action utilities
 *
 * @example
 * ```vue
 * <script setup>
 * import { useUIAction } from '@/composables/useUIAction';
 *
 * const { emitAction } = useUIAction({
 *   surfaceId: 'my-surface',
 *   componentId: 'my-button'
 * });
 *
 * function handleClick() {
 *   emitAction('click', { timestamp: Date.now() });
 * }
 * </script>
 * ```
 */
export function useUIAction(options: UseUIActionOptions = {}): UseUIActionReturn {
  const context = inject<UIActionContext | null>(UI_ACTION_CONTEXT_KEY, null);

  // Default connected state (false if no context)
  const defaultConnected = { value: false } as Ref<boolean>;

  /**
   * Emit a UI action event with the configured surface and component IDs
   */
  function emitAction(action: string, payload?: Record<string, unknown>): void {
    if (!options.surfaceId || !options.componentId) {
      console.warn('[useUIAction] surfaceId and componentId are required to emit actions');
      return;
    }

    const event = createUIActionEvent(
      options.surfaceId,
      options.componentId,
      action,
      payload ?? {},
    );

    emitFullAction(event);
  }

  /**
   * Emit a full UI action event
   */
  function emitFullAction(event: UIActionEvent): void {
    if (context?.emitAction) {
      context.emitAction(event);
    } else {
      // Fallback: log warning if no context is available
      console.warn('[useUIAction] No UI action context available. Event not sent:', event);
    }
  }

  return {
    emitAction,
    emitFullAction,
    isConnected: context?.isConnected ?? defaultConnected,
  };
}

/**
 * Create a UI action context for providing to child components
 *
 * @param emitAction - Function to emit UI action events
 * @param isConnected - Reactive ref indicating connection status
 * @returns UI action context
 */
export function createUIActionContext(
  emitAction: UIActionEmitter,
  isConnected: Ref<boolean>,
): UIActionContext {
  return {
    emitAction,
    isConnected,
  };
}
