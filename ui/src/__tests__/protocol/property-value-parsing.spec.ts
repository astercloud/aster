/**
 * Aster UI Protocol - Property Value Parsing Unit Tests
 *
 * Tests for simplified property value format support.
 *
 * @module __tests__/protocol/property-value-parsing
 */

import { describe, expect, it } from 'vitest';
import {
  parseExtendedPropertyValue,
  normalizePropertyValue,
  isLiteralString,
  isLiteralNumber,
  isLiteralBoolean,
  isPathReference,
  getLiteralValue,
} from '@/types/ui-protocol';
import type { PropertyValue } from '@/types/ui-protocol';

describe('Property Value Parsing', () => {
  describe('parseExtendedPropertyValue', () => {
    describe('direct literal values (A2UI style)', () => {
      it('should parse string literal', () => {
        const result = parseExtendedPropertyValue('hello');

        expect(result.type).toBe('literal');
        expect(result.value).toBe('hello');
      });

      it('should parse number literal', () => {
        const result = parseExtendedPropertyValue(42);

        expect(result.type).toBe('literal');
        expect(result.value).toBe(42);
      });

      it('should parse boolean true literal', () => {
        const result = parseExtendedPropertyValue(true);

        expect(result.type).toBe('literal');
        expect(result.value).toBe(true);
      });

      it('should parse boolean false literal', () => {
        const result = parseExtendedPropertyValue(false);

        expect(result.type).toBe('literal');
        expect(result.value).toBe(false);
      });

      it('should parse zero as number', () => {
        const result = parseExtendedPropertyValue(0);

        expect(result.type).toBe('literal');
        expect(result.value).toBe(0);
      });

      it('should parse empty string', () => {
        const result = parseExtendedPropertyValue('');

        expect(result.type).toBe('literal');
        expect(result.value).toBe('');
      });

      it('should parse negative number', () => {
        const result = parseExtendedPropertyValue(-123.45);

        expect(result.type).toBe('literal');
        expect(result.value).toBe(-123.45);
      });
    });

    describe('existing format (literalString/literalNumber/literalBoolean)', () => {
      it('should parse literalString object', () => {
        const result = parseExtendedPropertyValue({ literalString: 'test' });

        expect(result.type).toBe('literal');
        expect(result.value).toBe('test');
      });

      it('should parse literalNumber object', () => {
        const result = parseExtendedPropertyValue({ literalNumber: 99 });

        expect(result.type).toBe('literal');
        expect(result.value).toBe(99);
      });

      it('should parse literalBoolean object', () => {
        const result = parseExtendedPropertyValue({ literalBoolean: true });

        expect(result.type).toBe('literal');
        expect(result.value).toBe(true);
      });
    });

    describe('path reference', () => {
      it('should parse path object', () => {
        const result = parseExtendedPropertyValue({ path: '/user/name' });

        expect(result.type).toBe('path');
        expect(result.path).toBe('/user/name');
      });

      it('should parse root path', () => {
        const result = parseExtendedPropertyValue({ path: '/' });

        expect(result.type).toBe('path');
        expect(result.path).toBe('/');
      });

      it('should parse nested path', () => {
        const result = parseExtendedPropertyValue({ path: '/a/b/c/d' });

        expect(result.type).toBe('path');
        expect(result.path).toBe('/a/b/c/d');
      });
    });

    describe('edge cases', () => {
      it('should handle null', () => {
        const result = parseExtendedPropertyValue(null);

        expect(result.type).toBe('literal');
        expect(result.value).toBeUndefined();
      });

      it('should handle undefined', () => {
        const result = parseExtendedPropertyValue(undefined);

        expect(result.type).toBe('literal');
        expect(result.value).toBeUndefined();
      });

      it('should handle empty object', () => {
        const result = parseExtendedPropertyValue({});

        expect(result.type).toBe('literal');
        expect(result.value).toBeUndefined();
      });

      it('should handle object with unknown keys', () => {
        const result = parseExtendedPropertyValue({ unknown: 'value' });

        expect(result.type).toBe('literal');
        expect(result.value).toBeUndefined();
      });
    });
  });

  describe('normalizePropertyValue', () => {
    describe('convert simple values to standard format', () => {
      it('should convert string to literalString', () => {
        const result = normalizePropertyValue('hello');

        expect(result).toEqual({ literalString: 'hello' });
      });

      it('should convert number to literalNumber', () => {
        const result = normalizePropertyValue(42);

        expect(result).toEqual({ literalNumber: 42 });
      });

      it('should convert boolean to literalBoolean', () => {
        const result = normalizePropertyValue(true);

        expect(result).toEqual({ literalBoolean: true });
      });
    });

    describe('pass through standard format', () => {
      it('should pass through literalString', () => {
        const input: PropertyValue = { literalString: 'test' };
        const result = normalizePropertyValue(input);

        expect(result).toEqual({ literalString: 'test' });
      });

      it('should pass through literalNumber', () => {
        const input: PropertyValue = { literalNumber: 123 };
        const result = normalizePropertyValue(input);

        expect(result).toEqual({ literalNumber: 123 });
      });

      it('should pass through literalBoolean', () => {
        const input: PropertyValue = { literalBoolean: false };
        const result = normalizePropertyValue(input);

        expect(result).toEqual({ literalBoolean: false });
      });

      it('should pass through path reference', () => {
        const input: PropertyValue = { path: '/data' };
        const result = normalizePropertyValue(input);

        expect(result).toEqual({ path: '/data' });
      });
    });
  });

  describe('type guard functions', () => {
    describe('isLiteralString', () => {
      it('should return true for literalString', () => {
        const value: PropertyValue = { literalString: 'test' };
        expect(isLiteralString(value)).toBe(true);
      });

      it('should return false for other types', () => {
        expect(isLiteralString({ literalNumber: 1 })).toBe(false);
        expect(isLiteralString({ literalBoolean: true })).toBe(false);
        expect(isLiteralString({ path: '/x' })).toBe(false);
      });
    });

    describe('isLiteralNumber', () => {
      it('should return true for literalNumber', () => {
        const value: PropertyValue = { literalNumber: 42 };
        expect(isLiteralNumber(value)).toBe(true);
      });

      it('should return false for other types', () => {
        expect(isLiteralNumber({ literalString: 'x' })).toBe(false);
        expect(isLiteralNumber({ literalBoolean: true })).toBe(false);
        expect(isLiteralNumber({ path: '/x' })).toBe(false);
      });
    });

    describe('isLiteralBoolean', () => {
      it('should return true for literalBoolean', () => {
        const value: PropertyValue = { literalBoolean: true };
        expect(isLiteralBoolean(value)).toBe(true);
      });

      it('should return false for other types', () => {
        expect(isLiteralBoolean({ literalString: 'x' })).toBe(false);
        expect(isLiteralBoolean({ literalNumber: 1 })).toBe(false);
        expect(isLiteralBoolean({ path: '/x' })).toBe(false);
      });
    });

    describe('isPathReference', () => {
      it('should return true for path', () => {
        const value: PropertyValue = { path: '/user' };
        expect(isPathReference(value)).toBe(true);
      });

      it('should return false for other types', () => {
        expect(isPathReference({ literalString: 'x' })).toBe(false);
        expect(isPathReference({ literalNumber: 1 })).toBe(false);
        expect(isPathReference({ literalBoolean: true })).toBe(false);
      });
    });
  });

  describe('getLiteralValue', () => {
    it('should get string value', () => {
      const value: PropertyValue = { literalString: 'hello' };
      expect(getLiteralValue(value)).toBe('hello');
    });

    it('should get number value', () => {
      const value: PropertyValue = { literalNumber: 42 };
      expect(getLiteralValue(value)).toBe(42);
    });

    it('should get boolean value', () => {
      const value: PropertyValue = { literalBoolean: true };
      expect(getLiteralValue(value)).toBe(true);
    });

    it('should return null for path reference', () => {
      const value: PropertyValue = { path: '/data' };
      expect(getLiteralValue(value)).toBeNull();
    });
  });

  describe('format equivalence', () => {
    it('string formats should be equivalent', () => {
      const simple = parseExtendedPropertyValue('test');
      const standard = parseExtendedPropertyValue({ literalString: 'test' });

      expect(simple.type).toBe(standard.type);
      expect(simple.value).toBe(standard.value);
    });

    it('number formats should be equivalent', () => {
      const simple = parseExtendedPropertyValue(123);
      const standard = parseExtendedPropertyValue({ literalNumber: 123 });

      expect(simple.type).toBe(standard.type);
      expect(simple.value).toBe(standard.value);
    });

    it('boolean formats should be equivalent', () => {
      const simple = parseExtendedPropertyValue(true);
      const standard = parseExtendedPropertyValue({ literalBoolean: true });

      expect(simple.type).toBe(standard.type);
      expect(simple.value).toBe(standard.value);
    });

    it('normalized simple values should match standard format', () => {
      const normalizedString = normalizePropertyValue('test');
      const standardString: PropertyValue = { literalString: 'test' };
      expect(normalizedString).toEqual(standardString);

      const normalizedNumber = normalizePropertyValue(42);
      const standardNumber: PropertyValue = { literalNumber: 42 };
      expect(normalizedNumber).toEqual(standardNumber);

      const normalizedBoolean = normalizePropertyValue(false);
      const standardBoolean: PropertyValue = { literalBoolean: false };
      expect(normalizedBoolean).toEqual(standardBoolean);
    });
  });
});
