import { describe, expect, it } from 'vitest';

import {
  formatFileSize,
  getParentServerPath,
  isEditableFile,
  joinServerPath,
} from './fileExplorer';

describe('file explorer utilities', () => {
  it('normalizes paths without changing root behavior', () => {
    expect(joinServerPath('/', 'world')).toBe('/world');
    expect(joinServerPath('/world', 'data')).toBe('/world/data');
    expect(getParentServerPath('/world/data')).toBe('/world');
    expect(getParentServerPath('/world')).toBe('/');
  });

  it('recognizes editable files and formats sizes', () => {
    expect(isEditableFile('server.properties')).toBe(true);
    expect(isEditableFile('MOD.JAR')).toBe(false);
    expect(formatFileSize(1024)).toBe('1 KB');
  });
});
