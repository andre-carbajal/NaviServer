import type { FormEventHandler, KeyboardEventHandler } from 'react';

import { Button } from '../ui/Button';
import ConsoleView from './ConsoleView';

interface ConsolePanelProps {
  command: string;
  connected: boolean;
  logs: string[];
  onCommandChange: (value: string) => void;
  onKeyDown: KeyboardEventHandler<HTMLInputElement>;
  onSubmit: FormEventHandler<HTMLFormElement>;
}

export const ConsolePanel = ({
  command,
  connected,
  logs,
  onCommandChange,
  onKeyDown,
  onSubmit,
}: ConsolePanelProps) => (
  <div className="tw:flex tw:h-full tw:min-w-0 tw:flex-col tw:gap-2.5 tw:overflow-x-hidden tw:max-[1024px]:h-auto">
    <div className="tw:flex tw:items-center tw:justify-between">
      <h2 className="tw:m-0">Console</h2>
      <span className={connected ? 'tw:text-green-400' : 'tw:text-red-300'}>
        {connected ? '● Connected' : '○ Disconnected'}
      </span>
    </div>
    <ConsoleView logs={logs} />
    <form
      onSubmit={onSubmit}
      className="tw:grid tw:min-w-0 tw:grid-cols-[minmax(0,1fr)_auto] tw:gap-2 tw:max-[1024px]:w-full tw:max-[1024px]:grid-cols-[minmax(0,1fr)_minmax(72px,auto)]"
    >
      <input
        type="text"
        aria-label="Server console command"
        value={command}
        onChange={(event) => onCommandChange(event.target.value)}
        onKeyDown={onKeyDown}
        className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:disabled:cursor-not-allowed tw:disabled:opacity-60 tw:max-[1024px]:text-base tw:min-w-0"
        placeholder="Type a command..."
        disabled={!connected}
      />
      <Button
        type="submit"
        disabled={!connected || !command.trim()}
        className="tw:shrink-0 tw:max-[1024px]:min-w-[72px] tw:max-[1024px]:px-2.5"
      >
        Send
      </Button>
    </form>
  </div>
);
