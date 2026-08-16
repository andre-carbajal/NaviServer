import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import AddonsPanel from './AddonsPanel';

const { mockApi } = vi.hoisted(() => ({
  mockApi: {
    listAddons: vi.fn(),
    syncAddons: vi.fn(),
    searchAddons: vi.fn(),
    getAddonVersions: vi.fn(),
    previewAddonInstall: vi.fn(),
    installAddon: vi.fn(),
    updateAllAddons: vi.fn(),
    updateAddon: vi.fn(),
    setAddonDisabled: vi.fn(),
    deleteAddon: vi.fn(),
  },
}));

vi.mock('../services/api', () => ({
  api: mockApi,
}));

const server = {
  id: 'server-1',
  name: 'Fabric server',
  version: '1.21.1',
  loader: 'fabric',
  port: 25565,
  ram: 2048,
  status: 'STOPPED' as const,
};

const rootResult = {
  source: 'modrinth' as const,
  projectId: 'root-project',
  projectName: 'Root mod',
  latest: {
    versionId: 'root-version',
    versionName: 'Root version',
    versionLabel: '1.0.0',
    releaseType: 'release' as const,
    downloadUrl: 'https://example.test/root.jar',
    filename: 'root.jar',
    source: 'modrinth' as const,
  },
};

const configureApi = () => {
  mockApi.listAddons.mockResolvedValue({
    data: { addonType: 'mod', items: [] },
  });
  mockApi.searchAddons.mockResolvedValue({
    data: { items: [rootResult], hasMore: false, nextOffset: 1 },
  });
  mockApi.getAddonVersions.mockResolvedValue({
    data: { versions: [rootResult.latest] },
  });
  mockApi.previewAddonInstall.mockResolvedValue({
    data: {
      dependencies: [
        {
          name: 'Fabric API',
          source: 'modrinth',
          projectId: 'fabric-api',
          iconUrl: 'https://cdn.example.test/fabric-api.png',
          versionId: 'fabric-api-version',
          versionLabel: '0.100.0',
          filename: 'fabric-api.jar',
        },
      ],
    },
  });
};

describe('AddonsPanel install summary', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    configureApi();
  });

  afterEach(() => {
    cleanup();
  });

  it('shows required dependencies returned by the install preview', async () => {
    render(<AddonsPanel server={server} canManage />);

    fireEvent.click(
      await screen.findByRole('button', { name: 'Install Mods' }),
    );
    expect(await screen.findByText('Root mod')).toBeTruthy();
    const checkboxes = await screen.findAllByRole('checkbox');
    fireEvent.click(checkboxes[1]);
    fireEvent.click(screen.getByRole('button', { name: 'Install (1)' }));

    await waitFor(() => expect(mockApi.previewAddonInstall).toHaveBeenCalled());
    expect(await screen.findByText('Required dependencies (1)')).toBeTruthy();
    expect(await screen.findByText('Fabric API')).toBeTruthy();
    expect(screen.getByAltText('Fabric API icon')).toBeTruthy();
    expect(screen.getByText('modrinth • 0.100.0')).toBeTruthy();
    expect(screen.getByText('fabric-api.jar')).toBeTruthy();
    await waitFor(() =>
      expect(mockApi.previewAddonInstall).toHaveBeenCalledWith('server-1', {
        source: 'modrinth',
        projectId: 'root-project',
        versionId: 'root-version',
        fileId: undefined,
      }),
    );
  });

  it('does not request or show dependency previews when disabled', async () => {
    render(<AddonsPanel server={server} canManage />);

    const includeDependencies = await screen.findByRole('checkbox', {
      name: 'Include dependencies',
    });
    fireEvent.click(includeDependencies);
    fireEvent.click(
      await screen.findByRole('button', { name: 'Install Mods' }),
    );
    expect(await screen.findByText('Root mod')).toBeTruthy();
    const checkboxes = await screen.findAllByRole('checkbox');
    fireEvent.click(checkboxes[1]);
    fireEvent.click(screen.getByRole('button', { name: 'Install (1)' }));

    await screen.findByRole('heading', { name: 'Install Summary (1)' });
    expect(screen.queryByText(/Required dependencies/)).toBeNull();
    expect(mockApi.previewAddonInstall).not.toHaveBeenCalled();
  });
});
