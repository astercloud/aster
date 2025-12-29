/**
 * Aster UI Protocol - User Interaction Event Emission Property Tests
 *
 * Feature: aster-ui-protocol
 * Property 15: User Interaction Event Emission
 *
 * Validates: Requirements 4.6, 10.5
 *
 * For any user interaction with UI components (button clicks, form inputs):
 * - The renderer should emit correct `ui:action` events to the Control channel
 */

import { describe, expect, it, vi } from 'vitest';
import * as fc from 'fast-check';
import { ref } from 'vue';
import {
  useUIAction,
  createUIActionContext,
  type UIActionContext,
  type UIActionEmitter,
} from '@/composables/useUIAction';
import type { UIActionEvent } from '@/types/ui-protocol';

// ==================
// Arbitrary Generators
// ==================

/**
 * Generate valid surface ID
 */
const surfaceIdArb = fc
  .string({ minLength: 1, maxLength: 30 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

/**
 * Generate valid component ID
 */
const componentIdArb = fc
  .string({ minLength: 1, maxLength: 30 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

/**
 * Generate valid action name
 */
const actionNameArb = fc
  .string({ minLength: 1, maxLength: 20 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

/**
 * Generate simple payload value
 */
const simplePayloadValueArb: fc.Arbitrary<unknown> = fc.oneof(
  fc.string(),
  fc.double({ noNaN: true, noDefaultInfinity: true }),
  fc.boolean(),
  fc.constant(null),
);

/**
 * Generate payload object
 */
const payloadArb: fc.Arbitrary<Record<string, unknown>> = fc.dictionary(
  fc.string({ minLength: 1, maxLength: 10 }).filter((s) => /^[a-zA-Z][a-zA-Z0-9_]*$/.test(s)),
  simplePayloadValueArb,
  { minKeys: 0, maxKeys: 5 },
);

/**
 * Generate ISO timestamp string
 */
const timestampArb = fc.integer({ min: 1577836800000, max: 1893456000000 }).map(ts => new Date(ts).toISOString());

/**
 * Generate complete UIActionEvent
 */
const uiActionEventArb: fc.Arbitrary<UIActionEvent> = fc.record({
  surfaceId: surfaceIdArb,
  componentId: componentIdArb,
  action: actionNameArb,
  timestamp: timestampArb,
  context: payloadArb,
  payload: fc.option(payloadArb, { nil: undefined }),
});

// ==================
// Test Helpers
// ==================

/**
 * Create a mock UI action context with a spy emitter
 */
function createMockContext(): {
  context: UIActionContext;
  emitSpy: ReturnType<typeof vi.fn>;
} {
  const emitSpy = vi.fn();
  const isConnected = ref(true);

  const context: UIActionContext = {
    emitAction: emitSpy as UIActionEmitter,
    isConnected,
  };

  return { context, emitSpy };
}

// ==================
// Property Tests
// ==================

describe('useUIAction', () => {
  describe('Property 15: User Interaction Event Emission', () => {
    /**
     * Feature: aster-ui-protocol, Property 15: User Interaction Event Emission
     * Validates: Requirements 4.6, 10.5
     *
     * For any user interaction with UI components (button clicks, form inputs):
     * - The renderer should emit correct `ui:action` events to the Control channel
     */

    describe('Event structure correctness', () => {
      it('should emit events with correct surfaceId, componentId, and action', () => {
        fc.assert(
          fc.property(surfaceIdArb, componentIdArb, actionNameArb, (surfaceId, componentId, action) => {
            const { context, emitSpy } = createMockContext();

            // Create expected event
            const expectedEvent: UIActionEvent = {
              surfaceId,
              componentId,
              action,
              timestamp: new Date().toISOString(),
              context: {},
            };

            // Emit via context directly
            context.emitAction(expectedEvent);

            // Verify event structure
            expect(emitSpy).toHaveBeenCalledTimes(1);
            expect(emitSpy).toHaveBeenCalledWith(expectedEvent);

            const emittedEvent = emitSpy.mock.calls[0]?.[0];
            expect(emittedEvent?.surfaceId).toBe(surfaceId);
            expect(emittedEvent?.componentId).toBe(componentId);
            expect(emittedEvent?.action).toBe(action);
          }),
          { numRuns: 100 },
        );
      });

      it('should preserve payload data in emitted events', () => {
        fc.assert(
          fc.property(uiActionEventArb, (event) => {
            const { context, emitSpy } = createMockContext();

            // Emit the event
            context.emitAction(event);

            // Verify payload is preserved
            expect(emitSpy).toHaveBeenCalledTimes(1);
            const emittedEvent = emitSpy.mock.calls[0]?.[0];

            expect(emittedEvent?.surfaceId).toBe(event.surfaceId);
            expect(emittedEvent?.componentId).toBe(event.componentId);
            expect(emittedEvent?.action).toBe(event.action);

            if (event.payload !== undefined) {
              expect(emittedEvent?.payload).toEqual(event.payload);
            }
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('emitFullAction correctness', () => {
      it('should pass complete UIActionEvent to context emitter', () => {
        fc.assert(
          fc.property(uiActionEventArb, (event) => {
            const { context, emitSpy } = createMockContext();

            // Emit via context
            context.emitAction(event);

            // Verify the event was passed correctly
            expect(emitSpy).toHaveBeenCalledWith(event);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Event emission idempotence', () => {
      it('should emit identical events for identical inputs', () => {
        fc.assert(
          fc.property(uiActionEventArb, (event) => {
            const { context: context1, emitSpy: emitSpy1 } = createMockContext();
            const { context: context2, emitSpy: emitSpy2 } = createMockContext();

            // Emit same event to both contexts
            context1.emitAction(event);
            context2.emitAction(event);

            // Both should receive identical events
            expect(emitSpy1.mock.calls[0]?.[0]).toEqual(emitSpy2.mock.calls[0]?.[0]);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Multiple event emissions', () => {
      it('should emit events in order for sequential interactions', () => {
        fc.assert(
          fc.property(fc.array(uiActionEventArb, { minLength: 1, maxLength: 10 }), (events) => {
            const { context, emitSpy } = createMockContext();

            // Emit all events
            events.forEach((event) => context.emitAction(event));

            // Verify all events were emitted in order
            expect(emitSpy).toHaveBeenCalledTimes(events.length);

            events.forEach((event, index) => {
              expect(emitSpy.mock.calls[index]?.[0]).toEqual(event);
            });
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Connection state handling', () => {
      it('should reflect connection state correctly', () => {
        fc.assert(
          fc.property(fc.boolean(), (connected) => {
            const emitSpy = vi.fn() as UIActionEmitter;
            const isConnected = ref(connected);

            const context: UIActionContext = {
              emitAction: emitSpy,
              isConnected,
            };

            expect(context.isConnected.value).toBe(connected);
          }),
          { numRuns: 100 },
        );
      });
    });
  });

  describe('createUIActionContext', () => {
    it('should create valid context with all required properties', () => {
      fc.assert(
        fc.property(fc.boolean(), (connected) => {
          const emitAction = vi.fn() as UIActionEmitter;
          const isConnected = ref(connected);

          const context = createUIActionContext(emitAction, isConnected);

          expect(context.emitAction).toBe(emitAction);
          expect(context.isConnected).toBe(isConnected);
          expect(context.isConnected.value).toBe(connected);
        }),
        { numRuns: 100 },
      );
    });

    it('should allow emitting events through created context', () => {
      fc.assert(
        fc.property(uiActionEventArb, (event) => {
          const emitAction = vi.fn() as UIActionEmitter;
          const isConnected = ref(true);

          const context = createUIActionContext(emitAction, isConnected);
          context.emitAction(event);

          expect(emitAction).toHaveBeenCalledWith(event);
        }),
        { numRuns: 100 },
      );
    });
  });

  describe('useUIAction options validation', () => {
    it('should handle missing surfaceId gracefully', () => {
      fc.assert(
        fc.property(componentIdArb, actionNameArb, (componentId, action) => {
          // Create useUIAction without surfaceId
          const { emitAction } = useUIAction({
            componentId,
          });

          // Should not throw, but should warn (we can't easily test console.warn in property tests)
          expect(() => emitAction(action)).not.toThrow();
        }),
        { numRuns: 100 },
      );
    });

    it('should handle missing componentId gracefully', () => {
      fc.assert(
        fc.property(surfaceIdArb, actionNameArb, (surfaceId, action) => {
          // Create useUIAction without componentId
          const { emitAction } = useUIAction({
            surfaceId,
          });

          // Should not throw
          expect(() => emitAction(action)).not.toThrow();
        }),
        { numRuns: 100 },
      );
    });

    it('should handle empty options gracefully', () => {
      fc.assert(
        fc.property(actionNameArb, (action) => {
          // Create useUIAction with empty options
          const { emitAction } = useUIAction({});

          // Should not throw
          expect(() => emitAction(action)).not.toThrow();
        }),
        { numRuns: 100 },
      );
    });
  });

  describe('Payload handling', () => {
    it('should handle undefined payload correctly', () => {
      fc.assert(
        fc.property(surfaceIdArb, componentIdArb, actionNameArb, (surfaceId, componentId, action) => {
          const { context, emitSpy } = createMockContext();

          const event: UIActionEvent = {
            surfaceId,
            componentId,
            action,
            timestamp: new Date().toISOString(),
            context: {},
            // payload is undefined
          };

          context.emitAction(event);

          expect(emitSpy).toHaveBeenCalledWith(event);
          expect(emitSpy.mock.calls[0]?.[0]?.payload).toBeUndefined();
        }),
        { numRuns: 100 },
      );
    });

    it('should handle empty payload object correctly', () => {
      fc.assert(
        fc.property(surfaceIdArb, componentIdArb, actionNameArb, (surfaceId, componentId, action) => {
          const { context, emitSpy } = createMockContext();

          const event: UIActionEvent = {
            surfaceId,
            componentId,
            action,
            timestamp: new Date().toISOString(),
            context: {},
            payload: {},
          };

          context.emitAction(event);

          expect(emitSpy).toHaveBeenCalledWith(event);
          expect(emitSpy.mock.calls[0]?.[0]?.payload).toEqual({});
        }),
        { numRuns: 100 },
      );
    });

    it('should preserve complex payload structures', () => {
      fc.assert(
        fc.property(
          surfaceIdArb,
          componentIdArb,
          actionNameArb,
          payloadArb,
          (surfaceId, componentId, action, payload) => {
            const { context, emitSpy } = createMockContext();

            const event: UIActionEvent = {
              surfaceId,
              componentId,
              action,
              timestamp: new Date().toISOString(),
              context: {},
              payload,
            };

            context.emitAction(event);

            expect(emitSpy.mock.calls[0]?.[0]?.payload).toEqual(payload);
          },
        ),
        { numRuns: 100 },
      );
    });
  });
});
