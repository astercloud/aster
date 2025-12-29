/**
 * Aster UI Protocol - CreateSurface Unit Tests
 *
 * Tests for createSurface message processing and catalogId support.
 *
 * @module __tests__/protocol/create-surface
 */

import { describe, expect, it, beforeEach } from 'vitest';
import { createMessageProcessor, MessageProcessor } from '@/protocol/message-processor';

describe('CreateSurface Message', () => {
  let processor: MessageProcessor;

  beforeEach(() => {
    processor = createMessageProcessor();
  });

  describe('basic createSurface', () => {
    it('should create a new surface', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'test-surface',
        },
      });

      const surface = processor.getSurface('test-surface');
      expect(surface).toBeDefined();
      expect(surface?.rootComponentId).toBeNull();
      expect(surface?.componentTree).toBeNull();
      expect(surface?.dataModel).toEqual({});
    });

    it('should create surface with catalogId', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'catalog-surface',
          catalogId: 'https://example.com/components/v1',
        },
      });

      const surface = processor.getSurface('catalog-surface');
      expect(surface).toBeDefined();
      expect(surface?.catalogId).toBe('https://example.com/components/v1');
    });

    it('should reset existing surface', () => {
      // Create initial surface with data
      processor.processMessage({
        createSurface: { surfaceId: 'reset-test' },
      });
      processor.processMessage({
        dataModelUpdate: {
          surfaceId: 'reset-test',
          path: '/',
          contents: { key: 'value' },
        },
      });

      // Verify data exists
      expect(processor.getData('reset-test', '/key')).toBe('value');

      // Create surface again (should reset)
      processor.processMessage({
        createSurface: { surfaceId: 'reset-test' },
      });

      // Data should be cleared
      const surface = processor.getSurface('reset-test');
      expect(surface?.dataModel).toEqual({});
    });

    it('should initialize streaming state', () => {
      processor.processMessage({
        createSurface: { surfaceId: 'stream-test' },
      });

      const streamingState = processor.getStreamingState('stream-test');
      expect(streamingState).toBeDefined();
      expect(streamingState?.isStreaming).toBe(false);
      expect(streamingState?.renderedComponentIds.size).toBe(0);
      expect(streamingState?.pendingComponentIds.size).toBe(0);
    });
  });

  describe('catalogId handling', () => {
    it('should store catalogId from createSurface', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'cat-1',
          catalogId: 'catalog-a',
        },
      });

      expect(processor.getSurface('cat-1')?.catalogId).toBe('catalog-a');
    });

    it('should allow catalogId override in beginRendering', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'cat-2',
          catalogId: 'original-catalog',
        },
      });

      processor.processMessage({
        surfaceUpdate: {
          surfaceId: 'cat-2',
          components: [
            { id: 'root', component: { Text: { text: { literalString: 'Hello' } } } },
          ],
        },
      });

      processor.processMessage({
        beginRendering: {
          surfaceId: 'cat-2',
          root: 'root',
          catalogId: 'override-catalog',
        },
      });

      expect(processor.getSurface('cat-2')?.catalogId).toBe('override-catalog');
    });

    it('should preserve catalogId if not overridden in beginRendering', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'cat-3',
          catalogId: 'preserved-catalog',
        },
      });

      processor.processMessage({
        surfaceUpdate: {
          surfaceId: 'cat-3',
          components: [
            { id: 'root', component: { Text: { text: { literalString: 'Test' } } } },
          ],
        },
      });

      processor.processMessage({
        beginRendering: {
          surfaceId: 'cat-3',
          root: 'root',
          // No catalogId override
        },
      });

      expect(processor.getSurface('cat-3')?.catalogId).toBe('preserved-catalog');
    });

    it('should handle URL-style catalogId', () => {
      processor.processMessage({
        createSurface: {
          surfaceId: 'url-cat',
          catalogId: 'https://components.example.com/catalog/v2.0',
        },
      });

      expect(processor.getSurface('url-cat')?.catalogId).toBe(
        'https://components.example.com/catalog/v2.0',
      );
    });
  });

  describe('multiple surfaces', () => {
    it('should manage multiple surfaces independently', () => {
      processor.processMessage({
        createSurface: { surfaceId: 'surface-a', catalogId: 'cat-a' },
      });
      processor.processMessage({
        createSurface: { surfaceId: 'surface-b', catalogId: 'cat-b' },
      });

      expect(processor.getSurface('surface-a')?.catalogId).toBe('cat-a');
      expect(processor.getSurface('surface-b')?.catalogId).toBe('cat-b');
    });

    it('should not affect other surfaces when creating new one', () => {
      processor.processMessage({
        createSurface: { surfaceId: 'existing' },
      });
      processor.processMessage({
        dataModelUpdate: {
          surfaceId: 'existing',
          path: '/',
          contents: { data: 'preserved' },
        },
      });

      processor.processMessage({
        createSurface: { surfaceId: 'new-surface' },
      });

      expect(processor.getData('existing', '/data')).toBe('preserved');
    });
  });

  describe('surface lifecycle', () => {
    it('should allow full lifecycle: create -> update -> render -> delete', () => {
      // Create
      processor.processMessage({
        createSurface: { surfaceId: 'lifecycle', catalogId: 'test-catalog' },
      });
      expect(processor.hasSurface('lifecycle')).toBe(true);

      // Update components
      processor.processMessage({
        surfaceUpdate: {
          surfaceId: 'lifecycle',
          components: [
            { id: 'root', component: { Text: { text: { literalString: 'Hello' } } } },
          ],
        },
      });

      // Begin rendering
      processor.processMessage({
        beginRendering: { surfaceId: 'lifecycle', root: 'root' },
      });
      expect(processor.getSurface('lifecycle')?.componentTree).not.toBeNull();

      // Delete
      processor.processMessage({
        deleteSurface: { surfaceId: 'lifecycle' },
      });
      expect(processor.hasSurface('lifecycle')).toBe(false);
    });
  });
});
