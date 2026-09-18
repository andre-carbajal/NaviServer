import { beforeEach, describe, expect, it } from 'vitest';

import { readCreatingBackups, writeCreatingBackups } from './backups';

describe('creating backup persistence', () => {
  beforeEach(() => localStorage.clear());

  it('migrates the legacy entry when writing', () => {
    localStorage.setItem('creating_backups', JSON.stringify([{ name: 'old' }]));
    expect(readCreatingBackups()).toEqual([{ name: 'old' }]);
    writeCreatingBackups([]);
    expect(localStorage.getItem('creating_backups')).toBeNull();
    expect(readCreatingBackups()).toEqual([]);
  });
});
