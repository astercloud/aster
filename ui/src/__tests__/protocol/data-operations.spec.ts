/**
 * Aster UI Protocol - Data Model Operations Unit Tests
 *
 * Tests for add/replace/remove operations on data model.
 *
 * @module __tests__/protocol/data-operations
 */

import { describe, expect, it, beforeEach } from 'vitest';
import { addData, removeData, setData, getData } from '@/protocol/path-resolver';
import type { DataMap, DataValue } from '@/types/ui-protocol';

describe('Data Model Operations', () => {
  let dataModel: DataMap;

  beforeEach(() => {
    dataModel = {};
  });

  describe('add operation', () => {
    describe('array append', () => {
      it('should append single value to existing array', () => {
        dataModel = { items: [1, 2, 3] };
        const result = addData(dataModel, '/items', 4);
        expect(result).toBe(true);
        expect(dataModel.items).toEqual([1, 2, 3, 4]);
      });

      it('should append multiple values when adding array to array', () => {
        dataModel = { items: ['a', 'b'] };
        const result = addData(dataModel, '/items', ['c', 'd']);
        expect(result).toBe(true);
        expect(dataModel.items).toEqual(['a', 'b', 'c', 'd']);
      });

      it('should append object to array', () => {
        dataModel = { users: [{ id: 1 }] };
        const result = addData(dataModel, '/users', { id: 2 });
        expect(result).toBe(true);
        expect(dataModel.users).toEqual([{ id: 1 }, { id: 2 }]);
      });

      it('should handle empty array', () => {
        dataModel = { items: [] };
        const result = addData(dataModel, '/items', 'first');
        expect(result).toBe(true);
        expect(dataModel.items).toEqual(['first']);
      });
    });

    describe('object merge', () => {
      it('should merge properties into existing object', () => {
        dataModel = { user: { name: 'Alice' } };
        const result = addData(dataModel, '/user', { age: 30 });
        expect(result).toBe(true);
        expect(dataModel.user).toEqual({ name: 'Alice', age: 30 });
      });

      it('should overwrite existing properties when merging', () => {
        dataModel = { config: { theme: 'light', lang: 'en' } };
        const result = addData(dataModel, '/config', { theme: 'dark' });
        expect(result).toBe(true);
        expect(dataModel.config).toEqual({ theme: 'dark', lang: 'en' });
      });

      it('should merge to root object', () => {
        dataModel = { existing: 'value' };
        const result = addData(dataModel, '', { newKey: 'newValue' });
        expect(result).toBe(true);
        expect(dataModel).toEqual({ existing: 'value', newKey: 'newValue' });
      });
    });

    describe('path creation', () => {
      it('should create path and set value if path does not exist', () => {
        dataModel = {};
        const result = addData(dataModel, '/newPath', 'value');
        expect(result).toBe(true);
        expect(dataModel.newPath).toBe('value');
      });

      it('should create nested path', () => {
        dataModel = { level1: {} };
        const result = addData(dataModel, '/level1/level2', { data: 'test' });
        expect(result).toBe(true);
        expect((dataModel.level1 as DataMap).level2).toEqual({ data: 'test' });
      });
    });

    describe('edge cases', () => {
      it('should return false for invalid path', () => {
        const result = addData(dataModel, 'invalid', 'value');
        expect(result).toBe(false);
      });

      it('should handle null value', () => {
        dataModel = { items: [1] };
        const result = addData(dataModel, '/items', null);
        expect(result).toBe(true);
        expect(dataModel.items).toEqual([1, null]);
      });

      it('should replace primitive value at path', () => {
        dataModel = { count: 5 };
        const result = addData(dataModel, '/count', 10);
        expect(result).toBe(true);
        expect(dataModel.count).toBe(10);
      });
    });
  });

  describe('replace operation (setData)', () => {
    describe('value replacement', () => {
      it('should replace existing value', () => {
        dataModel = { name: 'old' };
        const result = setData(dataModel, '/name', 'new');
        expect(result).toBe(true);
        expect(dataModel.name).toBe('new');
      });

      it('should replace object with primitive', () => {
        dataModel = { data: { nested: 'value' } };
        const result = setData(dataModel, '/data', 'simple');
        expect(result).toBe(true);
        expect(dataModel.data).toBe('simple');
      });

      it('should replace primitive with object', () => {
        dataModel = { value: 42 };
        const result = setData(dataModel, '/value', { complex: true });
        expect(result).toBe(true);
        expect(dataModel.value).toEqual({ complex: true });
      });

      it('should replace array element', () => {
        dataModel = { items: ['a', 'b', 'c'] };
        const result = setData(dataModel, '/items/1', 'B');
        expect(result).toBe(true);
        expect(dataModel.items).toEqual(['a', 'B', 'c']);
      });
    });

    describe('path creation', () => {
      it('should create path if it does not exist', () => {
        dataModel = {};
        const result = setData(dataModel, '/newKey', 'value');
        expect(result).toBe(true);
        expect(dataModel.newKey).toBe('value');
      });

      it('should create intermediate objects', () => {
        dataModel = {};
        const result = setData(dataModel, '/a/b', 'deep');
        expect(result).toBe(true);
        expect((dataModel.a as DataMap).b).toBe('deep');
      });
    });

    describe('edge cases', () => {
      it('should return false for empty path', () => {
        const result = setData(dataModel, '', 'value');
        expect(result).toBe(false);
      });

      it('should return false for invalid path', () => {
        const result = setData(dataModel, 'noSlash', 'value');
        expect(result).toBe(false);
      });

      it('should handle boolean values', () => {
        dataModel = { flag: false };
        const result = setData(dataModel, '/flag', true);
        expect(result).toBe(true);
        expect(dataModel.flag).toBe(true);
      });

      it('should handle number values', () => {
        dataModel = { count: 0 };
        const result = setData(dataModel, '/count', 100);
        expect(result).toBe(true);
        expect(dataModel.count).toBe(100);
      });
    });
  });

  describe('remove operation', () => {
    describe('value deletion', () => {
      it('should remove existing key', () => {
        dataModel = { a: 1, b: 2 };
        const result = removeData(dataModel, '/a');
        expect(result).toBe(true);
        expect(dataModel).toEqual({ b: 2 });
        expect('a' in dataModel).toBe(false);
      });

      it('should remove nested key', () => {
        dataModel = { outer: { inner: 'value', keep: 'this' } };
        const result = removeData(dataModel, '/outer/inner');
        expect(result).toBe(true);
        expect(dataModel.outer).toEqual({ keep: 'this' });
      });
    });

    describe('array element deletion', () => {
      it('should remove array element and reindex', () => {
        dataModel = { items: ['a', 'b', 'c'] };
        const result = removeData(dataModel, '/items/1');
        expect(result).toBe(true);
        expect(dataModel.items).toEqual(['a', 'c']);
      });

      it('should remove first array element', () => {
        dataModel = { items: [1, 2, 3] };
        const result = removeData(dataModel, '/items/0');
        expect(result).toBe(true);
        expect(dataModel.items).toEqual([2, 3]);
      });

      it('should remove last array element', () => {
        dataModel = { items: [1, 2, 3] };
        const result = removeData(dataModel, '/items/2');
        expect(result).toBe(true);
        expect(dataModel.items).toEqual([1, 2]);
      });
    });

    describe('clear entire data model', () => {
      it('should clear all keys with empty path', () => {
        dataModel = { a: 1, b: 2, c: 3 };
        const result = removeData(dataModel, '');
        expect(result).toBe(true);
        expect(dataModel).toEqual({});
      });

      it('should clear all keys with "/" path', () => {
        dataModel = { x: 'y', z: 'w' };
        const result = removeData(dataModel, '/');
        expect(result).toBe(true);
        expect(dataModel).toEqual({});
      });
    });

    describe('edge cases', () => {
      it('should return false for non-existent path', () => {
        dataModel = { a: 1 };
        const result = removeData(dataModel, '/nonexistent');
        expect(result).toBe(false);
      });

      it('should return false for invalid array index', () => {
        dataModel = { items: [1, 2] };
        const result = removeData(dataModel, '/items/5');
        expect(result).toBe(false);
      });

      it('should return false for invalid path format', () => {
        dataModel = { a: 1 };
        const result = removeData(dataModel, 'invalid');
        expect(result).toBe(false);
      });
    });
  });

  describe('getData', () => {
    it('should get value at path', () => {
      dataModel = { user: { name: 'Alice' } };
      const result = getData(dataModel, '/user/name');
      expect(result).toBe('Alice');
    });

    it('should get array element', () => {
      dataModel = { items: ['a', 'b', 'c'] };
      const result = getData(dataModel, '/items/1');
      expect(result).toBe('b');
    });

    it('should return null for non-existent path', () => {
      dataModel = { a: 1 };
      const result = getData(dataModel, '/nonexistent');
      expect(result).toBeNull();
    });

    it('should return entire data model for empty path', () => {
      dataModel = { key: 'value' };
      const result = getData(dataModel, '');
      expect(result).toEqual({ key: 'value' });
    });
  });
});
