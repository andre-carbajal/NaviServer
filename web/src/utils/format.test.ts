import { describe, expect, it } from 'vitest';

import { formatBytes, humanSize } from './format';

describe('format utilities', () => {
  it('formats bytes using binary units', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(5 * 1024 * 1024)).toBe('5 MB');
  });

  it('formats estimated sizes for settings display', () => {
    expect(humanSize(512)).toBe('512 B');
    expect(humanSize(2048)).toBe('2.00 KB');
    expect(humanSize(3 * 1024 * 1024)).toBe('3.00 MB');
    expect(humanSize(2 * 1024 * 1024 * 1024)).toBe('2.00 GB');
  });
});
