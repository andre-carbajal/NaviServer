import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import AutomaticBackupSettings, {
  type AutoBackupDraft,
} from './AutomaticBackupSettings';

const server = {
  id: 'server-1',
  name: 'Survival',
  version: '1.21.1',
  loader: 'vanilla',
  port: 25565,
  ram: 2048,
  status: 'STOPPED' as const,
};

const draft: AutoBackupDraft = {
  enabled: false,
  intervalValue: 24,
  intervalUnit: 'hour',
  maxBackups: 10,
  saving: false,
  dirty: false,
  saved: false,
};

describe('AutomaticBackupSettings', () => {
  it('shows a read-only message to non-admin users', () => {
    render(
      <AutomaticBackupSettings
        canConfigure={false}
        servers={[server]}
        drafts={{ [server.id]: draft }}
        onUpdate={vi.fn()}
        onSave={vi.fn()}
      />,
    );

    expect(
      screen.getByText('Only administrators can configure automatic backups.'),
    ).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Save' })).toBeNull();
  });

  it('forwards edits and save actions for administrators', () => {
    const onUpdate = vi.fn();
    const onSave = vi.fn();
    render(
      <AutomaticBackupSettings
        canConfigure
        servers={[server]}
        drafts={{ [server.id]: draft }}
        onUpdate={onUpdate}
        onSave={onSave}
      />,
    );

    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onUpdate).toHaveBeenCalledWith(server.id, { enabled: true }, true);
    expect(onSave).toHaveBeenCalledWith(server.id);
  });
});
