import { Check, Copy } from 'lucide-react';

import React from 'react';

import { useCopy } from '../../hooks/useCopy';
import { Button } from './Button';

interface CopyButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  text: string;
  iconSize?: number;
  variant?: 'primary' | 'secondary' | 'danger';
  hideIcon?: boolean;
  onCopy?: () => void;
}

export const CopyButton: React.FC<CopyButtonProps> = ({
  text,
  iconSize = 14,
  variant = 'secondary',
  hideIcon = false,
  className = '',
  children,
  onCopy,
  ...props
}) => {
  const { copied, copy } = useCopy();

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    const success = await copy(text);
    if (success && onCopy) onCopy();
  };

  return (
    <Button
      variant={variant}
      onClick={handleCopy}
      type="button"
      className={`${className} ${copied ? 'copied' : ''}`}
      {...props}
    >
      {!hideIcon &&
        (copied ? <Check size={iconSize} /> : <Copy size={iconSize} />)}
      {children}
    </Button>
  );
};
