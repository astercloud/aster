/**
 * Aster UI Protocol - Property Value Property Tests
 *
 * Property 20: Simplified property value backward compatibility
 *
 * @module __tests__/protocol/property-value.property
 */

import { describe, expect, it } from 'vitest';
import * as fc from 'fast-check';
import {
  parseExtendedPropertyValue,
  normalizePropertyValue,
  isLiteralString,
  isLiteralNumber,
  isLiteralBoolean,
  isPathReference,
} from '@/types/ui-protocol';
import type { PropertyValue } from '@/types/ui-protocol';

// ==================
// Arbitrary Generators
// ==================

const stringValueArb = fc.string();
const numberValueArb = fc.double({ noNaN: true, noDefaultInfinity: true });
const booleanValueArb = fc.boolean();

const pathArb = fc
  .array(
    fc.string({ minLength: 1, maxLength: 20 }).filter((s) => /^[a-zA-Z][a-zA-Z0-9_]*$/.test(s)),
    { minLength: 1, maxLength: 5 },
  )
  .map((segments) => '/' + segments.join('/'));

const standardPropertyValueArb: fc.Arbitrary<PropertyValue> = fc.oneof(
  stringValueArb.map((s) => ({ literalString: s })),
  numberValueArb.map((n) => ({ literalNumber: n })),
  booleanValueArb.map((b) => ({ literalBoolean: b })),
  pathArb.map((p) => ({ path: p })),
);

// ==================
// Property Tests
// ==================

describe('Property Value Property Tests', () => {
  describe('Property 20: Simplified property value backward compatibility', () => {
    describe('String format equivalence', () => {
      it('simple string should parse equivalently to literalString', () => {
        fc.assert(
          fc.property(stringValueArb, (str) => {
            const simpleResult = parseExtendedPropertyValue(str);
            const standardResult = parseExtendedPropertyValue({ literalString: str });

            expect(simpleResult.type).toBe(standardResult.type);
            expect(simpleResult.value).toBe(standardResult.value);
          }),
          { numRuns: 100 },
        );
      });

      it('normalized simple string should equal standard format', () => {
        fc.assert(
          fc.property(stringValueArb, (str) => {
            const normalized = normalizePropertyValue(str);
            const standard: PropertyValue = { literalString: str };

            expect(normalized).toEqual(standard);
            expect(isLiteralString(normalized)).toBe(true);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Number format equivalence', () => {
      it('simple number should parse equivalently to literalNumber', () => {
        fc.assert(
          fc.property(numberValueArb, (num) => {
            const simpleResult = parseExtendedPropertyValue(num);
            const standardResult = parseExtendedPropertyValue({ literalNumber: num });

            expect(simpleResult.type).toBe(standardResult.type);
            expect(simpleResult.value).toBe(standardResult.value);
          }),
          { numRuns: 100 },
        );
      });

      it('normalized simple number should equal standard format', () => {
        fc.assert(
          fc.property(numberValueArb, (num) => {
            const normalized = normalizePropertyValue(num);
            const standard: PropertyValue = { literalNumber: num };

            expect(normalized).toEqual(standard);
            expect(isLiteralNumber(normalized)).toBe(true);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Boolean format equivalence', () => {
      it('simple boolean should parse equivalently to literalBoolean', () => {
        fc.assert(
          fc.property(booleanValueArb, (bool) => {
            const simpleResult = parseExtendedPropertyValue(bool);
            const standardResult = parseExtendedPropertyValue({ literalBoolean: bool });

            expect(simpleResult.type).toBe(standardResult.type);
            expect(simpleResult.value).toBe(standardResult.value);
          }),
          { numRuns: 100 },
        );
      });

      it('normalized simple boolean should equal standard format', () => {
        fc.assert(
          fc.property(booleanValueArb, (bool) => {
            const normalized = normalizePropertyValue(bool);
            const standard: PropertyValue = { literalBoolean: bool };

            expect(normalized).toEqual(standard);
            expect(isLiteralBoolean(normalized)).toBe(true);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Path reference handling', () => {
      it('path object should parse correctly', () => {
        fc.assert(
          fc.property(pathArb, (path) => {
            const result = parseExtendedPropertyValue({ path });

            expect(result.type).toBe('path');
            expect(result.path).toBe(path);
          }),
          { numRuns: 100 },
        );
      });

      it('normalized path should preserve path reference', () => {
        fc.assert(
          fc.property(pathArb, (path) => {
            const input: PropertyValue = { path };
            const normalized = normalizePropertyValue(input);

            expect(normalized).toEqual({ path });
            expect(isPathReference(normalized)).toBe(true);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Standard format passthrough', () => {
      it('standard PropertyValue should normalize to itself', () => {
        fc.assert(
          fc.property(standardPropertyValueArb, (pv) => {
            const normalized = normalizePropertyValue(pv);

            expect(normalized).toEqual(pv);
          }),
          { numRuns: 100 },
        );
      });

      it('standard PropertyValue should parse correctly', () => {
        fc.assert(
          fc.property(standardPropertyValueArb, (pv) => {
            const result = parseExtendedPropertyValue(pv);

            if ('literalString' in pv) {
              expect(result.type).toBe('literal');
              expect(result.value).toBe(pv.literalString);
            } else if ('literalNumber' in pv) {
              expect(result.type).toBe('literal');
              expect(result.value).toBe(pv.literalNumber);
            } else if ('literalBoolean' in pv) {
              expect(result.type).toBe('literal');
              expect(result.value).toBe(pv.literalBoolean);
            } else if ('path' in pv) {
              expect(result.type).toBe('path');
              expect(result.path).toBe(pv.path);
            }
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Type guard consistency', () => {
      it('exactly one type guard should return true for any PropertyValue', () => {
        fc.assert(
          fc.property(standardPropertyValueArb, (pv) => {
            const guards = [
              isLiteralString(pv),
              isLiteralNumber(pv),
              isLiteralBoolean(pv),
              isPathReference(pv),
            ];

            const trueCount = guards.filter(Boolean).length;
            expect(trueCount).toBe(1);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Round-trip consistency', () => {
      it('normalize then parse should preserve value', () => {
        fc.assert(
          fc.property(stringValueArb, (str) => {
            const normalized = normalizePropertyValue(str);
            const parsed = parseExtendedPropertyValue(normalized);

            expect(parsed.type).toBe('literal');
            expect(parsed.value).toBe(str);
          }),
          { numRuns: 100 },
        );
      });

      it('parse then normalize should be idempotent for standard format', () => {
        fc.assert(
          fc.property(standardPropertyValueArb, (pv) => {
            const normalized1 = normalizePropertyValue(pv);
            const normalized2 = normalizePropertyValue(normalized1);

            expect(normalized1).toEqual(normalized2);
          }),
          { numRuns: 100 },
        );
      });
    });

    describe('Edge cases', () => {
      it('should handle empty string', () => {
        const result = parseExtendedPropertyValue('');
        expect(result.type).toBe('literal');
        expect(result.value).toBe('');

        const normalized = normalizePropertyValue('');
        expect(normalized).toEqual({ literalString: '' });
      });

      it('should handle zero', () => {
        const result = parseExtendedPropertyValue(0);
        expect(result.type).toBe('literal');
        expect(result.value).toBe(0);

        const normalized = normalizePropertyValue(0);
        expect(normalized).toEqual({ literalNumber: 0 });
      });

      it('should handle false', () => {
        const result = parseExtendedPropertyValue(false);
        expect(result.type).toBe('literal');
        expect(result.value).toBe(false);

        const normalized = normalizePropertyValue(false);
        expect(normalized).toEqual({ literalBoolean: false });
      });
    });
  });
});
