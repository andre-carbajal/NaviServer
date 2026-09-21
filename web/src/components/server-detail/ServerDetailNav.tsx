import {
  BarChart3,
  HardDrive,
  Package,
  Settings2,
  Terminal,
  Users,
} from 'lucide-react';

export type DetailTab =
  'performance' | 'console' | 'players' | 'files' | 'addons' | 'settings';

interface ServerDetailNavProps {
  activeTab: DetailTab;
  addonsLabel: string;
  supportsAddons: boolean;
  onSelect: (tab: DetailTab) => void;
}

const items = [
  ['performance', 'Performance', BarChart3],
  ['console', 'Console', Terminal],
  ['players', 'Players', Users],
  ['files', 'Files', HardDrive],
] as const;

const buttonClass =
  'tw:flex tw:w-full tw:min-w-0 tw:cursor-pointer tw:items-center tw:gap-2 tw:overflow-hidden tw:rounded-[10px] tw:border tw:border-transparent tw:bg-transparent tw:px-3 tw:py-2.5 tw:text-left tw:text-text-muted tw:max-[1024px]:min-w-0 tw:max-[1024px]:flex-[1_1_calc(33.333%_-_8px)] tw:max-[640px]:justify-start tw:max-[640px]:px-2.5 tw:max-[640px]:py-[9px] tw:max-[640px]:text-[0.9rem]';

export const ServerDetailNav = ({
  activeTab,
  addonsLabel,
  supportsAddons,
  onSelect,
}: ServerDetailNavProps) => {
  const renderButton = (
    tab: DetailTab,
    label: string,
    Icon: typeof BarChart3,
  ) => (
    <button
      key={tab}
      type="button"
      className={`${buttonClass} ${activeTab === tab ? 'tw:!border-primary tw:!bg-[rgba(100,108,255,0.12)] tw:!text-white' : ''}`}
      onClick={() => onSelect(tab)}
    >
      <Icon size={16} />
      <span className="tw:min-w-0 tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap">
        {label}
      </span>
    </button>
  );

  return (
    <aside className="tw:box-border tw:flex tw:min-w-0 tw:h-fit tw:flex-col tw:gap-2 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-2.5 tw:max-[1024px]:order-[-1] tw:max-[1024px]:flex-row tw:max-[1024px]:flex-wrap tw:max-[1024px]:overflow-visible tw:max-[640px]:grid tw:max-[640px]:grid-cols-2 tw:max-[640px]:gap-2 tw:max-[640px]:p-2 tw:max-[320px]:grid-cols-1">
      {items.map(([tab, label, Icon]) => renderButton(tab, label, Icon))}
      {supportsAddons && renderButton('addons', addonsLabel, Package)}
      {renderButton('settings', 'Settings', Settings2)}
    </aside>
  );
};
