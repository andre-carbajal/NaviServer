import { describe, expect, it } from 'vitest';

import { buildPlayerLists } from './usePlayerLists';

describe('buildPlayerLists', () => {
  it('matches online operators and filters every list', () => {
    const result = buildPlayerLists(
      [{ id: 'abc', name: 'Alex' }],
      [{ uuid: 'abc', name: 'Alex' }, { name: 'Steve' }],
      [{ name: 'Griefer', reason: 'spam' }],
      [],
      'alex',
    );
    expect(result.filteredOnlineItems).toHaveLength(1);
    expect(result.filteredOperatorItems).toEqual([
      expect.objectContaining({ name: 'Alex', isOnline: true }),
    ]);
    expect(result.filteredBannedItems).toHaveLength(0);
  });
});
