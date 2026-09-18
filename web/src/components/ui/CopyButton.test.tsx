import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { CopyButton } from './CopyButton';

describe('CopyButton', () => {
  it('renders a visible, non-shrinking copy icon', () => {
    const { container } = render(
      <CopyButton text="localhost:25565" title="Copy" />,
    );
    expect(screen.getByRole('button', { name: 'Copy' })).toBeTruthy();
    expect(container.querySelector('svg')?.classList).toContain('lucide-copy');
    expect(screen.getByRole('button', { name: 'Copy' }).className).toContain(
      'tw:[&_svg]:shrink-0',
    );
  });
});
