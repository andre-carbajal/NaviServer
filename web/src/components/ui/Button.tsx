import React from 'react';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger';
}

const baseClasses =
  'tw:whitespace-nowrap tw:inline-flex tw:items-center tw:justify-center tw:gap-2 tw:rounded-lg tw:border tw:border-transparent tw:px-4 tw:py-2 tw:text-[0.9rem] tw:leading-[1.5] tw:font-semibold tw:cursor-pointer tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:disabled:pointer-events-none tw:disabled:cursor-not-allowed tw:disabled:opacity-50';

const variantClasses = {
  primary:
    'tw:bg-primary tw:text-white tw:shadow-[0_2px_4px_rgba(0,0,0,0.2)] tw:hover:-translate-y-px tw:hover:bg-primary-hover tw:hover:shadow-[0_4px_8px_rgba(0,0,0,0.3)]',
  secondary:
    'tw:border-white/20 tw:bg-white/[0.02] tw:text-text-main tw:hover:border-text-muted tw:hover:bg-white/5',
  danger:
    'tw:border-[rgba(244,67,54,0.3)] tw:bg-[rgba(244,67,54,0.1)] tw:text-danger tw:hover:border-danger tw:hover:bg-danger tw:hover:text-white tw:hover:shadow-[0_4px_8px_rgba(244,67,54,0.3)]',
} satisfies Record<NonNullable<ButtonProps['variant']>, string>;

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  className = '',
  ...props
}) => {
  return (
    <button
      type="button"
      className={`${baseClasses} ${variantClasses[variant]} ${className}`.trim()}
      {...props}
    >
      {children}
    </button>
  );
};
