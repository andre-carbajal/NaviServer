import { useMemo } from 'react';

import type { PlayerInfo } from '../types';

export interface OperatorEntry {
  uuid?: string;
  name?: string;
  level?: number;
  bypassesPlayerLimit?: boolean;
}

export interface BannedPlayerEntry {
  uuid?: string;
  name?: string;
  created?: string;
  source?: string;
  expires?: string;
  reason?: string;
}

export interface BannedIPEntry {
  ip?: string;
  created?: string;
  source?: string;
  expires?: string;
  reason?: string;
}

export interface PlayerListItem {
  key: string;
  name: string;
  uuid?: string;
  isOnline: boolean;
  source: 'online' | 'operator';
}

export interface BannedListItem {
  key: string;
  type: 'player' | 'ip';
  label: string;
  uuid?: string;
  detail?: string;
}

export const buildPlayerLists = (
  players: PlayerInfo[],
  operators: OperatorEntry[],
  bannedPlayers: BannedPlayerEntry[],
  bannedIps: BannedIPEntry[],
  search: string,
) => {
  const query = search.trim().toLowerCase();
  const names = new Set(players.map((player) => player.name.toLowerCase()));
  const ids = new Set(
    players.flatMap((player) => (player.id ? [player.id.toLowerCase()] : [])),
  );
  const matches = (name: string, uuid?: string, detail?: string) =>
    !query ||
    name.toLowerCase().includes(query) ||
    uuid?.toLowerCase().includes(query) ||
    detail?.toLowerCase().includes(query);

  const onlineItems: PlayerListItem[] = players.map((player, index) => ({
    key: `${player.id || player.name}-${index}`,
    name: player.name,
    uuid: player.id,
    isOnline: true,
    source: 'online',
  }));
  const operatorItems: PlayerListItem[] = operators.map((operator, index) => ({
    key: `op-${operator.uuid || operator.name || index}`,
    name: operator.name || operator.uuid || `Operator ${index + 1}`,
    uuid: operator.uuid,
    isOnline: Boolean(
      (operator.name && names.has(operator.name.toLowerCase())) ||
      (operator.uuid && ids.has(operator.uuid.toLowerCase())),
    ),
    source: 'operator',
  }));
  const bannedItems: BannedListItem[] = [
    ...bannedPlayers.map((entry, index) => ({
      key: `bp-${entry.uuid || entry.name || index}`,
      type: 'player' as const,
      label: entry.name || entry.uuid || `Banned player ${index + 1}`,
      uuid: entry.uuid,
      detail: entry.reason || entry.source,
    })),
    ...bannedIps.map((entry, index) => ({
      key: `bi-${entry.ip || index}`,
      type: 'ip' as const,
      label: entry.ip || `Banned IP ${index + 1}`,
      detail: entry.reason || entry.source,
    })),
  ];
  return {
    onlineItems,
    operatorItems,
    bannedItems,
    operatorNameSet: new Set(
      operators.flatMap((entry) =>
        entry.name ? [entry.name.toLowerCase()] : [],
      ),
    ),
    operatorUuidSet: new Set(
      operators.flatMap((entry) =>
        entry.uuid ? [entry.uuid.toLowerCase()] : [],
      ),
    ),
    filteredOnlineItems: onlineItems.filter((entry) =>
      matches(entry.name, entry.uuid),
    ),
    filteredOperatorItems: operatorItems
      .toSorted((a, b) =>
        a.isOnline === b.isOnline
          ? a.name.localeCompare(b.name)
          : a.isOnline
            ? -1
            : 1,
      )
      .filter((entry) => matches(entry.name, entry.uuid)),
    filteredBannedItems: bannedItems.filter((entry) =>
      matches(entry.label, entry.uuid, entry.detail),
    ),
  };
};

export const usePlayerLists = (
  players: PlayerInfo[],
  operators: OperatorEntry[],
  bannedPlayers: BannedPlayerEntry[],
  bannedIps: BannedIPEntry[],
  search: string,
) =>
  useMemo(
    () =>
      buildPlayerLists(players, operators, bannedPlayers, bannedIps, search),
    [bannedIps, bannedPlayers, operators, players, search],
  );
