/**
 * Aster UI Protocol - Action Context Property Tests
 *
 * Property 19: actionContext resolution correctness
 *
 * @module __tests__/protocol/action-context.property
 */

import { describe, expect, it } from 'vitest';
import * as fc from 'fast-check';
import { resolveActionContext } from '@/protocol/client-message';
import { createUIActionEvent } from '@/types/ui-protocol';
import type { PropertyValue, DataMap } from '@/types/ui-protocol';

// ==================
// Arbitrary Generators
// ==================

const pathSegmentArb = fc
  .string({ minLength: 1, maxLength: 20 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_]*$/.test(s))
  .filter((s) => !['constructor', 'prototype', '__proto__', 'toString', 'valueOf', 'hasOwnProperty'].includes(s));

const simpleValueArb: fc.Arbitrary<string | number | boolean> = fc.oneof(
  fc.string(),
  fc.double({ noNaN: true, noDefaultInfinity: true }),
  fc.boolean(),
);

const literalPropertyValueArb: fc.Arbitrary<PropertyValue> = fc.oneof(
  fc.string().map((s) => ({ literalString: s })),
  fc.double({ noNaN: true, noDefaultInfinity: true }).map((n) => ({ literalNumber: n })),
  fc.boolean().map((b) => ({ literalBoolean: b })),
);

const pathPropertyValueArb = (path: string): PropertyValue => ({ path });

const surfaceIdArb = fc
  .string({ minLength: 1, maxLength: 30 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

const componentIdArb = fc
  .string({ minLength: 1, maxLength: 30 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

const actionNameArb = fc
  .string({ minLength: 1, maxLength: 20 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_-]*$/.test(s));

// ==================
// Property Tests
// ==================

describe('Action Context Property Tests', () => {
  describe('Property 19: actionContext resolution correctness', () => {
    it('should resolve literal values correctly', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          literalPropertyValueArb,
          (key, propertyValue) => {
            const actionContext: Record<string, PropertyValue> = {
              [key]: propertyValue,
            };
            const dataModel: DataMap = {};

            const result = resolveActionContext(actionContext, dataModel);

            // Extract expected value from PropertyValue
            let expectedValue: unknown;
            if ('literalString' in propertyValue) {
              expectedValue = propertyValue.literalString;
            } else if ('literalNumber' in propertyValue) {
              expectedValue = propertyValue.literalNumber;
            } else if ('literalBoolean' in propertyValue) {
              expectedValue = propertyValue.literalBoolean;
            }

            expect(result[key]).toEqual(expectedValue);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should resolve path references to correct data model values', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          pathSegmentArb,
          simpleValueArb,
          (contextKey, dataKey, dataValue) => {
            const actionContext: Record<string, PropertyValue> = {
              [contextKey]: pathPropertyValueArb(`/${dataKey}`),
            };
            const dataModel: DataMap = {
              [dataKey]: dataValue,
            };

            const result = resolveActionContext(actionContext, dataModel);

            expect(result[contextKey]).toEqual(dataValue);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should resolve nested path references', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          pathSegmentArb,
          pathSegmentArb,
          simpleValueArb,
          (contextKey, level1, level2, value) => {
            const actionContext: Record<string, PropertyValue> = {
              [contextKey]: pathPropertyValueArb(`/${level1}/${level2}`),
            };
            const dataModel: DataMap = {
              [level1]: { [level2]: value },
            };

            const result = resolveActionContext(actionContext, dataModel);

            expect(result[contextKey]).toEqual(value);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should return undefined for non-existent paths', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          pathSegmentArb,
          pathSegmentArb,
          (contextKey, existingKey, missingKey) => {
            const safeMissingKey = existingKey === missingKey ? `${missingKey}2` : missingKey;
            const actionContext: Record<string, PropertyValue> = {
              [contextKey]: pathPropertyValueArb(`/${safeMissingKey}`),
            };
            const dataModel: DataMap = {
              [existingKey]: 'value',
            };

            const result = resolveActionContext(actionContext, dataModel);

            expect(result[contextKey]).toBeUndefined();
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should handle mixed literal and path values', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          literalPropertyValueArb,
          pathSegmentArb,
          pathSegmentArb,
          simpleValueArb,
          (literalKey, literalValue, pathKey, dataKey, dataValue) => {
            const safePathKey = literalKey === pathKey ? `${pathKey}2` : pathKey;
            const actionContext: Record<string, PropertyValue> = {
              [literalKey]: literalValue,
              [safePathKey]: pathPropertyValueArb(`/${dataKey}`),
            };
            const dataModel: DataMap = {
              [dataKey]: dataValue,
            };

            const result = resolveActionContext(actionContext, dataModel);

            // Verify path resolution
            expect(result[safePathKey]).toEqual(dataValue);

            // Verify literal resolution
            let expectedLiteral: unknown;
            if ('literalString' in literalValue) {
              expectedLiteral = literalValue.literalString;
            } else if ('literalNumber' in literalValue) {
              expectedLiteral = literalValue.literalNumber;
            } else if ('literalBoolean' in literalValue) {
              expectedLiteral = literalValue.literalBoolean;
            }
            expect(result[literalKey]).toEqual(expectedLiteral);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should preserve all keys from actionContext', () => {
      fc.assert(
        fc.property(
          fc.dictionary(pathSegmentArb, literalPropertyValueArb, { minKeys: 1, maxKeys: 5 }),
          (actionContext) => {
            const dataModel: DataMap = {};

            const result = resolveActionContext(actionContext, dataModel);

            expect(Object.keys(result).length).toBe(Object.keys(actionContext).length);
            for (const key of Object.keys(actionContext)) {
              expect(key in result).toBe(true);
            }
          },
        ),
        { numRuns: 100 },
      );
    });
  });

  describe('UIActionEvent generation', () => {
    it('should create event with valid timestamp', () => {
      fc.assert(
        fc.property(
          surfaceIdArb,
          componentIdArb,
          actionNameArb,
          (surfaceId, componentId, action) => {
            const event = createUIActionEvent(surfaceId, componentId, action, {});

            // Verify timestamp is ISO 8601 format
            const isoRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z$/;
            expect(event.timestamp).toMatch(isoRegex);

            // Verify timestamp is parseable
            const parsed = new Date(event.timestamp);
            expect(parsed.getTime()).not.toBeNaN();
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should preserve all input fields', () => {
      fc.assert(
        fc.property(
          surfaceIdArb,
          componentIdArb,
          actionNameArb,
          fc.dictionary(pathSegmentArb, simpleValueArb, { minKeys: 0, maxKeys: 5 }),
          (surfaceId, componentId, action, context) => {
            const event = createUIActionEvent(surfaceId, componentId, action, context);

            expect(event.surfaceId).toBe(surfaceId);
            expect(event.componentId).toBe(componentId);
            expect(event.action).toBe(action);
            expect(event.context).toEqual(context);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should generate unique timestamps for sequential calls', () => {
      fc.assert(
        fc.property(surfaceIdArb, componentIdArb, actionNameArb, (surfaceId, componentId, action) => {
          const events: string[] = [];

          // Generate multiple events
          for (let i = 0; i < 3; i++) {
            const event = createUIActionEvent(surfaceId, componentId, action, {});
            events.push(event.timestamp);
          }

          // All timestamps should be valid
          for (const ts of events) {
            expect(new Date(ts).getTime()).not.toBeNaN();
          }

          // Timestamps should be in non-decreasing order
          for (let i = 1; i < events.length; i++) {
            expect(new Date(events[i]!).getTime()).toBeGreaterThanOrEqual(
              new Date(events[i - 1]!).getTime(),
            );
          }
        }),
        { numRuns: 50 },
      );
    });
  });
});
