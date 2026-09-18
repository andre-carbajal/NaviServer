import { describe, expect, it } from 'vitest';

import { mergeAddonResults } from './addons';

describe('mergeAddonResults', () => {
  it('deduplicates pages and keeps the newest result', () => {
    const old = { source: 'modrinth', projectId: 'a', title: 'Old' } as never;
    const fresh = {
      source: 'modrinth',
      projectId: 'a',
      title: 'Fresh',
    } as never;
    expect(mergeAddonResults([old], [fresh])).toEqual([fresh]);
  });
});
