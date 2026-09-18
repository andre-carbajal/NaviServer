import type { AddonSearchResult } from '../types';

export const addonProjectKey = (
  addon: Pick<AddonSearchResult, 'source' | 'projectId'>,
) => `${addon.source}-${addon.projectId}`;

export const mergeAddonResults = (
  current: AddonSearchResult[],
  next: AddonSearchResult[],
) =>
  Array.from(
    new Map(
      [...current, ...next].map((addon) => [addonProjectKey(addon), addon]),
    ).values(),
  );
