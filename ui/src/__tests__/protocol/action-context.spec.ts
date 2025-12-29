/**
 * Aster UI Protocol - Action Context Unit Tests
 *
 * Tests for actionContext resolution and UIActionEvent generation.
 *
 * @module __tests__/protocol/action-context
 */

import { describe, expect, it } from 'vitest';
import { resolveActionContext } from '@/protocol/client-message';
import { createUIActionEvent, createUserActionMessage } from '@/types/ui-protocol';
import type { PropertyValue, DataMap } from '@/types/ui-protocol';

describe('Action Context', () => {
  describe('resolveActionContext', () => {
    describe('literal value resolution', () => {
      it('should resolve literalString values', () => {
        const actionContext: Record<string, PropertyValue> = {
          name: { literalString: 'Alice' },
        };
        const dataModel: DataMap = {};

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ name: 'Alice' });
      });

      it('should resolve literalNumber values', () => {
        const actionContext: Record<string, PropertyValue> = {
          count: { literalNumber: 42 },
        };
        const dataModel: DataMap = {};

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ count: 42 });
      });

      it('should resolve literalBoolean values', () => {
        const actionContext: Record<string, PropertyValue> = {
          enabled: { literalBoolean: true },
        };
        const dataModel: DataMap = {};

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ enabled: true });
      });

      it('should resolve mixed literal values', () => {
        const actionContext: Record<string, PropertyValue> = {
          name: { literalString: 'Test' },
          count: { literalNumber: 10 },
          active: { literalBoolean: false },
        };
        const dataModel: DataMap = {};

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({
          name: 'Test',
          count: 10,
          active: false,
        });
      });
    });

    describe('path reference resolution', () => {
      it('should resolve simple path reference', () => {
        const actionContext: Record<string, PropertyValue> = {
          userName: { path: '/user/name' },
        };
        const dataModel: DataMap = {
          user: { name: 'Bob' },
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ userName: 'Bob' });
      });

      it('should resolve multiple path references', () => {
        const actionContext: Record<string, PropertyValue> = {
          firstName: { path: '/user/firstName' },
          lastName: { path: '/user/lastName' },
          age: { path: '/user/age' },
        };
        const dataModel: DataMap = {
          user: {
            firstName: 'John',
            lastName: 'Doe',
            age: 30,
          },
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({
          firstName: 'John',
          lastName: 'Doe',
          age: 30,
        });
      });

      it('should resolve array element path', () => {
        const actionContext: Record<string, PropertyValue> = {
          selectedItem: { path: '/items/1' },
        };
        const dataModel: DataMap = {
          items: ['first', 'second', 'third'],
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ selectedItem: 'second' });
      });

      it('should return undefined for non-existent path', () => {
        const actionContext: Record<string, PropertyValue> = {
          missing: { path: '/nonexistent/path' },
        };
        const dataModel: DataMap = { other: 'data' };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ missing: undefined });
      });
    });

    describe('mixed literal and path values', () => {
      it('should resolve both literal and path values', () => {
        const actionContext: Record<string, PropertyValue> = {
          action: { literalString: 'submit' },
          userId: { path: '/user/id' },
          timestamp: { literalNumber: 1234567890 },
        };
        const dataModel: DataMap = {
          user: { id: 'user-123' },
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({
          action: 'submit',
          userId: 'user-123',
          timestamp: 1234567890,
        });
      });
    });

    describe('nested object resolution', () => {
      it('should resolve path to nested object', () => {
        const actionContext: Record<string, PropertyValue> = {
          formData: { path: '/form' },
        };
        const dataModel: DataMap = {
          form: {
            name: 'Test',
            email: 'test@example.com',
          },
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({
          formData: {
            name: 'Test',
            email: 'test@example.com',
          },
        });
      });

      it('should resolve path to array', () => {
        const actionContext: Record<string, PropertyValue> = {
          selectedItems: { path: '/selection' },
        };
        const dataModel: DataMap = {
          selection: [1, 2, 3],
        };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({
          selectedItems: [1, 2, 3],
        });
      });
    });

    describe('edge cases', () => {
      it('should handle empty actionContext', () => {
        const actionContext: Record<string, PropertyValue> = {};
        const dataModel: DataMap = { data: 'value' };

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({});
      });

      it('should handle empty dataModel', () => {
        const actionContext: Record<string, PropertyValue> = {
          value: { path: '/missing' },
        };
        const dataModel: DataMap = {};

        const result = resolveActionContext(actionContext, dataModel);

        expect(result).toEqual({ value: undefined });
      });

      it('should handle null values in data model', () => {
        const actionContext: Record<string, PropertyValue> = {
          nullValue: { path: '/nullField' },
        };
        const dataModel: DataMap = { nullField: null };

        const result = resolveActionContext(actionContext, dataModel);

        // getData returns null for null values, resolvePropertyValue converts to undefined
        expect(result.nullValue).toBeUndefined();
      });
    });
  });

  describe('createUIActionEvent', () => {
    it('should create event with all required fields', () => {
      const event = createUIActionEvent(
        'surface-1',
        'button-1',
        'click',
        { key: 'value' },
      );

      expect(event.surfaceId).toBe('surface-1');
      expect(event.componentId).toBe('button-1');
      expect(event.action).toBe('click');
      expect(event.context).toEqual({ key: 'value' });
      expect(event.timestamp).toBeDefined();
      expect(typeof event.timestamp).toBe('string');
    });

    it('should create ISO 8601 timestamp', () => {
      const event = createUIActionEvent('s', 'c', 'a', {});

      // ISO 8601 format check
      const isoRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z$/;
      expect(event.timestamp).toMatch(isoRegex);
    });

    it('should handle empty context', () => {
      const event = createUIActionEvent('s', 'c', 'a', {});

      expect(event.context).toEqual({});
    });

    it('should handle complex context', () => {
      const context = {
        user: { id: 1, name: 'Test' },
        items: [1, 2, 3],
        nested: { deep: { value: true } },
      };

      const event = createUIActionEvent('s', 'c', 'a', context);

      expect(event.context).toEqual(context);
    });
  });

  describe('createUserActionMessage', () => {
    it('should create message with all required fields', () => {
      const message = createUserActionMessage(
        'submit',
        'surface-1',
        'form-1',
        { formData: 'test' },
      );

      expect(message.name).toBe('submit');
      expect(message.surfaceId).toBe('surface-1');
      expect(message.sourceComponentId).toBe('form-1');
      expect(message.context).toEqual({ formData: 'test' });
      expect(message.timestamp).toBeDefined();
    });

    it('should create ISO 8601 timestamp', () => {
      const message = createUserActionMessage('a', 's', 'c', {});

      const isoRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z$/;
      expect(message.timestamp).toMatch(isoRegex);
    });

    it('should handle empty context', () => {
      const message = createUserActionMessage('a', 's', 'c', {});

      expect(message.context).toEqual({});
    });
  });
});
