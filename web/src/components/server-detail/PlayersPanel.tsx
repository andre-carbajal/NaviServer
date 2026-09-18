import { Ban, Globe, Search } from 'lucide-react';

import type {
  BannedListItem,
  PlayerListItem,
} from '../../hooks/usePlayerLists';
import PlayerAvatar from './PlayerAvatar';

export type PlayerFilter = 'all' | 'admins' | 'banned';

interface PlayersPanelProps {
  banned: BannedListItem[];
  bannedCount: number;
  canModerate: boolean;
  filter: PlayerFilter;
  isActionLoading: boolean;
  maxPlayers: number;
  online: PlayerListItem[];
  onlineCount: number;
  onlinePlayers: number;
  operators: PlayerListItem[];
  operatorCount: number;
  search: string;
  onFilterChange: (filter: PlayerFilter) => void;
  onPardon: (item: BannedListItem) => void;
  onSearchChange: (value: string) => void;
  onSelectPlayer: (player: PlayerListItem, isOperator: boolean) => void;
}

const Empty = ({ children }: { children: string }) => (
  <div className="tw:p-6 tw:text-center tw:text-text-muted">{children}</div>
);

const PlayerRows = ({
  canModerate,
  items,
  operator,
  onSelect,
}: {
  canModerate: boolean;
  items: PlayerListItem[];
  operator: boolean;
  onSelect: (player: PlayerListItem, isOperator: boolean) => void;
}) => (
  <ul className="tw:m-0 tw:flex tw:list-none tw:flex-col tw:gap-2.5 tw:p-0">
    {items.map((player) => (
      <li
        key={player.key}
        className={`tw:flex tw:items-center tw:gap-2.5 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/3 tw:p-2.5 ${canModerate ? 'tw:hover:border-[rgba(167,139,250,0.45)] tw:hover:bg-[rgba(167,139,250,0.1)]' : ''}`}
      >
        <button
          type="button"
          className="tw:flex tw:min-w-0 tw:flex-1 tw:items-center tw:gap-2.5 tw:border-0 tw:bg-transparent tw:p-0 tw:text-left tw:font-[inherit] tw:text-text-main"
          disabled={!canModerate}
          onClick={() => onSelect(player, operator)}
        >
          <PlayerAvatar player={{ name: player.name, id: player.uuid || '' }} />
          <div className="tw:flex tw:min-w-0 tw:flex-col tw:gap-0.5">
            <strong>{player.name}</strong>
            <small className="tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap tw:text-[0.72rem] tw:text-text-muted">
              {player.uuid || 'No UUID available'}
            </small>
          </div>
          <span
            className={`tw:ml-auto tw:rounded-full tw:border tw:px-2 tw:py-[3px] tw:text-[0.7rem] ${player.isOnline ? 'tw:border-green-400/35 tw:bg-green-400/12 tw:text-green-400' : 'tw:border-border tw:bg-white/3 tw:text-text-muted'}`}
          >
            {player.isOnline ? 'Online' : 'Offline'}
          </span>
        </button>
      </li>
    ))}
  </ul>
);

export const PlayersPanel = ({
  banned,
  bannedCount,
  canModerate,
  filter,
  isActionLoading,
  maxPlayers,
  online,
  onlineCount,
  onlinePlayers,
  operators,
  operatorCount,
  search,
  onFilterChange,
  onPardon,
  onSearchChange,
  onSelectPlayer,
}: PlayersPanelProps) => (
  <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-3.5">
    <div className="tw:mb-2.5 tw:flex tw:items-center tw:justify-between tw:gap-2.5">
      <h2 className="tw:m-0">Player Management</h2>
      <span>
        Online {onlinePlayers}/{maxPlayers}
      </span>
    </div>
    <div className="tw:mb-3 tw:flex tw:flex-col tw:gap-2.5">
      <label className="tw:flex tw:items-center tw:gap-2 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/3 tw:px-2.5 tw:text-text-muted">
        <Search size={16} />
        <input
          type="text"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search players..."
          className="tw:w-full tw:border-0 tw:bg-transparent tw:py-2.5 tw:text-text-main tw:outline-none"
        />
      </label>
      <div className="tw:flex tw:flex-wrap tw:gap-2">
        {(
          [
            ['all', 'All', onlineCount],
            ['admins', 'Admins', operatorCount],
            ['banned', 'Banned', bannedCount],
          ] as const
        ).map(([value, label, count]) => (
          <button
            key={value}
            type="button"
            className={`tw:cursor-pointer tw:rounded-full tw:border tw:border-border tw:bg-white/3 tw:px-3 tw:py-[7px] tw:text-[0.82rem] tw:text-text-muted ${filter === value ? 'tw:border-[rgba(167,139,250,0.45)] tw:bg-[rgba(167,139,250,0.2)] tw:text-white' : ''}`}
            onClick={() => onFilterChange(value)}
          >
            {label} ({count})
          </button>
        ))}
      </div>
    </div>
    {filter === 'all' &&
      (online.length ? (
        <PlayerRows
          canModerate={canModerate}
          items={online}
          operator={false}
          onSelect={onSelectPlayer}
        />
      ) : (
        <Empty>No players found</Empty>
      ))}
    {filter === 'admins' &&
      (operators.length ? (
        <PlayerRows
          canModerate={canModerate}
          items={operators}
          operator
          onSelect={onSelectPlayer}
        />
      ) : (
        <Empty>No operators found</Empty>
      ))}
    {filter === 'banned' &&
      (banned.length ? (
        <ul className="tw:m-0 tw:flex tw:list-none tw:flex-col tw:gap-2.5 tw:p-0">
          {banned.map((item) => (
            <li
              key={item.key}
              className="tw:flex tw:items-center tw:gap-2.5 tw:rounded-[10px] tw:border tw:border-border tw:bg-white/3 tw:p-2.5"
            >
              <div className="tw:flex tw:h-7 tw:w-7 tw:shrink-0 tw:items-center tw:justify-center tw:rounded-md tw:border tw:border-border tw:bg-white/4 tw:text-text-muted">
                {item.type === 'player' ? (
                  <Ban size={16} />
                ) : (
                  <Globe size={16} />
                )}
              </div>
              <div className="tw:flex tw:min-w-0 tw:flex-col tw:gap-0.5">
                <strong>{item.label}</strong>
                <small className="tw:overflow-hidden tw:text-ellipsis tw:whitespace-nowrap tw:text-[0.72rem] tw:text-text-muted">
                  {item.type === 'player'
                    ? item.uuid || item.detail || 'Banned player'
                    : item.detail || 'Banned IP'}
                </small>
              </div>
              <button
                type="button"
                className="tw:ml-auto tw:cursor-pointer tw:rounded-lg tw:border tw:border-emerald-400/35 tw:bg-emerald-400/12 tw:px-2.5 tw:py-1.5 tw:text-emerald-300 tw:disabled:cursor-not-allowed tw:disabled:opacity-50"
                onClick={() => onPardon(item)}
                disabled={!canModerate || isActionLoading}
              >
                Pardon
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <Empty>No banned entries</Empty>
      ))}
    {!canModerate && (
      <p className="tw:mb-0 tw:mt-2.5 tw:text-[0.82rem] tw:text-text-muted">
        You can view players, but moderation actions require console permission.
      </p>
    )}
  </div>
);
