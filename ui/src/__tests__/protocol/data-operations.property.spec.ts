/**
 * Aster UI Protocol - Data Model Operations Property Tests
 *
 * Property 17: Array add operation append behavior
 * Property 18: Remove operation deletion behavior
 *
 * @module __tests__/protocol/data-operations.property
 */

import { describe, expect, it } from 'vitest';
import * as fc from 'fast-check';
import { addData, removeData, setData, getData } from '@/protocol/path-resolver';
import type { DataMap, DataValue } from '@/types/ui-protocol';

// ==================
// Arbitrary Generators
// ==================

const pathSegmentArb = fc
  .string({ minLength: 1, maxLength: 20 })
  .filter((s) => /^[a-zA-Z][a-zA-Z0-9_]*$/.test(s))
  .filter((s) => !['constructor', 'prototype', '__proto__', 'toString', 'valueOf', 'hasOwnProperty'].includes(s));

const simpleValueArb: fc.Arbitrary<DataValue> = fc.oneof(
  fc.string(),
  fc.double({ noNaN: true, noDefaultInfinity: true }),
  fc.boolean(),
  fc.constant(null),
);

const arrayValueArb: fc.Arbitrary<DataValue[]> = fc.array(simpleValueArb, {
  minLength: 0,
  maxLength: 10,
});

// ==================
// Property Tests
// ==================

describe('Data Model Operations Property Tests', () => {
  describe('Property 17: Array add operation append behavior', () => {
    it('should append single value to array, preserving existing elements', () => {
      fc.assert(
        fc.property(
          arrayValueArb,
          simpleValueArb,
          (existingArray, newValue) => {
            const dataModel: DataMap = { items: [...existingArray] };
            const originalLength = existingArray.length;

            const result = addData(dataModel, '/items', newValue);

            expect(result).toBe(true);
            const items = dataModel.items as DataValue[];
            expect(items.length).toBe(originalLength + 1);
            expect(items[items.length - 1]).toEqual(newValue);

            // Verify existing elements are preserved
            for (let i = 0; i < originalLength; i++) {
              expect(items[i]).toEqual(existingArray[i]);
            }
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should append all elements when adding array to array', () => {
      fc.assert(
        fc.property(
          arrayValueArb,
          arrayValueArb,
          (existingArray, newArray) => {
            const dataModel: DataMap = { items: [...existingArray] };
            const originalLength = existingArray.length;

            const result = addData(dataModel, '/items', newArray);

            expect(result).toBe(true);
            const items = dataModel.items as DataValue[];
            expect(items.length).toBe(originalLength + newArray.length);

            // Verify existing elements
            for (let i = 0; i < originalLength; i++) {
              expect(items[i]).toEqual(existingArray[i]);
            }

            // Verify new elements
            for (let i = 0; i < newArray.length; i++) {
              expect(items[originalLength + i]).toEqual(newArray[i]);
            }
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should merge properties when adding object to object', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          simpleValueArb,
          pathSegmentArb,
          simpleValueArb,
          (key1, value1, key2, value2) => {
            // Ensure different keys
            const safeKey2 = key1 === key2 ? `${key2}2` : key2;

            const dataModel: DataMap = {
              obj: { [key1]: value1 },
            };

            const result = addData(dataModel, '/obj', { [safeKey2]: value2 });

            expect(result).toBe(true);
            const obj = dataModel.obj as DataMap;
            expect(obj[key1]).toEqual(value1);
            expect(obj[safeKey2]).toEqual(value2);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should create path if it does not exist', () => {
      fc.assert(
        fc.property(pathSegmentArb, simpleValueArb, (key, value) => {
          const dataModel: DataMap = {};

          const result = addData(dataModel, `/${key}`, value);

          expect(result).toBe(true);
          expect(dataModel[key]).toEqual(value);
        }),
        { numRuns: 100 },
      );
    });
  });

  describe('Property 18: Remove operation deletion behavior', () => {
    it('should remove key from object', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          simpleValueArb,
          pathSegmentArb,
          simpleValueArb,
          (key1, value1, key2, value2) => {
            const safeKey2 = key1 === key2 ? `${key2}2` : key2;
            const dataModel: DataMap = {
              [key1]: value1,
              [safeKey2]: value2,
            };

            const result = removeData(dataModel, `/${key1}`);

            expect(result).toBe(true);
            expect(key1 in dataModel).toBe(false);
            expect(dataModel[safeKey2]).toEqual(value2);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should remove array element and reindex', () => {
      fc.assert(
        fc.property(
          fc.array(simpleValueArb, { minLength: 2, maxLength: 10 }),
          (array) => {
            const dataModel: DataMap = { items: [...array] };
            const originalLength = array.length;
            const indexToRemove = Math.floor(originalLength / 2);

            const result = removeData(dataModel, `/items/${indexToRemove}`);

            expect(result).toBe(true);
            const items = dataModel.items as DataValue[];
            expect(items.length).toBe(originalLength - 1);

            // Verify elements before removed index
            for (let i = 0; i < indexToRemove; i++) {
              expect(items[i]).toEqual(array[i]);
            }

            // Verify elements after removed index (shifted)
            for (let i = indexToRemove; i < items.length; i++) {
              expect(items[i]).toEqual(array[i + 1]);
            }
          },
        ),
        { numRuns: 100 },
      );
    });

    it('should return false for non-existent path', () => {
      fc.assert(
        fc.property(pathSegmentArb, pathSegmentArb, (existingKey, missingKey) => {
          const safeMissingKey = existingKey === missingKey ? `${missingKey}2` : missingKey;
          const dataModel: DataMap = { [existingKey]: 'value' };

          const result = removeData(dataModel, `/${safeMissingKey}`);

          expect(result).toBe(false);
          expect(dataModel[existingKey]).toBe('value');
        }),
        { numRuns: 100 },
      );
    });

    it('should clear all keys when removing root', () => {
      fc.assert(
        fc.property(
          fc.dictionary(pathSegmentArb, simpleValueArb, { minKeys: 1, maxKeys: 5 }),
          (initialData) => {
            const dataModel: DataMap = { ...initialData };

            const result = removeData(dataModel, '/');

            expect(result).toBe(true);
            expect(Object.keys(dataModel).length).toBe(0);
          },
        ),
        { numRuns: 100 },
      );
    });
  });

  describe('Operation consistency', () => {
    it('add then remove should restore original state for objects', () => {
      fc.assert(
        fc.property(
          pathSegmentArb,
          simpleValueArb,
          pathSegmentArb,
          simpleValueArb,
          (existingKey, existingValue, newKey, newValue) => {
            const safeNewKey = existingKey === newKey ? `${newKey}2` : newKey;
            const dataModel: DataMap = { [existingKey]: existingValue };
            const originalState = JSON.stringify(dataModel);

            // Add new key
            addData(dataModel, `/${safeNewKey}`, newValue);
            expect(dataModel[safeNewKey]).toEqual(newValue);

            // Remove added key
            removeData(dataModel, `/${safeNewKey}`);

            expect(JSON.stringify(dataModel)).toBe(originalState);
          },
        ),
        { numRuns: 100 },
      );
    });

    it('set then get should return same value', () => {
      fc.assert(
        fc.property(pathSegmentArb, simpleValueArb, (key, value) => {
          const dataModel: DataMap = {};

          setData(dataModel, `/${key}`, value);
          const retrieved = getData(dataModel, `/${key}`);

          expect(retrieved).toEqual(value);
        }),
        { numRuns: 100 },
      );
    });

    it('multiple adds should accumulate in array', () => {
      fc.assert(
        fc.property(
          fc.array(simpleValueArb, { minLength: 1, maxLength: 5 }),
          (valuesToAdd) => {
            const dataModel: DataMap = { items: [] };

            for (const value of valuesToAdd) {
              addData(dataModel, '/items', value);
            }

            const items = dataModel.items as DataValue[];
            expect(items.length).toBe(valuesToAdd.length);

            for (let i = 0; i < valuesToAdd.length; i++) {
              expect(items[i]).toEqual(valuesToAdd[i]);
            }
          },
        ),
        { numRuns: 100 },
      );
    });
  });
});
