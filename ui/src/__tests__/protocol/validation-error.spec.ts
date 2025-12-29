/**
 * Aster UI Protocol - Validation Error Unit Tests
 *
 * Tests for validation error generation and format.
 *
 * @module __tests__/protocol/validation-error
 */

import { describe, expect, it, beforeEach } from 'vitest';
import { createMessageProcessor, MessageProcessor } from '@/protocol/message-processor';
import {
  createValidationError,
  createGenericError,
  isValidationError,
} from '@/types/ui-protocol';
import type { ProtocolError, ValidationError } from '@/types/ui-protocol';

describe('Validation Error', () => {
  describe('createValidationError', () => {
    it('should create validation error with all fields', () => {
      const error = createValidationError(
        'surface-1',
        '/dataModelUpdate/contents',
        'contents is required',
      );

      expect(error.code).toBe('VALIDATION_FAILED');
      expect(error.surfaceId).toBe('surface-1');
      expect(error.path).toBe('/dataModelUpdate/contents');
      expect(error.message).toBe('contents is required');
    });

    it('should create error with root path', () => {
      const error = createValidationError('s', '/', 'Invalid message');

      expect(error.path).toBe('/');
    });

    it('should create error with nested path', () => {
      const error = createValidationError(
        's',
        '/surfaceUpdate/components/0/id',
        'id is required',
      );

      expect(error.path).toBe('/surfaceUpdate/components/0/id');
    });
  });

  describe('createGenericError', () => {
    it('should create generic error with custom code', () => {
      const error = createGenericError(
        'UNKNOWN_COMPONENT',
        'surface-1',
        'Component type "CustomWidget" is not registered',
      );

      expect(error.code).toBe('UNKNOWN_COMPONENT');
      expect(error.surfaceId).toBe('surface-1');
      expect(error.message).toBe('Component type "CustomWidget" is not registered');
      expect(error.details).toBeUndefined();
    });

    it('should create generic error with details', () => {
      const error = createGenericError(
        'RENDER_ERROR',
        'surface-1',
        'Failed to render component',
        { componentId: 'btn-1', reason: 'missing props' },
      );

      expect(error.details).toEqual({
        componentId: 'btn-1',
        reason: 'missing props',
      });
    });
  });

  describe('isValidationError', () => {
    it('should return true for validation error', () => {
      const error: ProtocolError = {
        code: 'VALIDATION_FAILED',
        surfaceId: 's',
        path: '/test',
        message: 'test error',
      };

      expect(isValidationError(error)).toBe(true);
    });

    it('should return false for generic error', () => {
      const error: ProtocolError = {
        code: 'OTHER_ERROR',
        surfaceId: 's',
        message: 'other error',
      };

      expect(isValidationError(error)).toBe(false);
    });

    it('should return false for validation code without path', () => {
      const error: ProtocolError = {
        code: 'VALIDATION_FAILED',
        surfaceId: 's',
        message: 'no path',
        // path is undefined
      };

      expect(isValidationError(error)).toBe(false);
    });
  });

  describe('MessageProcessor validation', () => {
    let processor: MessageProcessor;

    beforeEach(() => {
      processor = createMessageProcessor();
      processor.clearValidationErrors();
    });

    describe('validateMessage', () => {
      it('should return error for empty message', () => {
        const errors = processor.validateMessage({});

        expect(errors.length).toBe(1);
        expect(errors[0]?.message).toContain('exactly one operation type');
      });

      it('should return error for multiple operations', () => {
        const errors = processor.validateMessage({
          createSurface: { surfaceId: 's1' },
          deleteSurface: { surfaceId: 's2' },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.message).toContain('exactly one operation type');
      });

      it('should return error for missing surfaceId in createSurface', () => {
        const errors = processor.validateMessage({
          createSurface: { surfaceId: '' },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/createSurface/surfaceId');
      });

      it('should return error for missing surfaceId in surfaceUpdate', () => {
        const errors = processor.validateMessage({
          surfaceUpdate: { surfaceId: '', components: [] },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/surfaceUpdate/surfaceId');
      });

      it('should return error for invalid components in surfaceUpdate', () => {
        const errors = processor.validateMessage({
          surfaceUpdate: {
            surfaceId: 'test',
            components: 'not-an-array' as any,
          },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/surfaceUpdate/components');
      });

      it('should return error for missing surfaceId in dataModelUpdate', () => {
        const errors = processor.validateMessage({
          dataModelUpdate: { surfaceId: '', contents: {} },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/dataModelUpdate/surfaceId');
      });

      it('should return error for invalid op in dataModelUpdate', () => {
        const errors = processor.validateMessage({
          dataModelUpdate: {
            surfaceId: 'test',
            op: 'invalid' as any,
            contents: {},
          },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/dataModelUpdate/op');
      });

      it('should return error for missing contents in add operation', () => {
        const errors = processor.validateMessage({
          dataModelUpdate: {
            surfaceId: 'test',
            op: 'add',
            // contents missing
          },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/dataModelUpdate/contents');
      });

      it('should not require contents for remove operation', () => {
        const errors = processor.validateMessage({
          dataModelUpdate: {
            surfaceId: 'test',
            op: 'remove',
            path: '/item',
            // contents not required for remove
          },
        });

        expect(errors.length).toBe(0);
      });

      it('should return error for missing surfaceId in beginRendering', () => {
        const errors = processor.validateMessage({
          beginRendering: { surfaceId: '', root: 'root' },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/beginRendering/surfaceId');
      });

      it('should return error for missing root in beginRendering', () => {
        const errors = processor.validateMessage({
          beginRendering: { surfaceId: 'test', root: '' },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/beginRendering/root');
      });

      it('should return error for missing surfaceId in deleteSurface', () => {
        const errors = processor.validateMessage({
          deleteSurface: { surfaceId: '' },
        });

        expect(errors.length).toBe(1);
        expect(errors[0]?.path).toBe('/deleteSurface/surfaceId');
      });

      it('should return no errors for valid createSurface', () => {
        const errors = processor.validateMessage({
          createSurface: { surfaceId: 'valid-surface' },
        });

        expect(errors.length).toBe(0);
      });

      it('should return no errors for valid dataModelUpdate', () => {
        const errors = processor.validateMessage({
          dataModelUpdate: {
            surfaceId: 'test',
            op: 'replace',
            contents: { key: 'value' },
          },
        });

        expect(errors.length).toBe(0);
      });
    });

    describe('processMessage with validation', () => {
      it('should store validation errors when processing invalid message', () => {
        processor.processMessage({});

        const errors = processor.getValidationErrors();
        expect(errors.length).toBeGreaterThan(0);
      });

      it('should not process invalid message', () => {
        processor.processMessage({
          createSurface: { surfaceId: '' },
        });

        // Surface should not be created
        expect(processor.hasSurface('')).toBe(false);
      });

      it('should clear validation errors', () => {
        processor.processMessage({});
        expect(processor.getValidationErrors().length).toBeGreaterThan(0);

        processor.clearValidationErrors();
        expect(processor.getValidationErrors().length).toBe(0);
      });

      it('should accumulate validation errors', () => {
        processor.processMessage({});
        processor.processMessage({});

        const errors = processor.getValidationErrors();
        expect(errors.length).toBe(2);
      });
    });
  });

  describe('error format for LLM self-correction', () => {
    it('should provide JSON Pointer path for precise error location', () => {
      const error = createValidationError(
        'surface-1',
        '/surfaceUpdate/components/2/component/Button/label',
        'label property is required',
      );

      // Path should be a valid JSON Pointer
      expect(error.path.startsWith('/')).toBe(true);
      expect(error.path.split('/').length).toBeGreaterThan(1);
    });

    it('should provide clear error message', () => {
      const error = createValidationError(
        's',
        '/path',
        'Expected string but got number',
      );

      expect(error.message).toBeTruthy();
      expect(typeof error.message).toBe('string');
    });

    it('should include surfaceId for context', () => {
      const error = createValidationError(
        'my-form-surface',
        '/path',
        'error',
      );

      expect(error.surfaceId).toBe('my-form-surface');
    });
  });
});
