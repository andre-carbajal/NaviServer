import type { Backup } from '../types';

export interface CreatingBackup extends Backup {
  serverId: string;
}

const storageKey = 'creating_backups:v1';
const legacyStorageKey = 'creating_backups';

export const readCreatingBackups = (): CreatingBackup[] => {
  const stored =
    localStorage.getItem(storageKey) ?? localStorage.getItem(legacyStorageKey);
  if (!stored) return [];
  try {
    return JSON.parse(stored) as CreatingBackup[];
  } catch (error) {
    console.error(error);
    return [];
  }
};

export const writeCreatingBackups = (backups: CreatingBackup[]) => {
  localStorage.setItem(storageKey, JSON.stringify(backups));
  localStorage.removeItem(legacyStorageKey);
};

export const formatBackupDateTime = (value?: string) => {
  if (!value) return '-';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString();
};
