import { describe, expect, it } from 'vitest';
import { formatFileSize, formatTime, truncate } from '@/utils/format';

describe('format utils', () => {
  describe('truncate', () => {
    it('should return original string if shorter than max length', () => {
      expect(truncate('hello', 10)).toBe('hello');
    });

    it('should truncate string and add ellipsis', () => {
      expect(truncate('hello world', 8)).toBe('hello...');
    });

    it('should handle empty string', () => {
      expect(truncate('', 10)).toBe('');
    });
  });

  describe('formatFileSize', () => {
    it('should format bytes', () => {
      expect(formatFileSize(500)).toBe('500.00 B');
    });

    it('should format kilobytes', () => {
      expect(formatFileSize(1024)).toBe('1.00 KB');
    });

    it('should format megabytes', () => {
      expect(formatFileSize(1024 * 1024)).toBe('1.00 MB');
    });
  });
});
