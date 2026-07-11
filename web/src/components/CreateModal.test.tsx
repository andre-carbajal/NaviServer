import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import CreateModal from './CreateModal';

const { mockApi } = vi.hoisted(() => ({
  mockApi: {
    getLoaders: vi.fn(),
    getLoaderMetadata: vi.fn(),
  },
}));

vi.mock('../services/api.ts', () => ({
  api: mockApi,
}));

describe('CreateModal Bedrock support', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.getLoaders.mockResolvedValue({
      data: ['vanilla', 'bedrock'],
    });
    mockApi.getLoaderMetadata.mockImplementation((loader: string) =>
      Promise.resolve({
        data:
          loader === 'bedrock'
            ? {
                latestVersion: '1.26.33',
                minecraftVersions: ['1.26.33', '1.26.40-preview.30'],
              }
            : {
                latestVersion: '1.21.8',
                minecraftVersions: ['1.21.8'],
              },
      }),
    );
  });

  it('shows Bedrock previews, hides JVM RAM, and submits the compatibility RAM value', async () => {
    const onCreate = vi.fn();
    render(<CreateModal isOpen onClose={vi.fn()} onCreate={onCreate} />);

    const loaderSelect = await screen.findByLabelText('Loader');
    fireEvent.change(loaderSelect, { target: { value: 'bedrock' } });

    expect(await screen.findByText('Show previews')).toBeTruthy();
    expect(screen.queryByLabelText('RAM (MB)')).toBeNull();
    expect(screen.getByText(/manages memory automatically/i)).toBeTruthy();

    fireEvent.click(screen.getByLabelText('Show previews'));
    await waitFor(() =>
      expect(mockApi.getLoaderMetadata).toHaveBeenCalledWith(
        'bedrock',
        expect.objectContaining({ includeSnapshots: true }),
      ),
    );

    fireEvent.change(screen.getByLabelText('Server Name'), {
      target: { value: 'Bedrock Test' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create Server' }));

    expect(onCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Bedrock Test',
        loader: 'bedrock',
        ram: 4096,
      }),
    );
  });
});
