import type { AxiosError } from 'axios';
import {
  Download,
  Globe,
  Loader2,
  Power,
  RefreshCw,
  Search,
  Trash2,
  Upload,
} from 'lucide-react';

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import { api } from '../../services/api';
import type {
  Addon,
  AddonInstallDependency,
  AddonListResponse,
  AddonSearchResult,
  AddonSource,
  Server,
} from '../../types';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';

interface AddonsPanelProps {
  server: Server;
  canManage: boolean;
  confirmAction: (
    title: string,
    message: string,
    confirmText: string,
  ) => Promise<boolean>;
}

const AddonsPanel: React.FC<AddonsPanelProps> = ({
  server,
  canManage,
  confirmAction,
}) => {
  const SEARCH_BATCH_SIZE = 20;
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<Addon[]>([]);
  const [addonType, setAddonType] = useState<'mod' | 'plugin'>('mod');
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchSource, setSearchSource] = useState<'modrinth' | 'curseforge'>(
    'modrinth',
  );
  const [searching, setSearching] = useState(false);
  const [searchingMore, setSearchingMore] = useState(false);
  const [searchResults, setSearchResults] = useState<AddonSearchResult[]>([]);
  const [searchOffset, setSearchOffset] = useState(0);
  const [searchHasMore, setSearchHasMore] = useState(false);
  const [includeDependencies, setIncludeDependencies] = useState(true);
  const [actionKey, setActionKey] = useState<string | null>(null);
  const [isInstallOpen, setIsInstallOpen] = useState(false);
  const [isInstallSummaryOpen, setIsInstallSummaryOpen] = useState(false);
  const [loadingSummaryVersions, setLoadingSummaryVersions] = useState(false);
  const [loadingInstallPreview, setLoadingInstallPreview] = useState(false);
  const [installPreviewError, setInstallPreviewError] = useState<string | null>(
    null,
  );
  const [summaryVersionsError, setSummaryVersionsError] = useState<
    string | null
  >(null);
  const [previewDependencies, setPreviewDependencies] = useState<
    AddonInstallDependency[]
  >([]);
  const [selectedInstalls, setSelectedInstalls] = useState<
    Record<string, AddonSearchResult>
  >({});
  const [selectedVersionByKey, setSelectedVersionByKey] = useState<
    Record<string, string>
  >({});
  const [hydratedVersionKeyState, setHydratedVersionKeyState] = useState<
    Record<string, boolean>
  >({});
  const searchRequestRef = useRef(0);
  const lastBaseSearchKey = useRef('');
  const searchResultsRef = useRef<HTMLDivElement | null>(null);
  const hydratedVersionKeys = useRef<Set<string>>(null!);
  const installPreviewRequestRef = useRef(0);

  if (hydratedVersionKeys.current === null) {
    hydratedVersionKeys.current = new Set();
  }

  const isStopped = server.status === 'STOPPED';

  const loadAddons = useCallback(
    async (useSync = false) => {
      setLoading(true);
      setError(null);
      try {
        const response = useSync
          ? await api.syncAddons(server.id)
          : await api.listAddons(server.id);
        const data = response.data as AddonListResponse;
        setItems(data.items || []);
        setAddonType(data.addonType || 'mod');
      } catch (err) {
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('Failed to load addons');
        }
      } finally {
        setLoading(false);
      }
    },
    [server.id],
  );

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadAddons(false);
  }, [loadAddons]);

  const runAction = async (key: string, action: () => Promise<unknown>) => {
    setActionKey(key);
    setError(null);
    try {
      await action();
      await loadAddons(true);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Action failed');
      }
    } finally {
      setActionKey(null);
    }
  };

  const handleSearch = useCallback(
    async (overrides?: {
      query?: string;
      source?: 'modrinth' | 'curseforge';
      offset?: number;
      append?: boolean;
    }) => {
      const query =
        overrides?.query !== undefined
          ? overrides.query.trim()
          : searchQuery.trim();
      const source = overrides?.source ?? searchSource;
      const offset = overrides?.offset ?? 0;
      const append = overrides?.append ?? false;
      const requestId = ++searchRequestRef.current;
      if (!append) {
        lastBaseSearchKey.current = `${source}\n${query}`;
      }

      if (append) {
        setSearchingMore(true);
      } else {
        setSearching(true);
      }
      setError(null);
      try {
        const response = await api.searchAddons(server.id, {
          query,
          source,
          offset,
          limit: SEARCH_BATCH_SIZE,
        });
        if (searchRequestRef.current != requestId) {
          return;
        }
        const nextItems = response.data.items || [];
        if (append) {
          setSearchResults((prev) => {
            const map = new Map<string, AddonSearchResult>();
            for (const item of prev) {
              map.set(`${item.source}-${item.projectId}`, item);
            }
            for (const item of nextItems) {
              map.set(`${item.source}-${item.projectId}`, item);
            }
            return Array.from(map.values());
          });
        } else {
          setSearchResults(nextItems);
        }
        setSearchOffset(response.data.nextOffset || offset + nextItems.length);
        setSearchHasMore(Boolean(response.data.hasMore));
      } catch (err) {
        if (
          err &&
          typeof err === 'object' &&
          'response' in err &&
          (err as AxiosError<string>).response?.data
        ) {
          setError(String((err as AxiosError<string>).response?.data));
        } else if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('Search failed');
        }
      } finally {
        if (searchRequestRef.current === requestId) {
          if (append) {
            setSearchingMore(false);
          } else {
            setSearching(false);
          }
        }
      }
    },
    [searchQuery, searchSource, server.id, SEARCH_BATCH_SIZE],
  );

  const openInstallModal = () => {
    setSearchSource('modrinth');
    setSearchQuery('');
    setSearchResults([]);
    setSearchOffset(0);
    setSearchHasMore(false);
    setSelectedInstalls({});
    setSelectedVersionByKey({});
    setHydratedVersionKeyState({});
    setPreviewDependencies([]);
    setInstallPreviewError(null);
    setSummaryVersionsError(null);
    hydratedVersionKeys.current.clear();
    setIsInstallOpen(true);
    void handleSearch({ query: '', source: 'modrinth' });
  };

  const handleManualSearch = () => {
    const query = searchQuery.trim();
    setSearchOffset(0);
    setSearchHasMore(false);
    void handleSearch({
      query,
      source: searchSource,
    });
  };

  useEffect(() => {
    if (!isInstallOpen) {
      return;
    }
    const query = searchQuery.trim();
    const searchKey = `${searchSource}\n${query}`;
    if (searchKey === lastBaseSearchKey.current) {
      return;
    }
    const timeout = window.setTimeout(() => {
      setSearchOffset(0);
      setSearchHasMore(false);
      void handleSearch({
        query,
        source: searchSource,
      });
    }, 350);
    return () => window.clearTimeout(timeout);
  }, [handleSearch, isInstallOpen, searchQuery, searchSource]);

  const loadMoreSearchResults = useCallback(() => {
    if (searching || searchingMore || !searchHasMore) {
      return;
    }
    void handleSearch({
      query: searchQuery.trim(),
      source: searchSource,
      offset: searchOffset,
      append: true,
    });
  }, [
    handleSearch,
    searchHasMore,
    searchOffset,
    searchQuery,
    searchSource,
    searching,
    searchingMore,
  ]);

  useEffect(() => {
    if (!isInstallOpen || searching || searchingMore || !searchHasMore) {
      return;
    }
    const resultsElement = searchResultsRef.current;
    if (!resultsElement) {
      return;
    }
    const hasScrollableOverflow =
      resultsElement.scrollHeight > resultsElement.clientHeight + 8;
    if (!hasScrollableOverflow) {
      loadMoreSearchResults();
    }
  }, [
    isInstallOpen,
    loadMoreSearchResults,
    searchHasMore,
    searchResults.length,
    searching,
    searchingMore,
  ]);

  const title = addonType === 'plugin' ? 'Plugins' : 'Mods';
  const addonLabel = addonType === 'plugin' ? 'plugin' : 'mod';

  const toggleAddon = async (addon: Addon) => {
    if (!addon.disabled) {
      const confirmed = await confirmAction(
        `Disable ${addonLabel}`,
        `Disable ${addon.projectName || addon.name}?`,
        'Disable',
      );
      if (!confirmed) return;
    }

    await runAction(`toggle-${addon.id}`, () =>
      api.setAddonDisabled(server.id, addon.id, {
        disabled: !addon.disabled,
      }),
    );
  };

  const deleteAddon = async (addon: Addon) => {
    const confirmed = await confirmAction(
      `Delete ${addonLabel}`,
      `Delete ${addon.projectName || addon.name}?`,
      'Delete',
    );
    if (!confirmed) return;

    await runAction(`delete-${addon.id}`, () =>
      api.deleteAddon(server.id, addon.id),
    );
  };

  const updateAddon = async (addon: Addon) => {
    const confirmed = await confirmAction(
      `Update ${addonLabel}`,
      `Update ${addon.projectName || addon.name} to ${addon.latest?.versionLabel || 'the latest version'}?`,
      'Update',
    );
    if (!confirmed) return;

    await runAction(`update-${addon.id}`, () =>
      api.updateAddon(server.id, addon.id, {
        includeDependencies,
      }),
    );
  };

  const updateAllAddons = async () => {
    const addonTitle = title.toLowerCase();
    const confirmed = await confirmAction(
      `Update all ${addonTitle}`,
      `Update all installed ${addonTitle} with available updates?`,
      'Update all',
    );
    if (!confirmed) return;

    await runAction('update-all', () =>
      api.updateAllAddons(server.id, {
        includeDependencies,
      }),
    );
  };

  const selectedInstallEntries = useMemo(
    () => Object.entries(selectedInstalls),
    [selectedInstalls],
  );
  const selectedInstallCount = selectedInstallEntries.length;
  const selectedInstallKey = selectedInstallEntries
    .map(([key]) => key)
    .sort((a, b) => a.localeCompare(b))
    .join('|');
  const installedProjectKeys = useMemo(() => {
    const keys = new Set<string>();
    for (const addon of items) {
      if (!addon.projectId) continue;
      keys.add(`${addon.source}-${addon.projectId}`);
    }
    return keys;
  }, [items]);
  const resolveChosenVersion = useCallback(
    (key: string, result: AddonSearchResult) => {
      const versionId = selectedVersionByKey[key];
      if (!versionId) return result.latest || null;
      return (
        result.versions?.find((version) => version.versionId === versionId) ||
        result.latest ||
        null
      );
    },
    [selectedVersionByKey],
  );
  const selectedVersionSignature = Object.entries(selectedInstalls)
    .map(
      ([key, result]) =>
        `${key}:${selectedVersionByKey[key] || result.latest?.versionId || ''}`,
    )
    .sort((a, b) => a.localeCompare(b))
    .join('|');
  const summaryVersionsReady = selectedInstallEntries.every(
    ([key]) => hydratedVersionKeyState[key] === true,
  );
  const selectedVersionsReady =
    summaryVersionsReady &&
    selectedInstallEntries.every(([key, result]) =>
      Boolean(resolveChosenVersion(key, result)),
    );
  const installDependencyError = installPreviewError || summaryVersionsError;

  useEffect(() => {
    if (!isInstallSummaryOpen || selectedInstallKey === '') {
      return;
    }
    const entriesToHydrate = Object.entries(selectedInstalls).filter(
      ([key]) => !hydratedVersionKeys.current.has(key),
    );
    if (entriesToHydrate.length === 0) {
      return;
    }

    let cancelled = false;
    void Promise.all(
      entriesToHydrate.map(async ([key, result]) => {
        const response = await api.getAddonVersions(server.id, {
          source: result.source as Exclude<AddonSource, 'manual'>,
          projectId: result.projectId,
        });
        return {
          key,
          versions: response.data.versions || [],
        };
      }),
    )
      .then((updates) => {
        if (cancelled) return;
        for (const update of updates) {
          hydratedVersionKeys.current.add(update.key);
        }
        setSelectedInstalls((prev) => {
          const next = { ...prev };
          for (const update of updates) {
            const current = next[update.key];
            if (!current) continue;
            next[update.key] = {
              ...current,
              latest: update.versions[0] || current.latest,
              versions: update.versions,
            };
          }
          return next;
        });
        setSelectedVersionByKey((prev) => {
          const next = { ...prev };
          for (const update of updates) {
            if (!next[update.key] && update.versions[0]) {
              next[update.key] = update.versions[0].versionId;
            }
          }
          return next;
        });
        setHydratedVersionKeyState((prev) => {
          const next = { ...prev };
          for (const update of updates) {
            next[update.key] = true;
          }
          return next;
        });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        if (err instanceof Error) {
          setError(err.message);
          setSummaryVersionsError(err.message);
        } else {
          setError('Failed to load versions');
          setSummaryVersionsError('Failed to load compatible versions');
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingSummaryVersions(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [isInstallSummaryOpen, selectedInstallKey, selectedInstalls, server.id]);

  useEffect(() => {
    const requestId = ++installPreviewRequestRef.current;
    // This effect resets transient preview state when the selected roots or
    // the dependency checkbox changes. The async request below is still
    // cancelled and guarded by requestId before applying results.
    /* eslint-disable react-hooks/set-state-in-effect */
    if (
      !isInstallSummaryOpen ||
      !includeDependencies ||
      selectedInstallKey === ''
    ) {
      setLoadingInstallPreview(false);
      setInstallPreviewError(null);
      setPreviewDependencies([]);
      return;
    }
    if (!summaryVersionsReady || loadingSummaryVersions) {
      setLoadingInstallPreview(false);
      setInstallPreviewError(null);
      setPreviewDependencies([]);
      return;
    }

    let cancelled = false;
    setLoadingInstallPreview(true);
    setInstallPreviewError(null);
    setPreviewDependencies([]);

    const previewRequests = Object.entries(selectedInstalls).map(
      async ([key, result]) => {
        if (result.source !== 'modrinth' && result.source !== 'curseforge') {
          throw new Error(`Unsupported addon source: ${result.source}`);
        }
        const version = resolveChosenVersion(key, result);
        if (!version) {
          throw new Error(
            `No compatible version found for ${result.projectName}`,
          );
        }
        const response = await api.previewAddonInstall(server.id, {
          source: result.source,
          projectId: result.projectId,
          versionId:
            result.source === 'modrinth' ? version.versionId : undefined,
          fileId: result.source === 'curseforge' ? version.fileId : undefined,
        });
        return response.data.dependencies || [];
      },
    );

    void Promise.all(previewRequests)
      .then((responses) => {
        if (cancelled || installPreviewRequestRef.current !== requestId) {
          return;
        }
        const selectedRootKeys = new Set(
          selectedInstallEntries.map(
            ([, result]) => `${result.source}:${result.projectId}`,
          ),
        );
        const dependencyMap = new Map<string, AddonInstallDependency>();
        for (const dependencies of responses) {
          for (const dependency of dependencies) {
            const dependencyKey = `${dependency.source}:${dependency.projectId}`;
            if (selectedRootKeys.has(dependencyKey)) {
              continue;
            }
            dependencyMap.set(dependencyKey, dependency);
          }
        }
        setPreviewDependencies(Array.from(dependencyMap.values()));
      })
      .catch((err: unknown) => {
        if (cancelled || installPreviewRequestRef.current !== requestId) {
          return;
        }
        if (
          err &&
          typeof err === 'object' &&
          'response' in err &&
          (err as AxiosError<string>).response?.data
        ) {
          setInstallPreviewError(
            String((err as AxiosError<string>).response?.data),
          );
        } else if (err instanceof Error) {
          setInstallPreviewError(err.message);
        } else {
          setInstallPreviewError('Failed to preview dependencies');
        }
      })
      .finally(() => {
        if (!cancelled && installPreviewRequestRef.current === requestId) {
          setLoadingInstallPreview(false);
        }
      });

    return () => {
      cancelled = true;
    };
    /* eslint-enable react-hooks/set-state-in-effect */
  }, [
    includeDependencies,
    isInstallSummaryOpen,
    loadingSummaryVersions,
    selectedInstallKey,
    selectedInstallEntries,
    selectedInstalls,
    selectedVersionSignature,
    resolveChosenVersion,
    server.id,
    summaryVersionsReady,
  ]);

  return (
    <div className="tw:grid tw:min-w-0 tw:gap-4">
      <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-[14px]">
        <div className="tw:mb-4 tw:flex tw:items-center tw:gap-3 tw:[&_h3]:m-0 tw:[&_h3]:text-[1.1rem] tw:[&_p]:mt-0.5 tw:[&_p]:mb-0 tw:[&_p]:text-[0.9rem] tw:[&_p]:text-text-muted">
          <div className="tw:flex tw:h-10 tw:w-10 tw:items-center tw:justify-center tw:rounded-xl tw:border tw:border-border tw:bg-white/4">
            <Upload size={18} />
          </div>
          <div>
            <h3>{title} Manager</h3>
            <p>Install, update and remove {title.toLowerCase()}.</p>
          </div>
        </div>

        {!isStopped && (
          <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
            Stop the server to install, remove or update {title.toLowerCase()}.
          </p>
        )}

        {!canManage && (
          <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
            You need console permission to manage {title.toLowerCase()}.
          </p>
        )}

        <div className="tw:mt-3 tw:flex tw:flex-wrap tw:items-center tw:gap-2.5">
          <Button
            variant="secondary"
            onClick={() => loadAddons(true)}
            disabled={loading || actionKey !== null}
          >
            {loading ? (
              <Loader2 size={14} className="tw:animate-spin" />
            ) : (
              <RefreshCw size={14} />
            )}
            Sync
          </Button>
          <label className="tw:flex tw:items-center tw:gap-2 tw:text-[0.9rem] tw:text-text-muted">
            <input
              type="checkbox"
              className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:flex tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:items-center tw:before:justify-center tw:before:leading-none tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
              checked={includeDependencies}
              onChange={(e) => setIncludeDependencies(e.target.checked)}
            />{' '}
            Include dependencies
          </label>
          <Button
            variant="secondary"
            onClick={() => void updateAllAddons()}
            disabled={!canManage || !isStopped || actionKey !== null}
          >
            {actionKey === 'update-all' ? (
              <Loader2 size={14} className="tw:animate-spin" />
            ) : (
              <Download size={14} />
            )}
            Update all
          </Button>
          <Button
            onClick={openInstallModal}
            disabled={!canManage || !isStopped || actionKey !== null}
          >
            Install {title}
          </Button>
        </div>

        {error && <p className="tw:mt-2 tw:text-red-400">{error}</p>}
      </div>

      <Modal
        isOpen={isInstallOpen}
        onClose={() => setIsInstallOpen(false)}
        title={`Install ${title}`}
        contentClassName="tw:max-w-[840px] tw:max-h-[78vh]"
      >
        <div className="tw:mt-[14px] tw:grid tw:grid-cols-[1fr_180px_auto] tw:items-center tw:gap-2.5 tw:[&_label]:flex tw:[&_label]:items-center tw:[&_label]:gap-2">
          <label>
            <Search size={16} />
            <input
              type="text"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  handleManualSearch();
                }
              }}
              placeholder={`Search ${title.toLowerCase()}...`}
            />
          </label>
          <select
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={searchSource}
            onChange={(e) => {
              const nextSource = e.target.value as 'modrinth' | 'curseforge';
              setSearchSource(nextSource);
              setSearchOffset(0);
              setSearchHasMore(false);
              if (searchQuery.trim() === '') {
                void handleSearch({
                  query: '',
                  source: nextSource,
                });
              }
            }}
          >
            <option value="modrinth">Modrinth</option>
            <option value="curseforge">CurseForge</option>
          </select>
          {searching && <Loader2 size={18} className="tw:animate-spin" />}
        </div>

        <div
          ref={searchResultsRef}
          className="tw:mt-[14px] tw:max-h-[42vh] tw:overflow-y-auto tw:[&_ul]:mt-2.5 tw:[&_ul]:flex tw:[&_ul]:list-none tw:[&_ul]:flex-col tw:[&_ul]:gap-2.5 tw:[&_ul]:p-0 tw:[&_li]:flex tw:[&_li]:cursor-pointer tw:[&_li]:items-center tw:[&_li]:justify-between tw:[&_li]:gap-3 tw:[&_li]:rounded-[10px] tw:[&_li]:border tw:[&_li]:border-border tw:[&_li]:px-3 tw:[&_li]:py-2.5 tw:[&_li.selected]:border-[#6a7cff] tw:[&_li.selected]:bg-[#6a7cff]/14 tw:[&_li.installed]:opacity-65"
          onScroll={(event) => {
            const target = event.currentTarget;
            const threshold = 64;
            if (
              target.scrollTop + target.clientHeight >=
              target.scrollHeight - threshold
            ) {
              loadMoreSearchResults();
            }
          }}
        >
          <ul>
            {searchResults.map((result) => {
              const resultKey = `${result.source}-${result.projectId}`;
              const isInstalled = installedProjectKeys.has(resultKey);
              let resultDescription =
                result.projectSlug || result.description || '';
              if (isInstalled) {
                resultDescription = 'Already installed';
              } else if (result.authorName?.trim()) {
                resultDescription = `by ${result.authorName}`;
              }

              return (
                <li
                  key={resultKey}
                  className={isInstalled ? 'tw:opacity-65' : undefined}
                >
                  <label className="tw:grid tw:w-full tw:grid-cols-[auto_28px_1fr] tw:items-center tw:gap-2.5">
                    <input
                      type="checkbox"
                      className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:flex tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:items-center tw:before:justify-center tw:before:leading-none tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
                      disabled={isInstalled}
                      checked={Boolean(selectedInstalls[resultKey])}
                      onChange={(e) => {
                        const key = resultKey;
                        if (isInstalled) {
                          return;
                        }
                        if (e.target.checked) {
                          setSelectedInstalls((prev) => ({
                            ...prev,
                            [key]: result,
                          }));
                          setSelectedVersionByKey((prev) => ({
                            ...prev,
                            [key]: result.latest?.versionId || '',
                          }));
                          return;
                        }
                        setSelectedInstalls((prev) => {
                          const next = { ...prev };
                          delete next[key];
                          hydratedVersionKeys.current.delete(key);
                          return next;
                        });
                        setHydratedVersionKeyState((prev) => {
                          const next = { ...prev };
                          delete next[key];
                          return next;
                        });
                        setSelectedVersionByKey((prev) => {
                          const next = { ...prev };
                          delete next[key];
                          return next;
                        });
                      }}
                    />
                    {result.iconUrl ? (
                      <img
                        src={result.iconUrl}
                        alt={`${result.projectName} icon`}
                        className="tw:h-7 tw:w-7 tw:rounded-md tw:object-cover"
                      />
                    ) : (
                      <div className="tw:h-7 tw:w-7 tw:rounded-md tw:bg-white/8" />
                    )}
                    <div className="tw:min-w-0 tw:[&_strong]:block tw:[&_small]:block">
                      <strong>{result.projectName}</strong>
                      <small>{resultDescription}</small>
                    </div>
                  </label>
                  {result.projectUrl && (
                    <a
                      href={result.projectUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="tw:inline-flex tw:h-[30px] tw:w-[30px] tw:shrink-0 tw:items-center tw:justify-center tw:rounded-lg tw:border tw:border-border tw:bg-transparent tw:text-text-muted tw:hover:border-[#6a7cff] tw:hover:text-text-main"
                      aria-label={`Open ${result.projectName} page`}
                    >
                      <Globe size={14} />
                    </a>
                  )}
                </li>
              );
            })}
          </ul>

          {searchingMore && (
            <div className="tw:mt-2.5 tw:flex tw:items-center tw:justify-between tw:gap-2.5 tw:[&_span]:text-[0.9rem] tw:[&_span]:text-text-muted">
              <span>Loading more...</span>
            </div>
          )}
          {!searchingMore && !searchHasMore && searchResults.length > 0 && (
            <div className="tw:mt-2.5 tw:flex tw:items-center tw:justify-between tw:gap-2.5 tw:[&_span]:text-[0.9rem] tw:[&_span]:text-text-muted">
              <span>End of results</span>
            </div>
          )}
        </div>

        <div className="tw:mt-3 tw:grid tw:grid-cols-[1fr_auto] tw:gap-2.5">
          <Button
            variant="secondary"
            onClick={() => {
              const needsVersionHydration = selectedInstallEntries.some(
                ([key]) => !hydratedVersionKeys.current.has(key),
              );
              setLoadingSummaryVersions(needsVersionHydration);
              setInstallPreviewError(null);
              setSummaryVersionsError(null);
              setIsInstallSummaryOpen(true);
            }}
            disabled={
              selectedInstallCount === 0 ||
              actionKey !== null ||
              !canManage ||
              !isStopped
            }
          >
            Install ({selectedInstallCount})
          </Button>
        </div>
      </Modal>

      <div className="tw:box-border tw:min-w-0 tw:rounded-[14px] tw:border tw:border-border tw:bg-bg-card tw:p-[14px]">
        <h3>Installed {title}</h3>
        {loading && <p>Loading...</p>}
        {!loading && items.length === 0 && (
          <p>No {title.toLowerCase()} found.</p>
        )}
        {!loading && items.length > 0 && (
          <ul className="tw:mt-2.5 tw:flex tw:list-none tw:flex-col tw:gap-2.5 tw:p-0 tw:[&_li]:flex tw:[&_li]:items-center tw:[&_li]:justify-between tw:[&_li]:gap-3 tw:[&_li]:rounded-[10px] tw:[&_li]:border tw:[&_li]:border-border tw:[&_li]:px-3 tw:[&_li]:py-2.5 tw:[&_li.disabled]:opacity-65 tw:[&_small]:block tw:[&_small]:text-text-muted">
            {items.map((addon) => {
              const canUpdate =
                addon.status === 'update_available' && !addon.disabled;
              return (
                <li
                  key={addon.id}
                  className={addon.disabled ? 'tw:opacity-65' : undefined}
                >
                  <div className="tw:flex tw:min-w-0 tw:items-center tw:gap-2.5 tw:[&_div]:min-w-0 tw:[&_strong]:[overflow-wrap:anywhere] tw:[&_small]:[overflow-wrap:anywhere]">
                    {addon.iconUrl ? (
                      <img
                        src={addon.iconUrl}
                        alt={`${addon.projectName || addon.name} icon`}
                        className="tw:h-7 tw:w-7 tw:rounded-md tw:object-cover"
                      />
                    ) : (
                      <div className="tw:h-7 tw:w-7 tw:rounded-md tw:bg-white/8" />
                    )}
                    <div>
                      <strong>{addon.projectName || addon.name}</strong>
                      <small>
                        {addon.source} • {addon.fileName}
                        {addon.versionLabel ? ` • ${addon.versionLabel}` : ''}
                      </small>
                      {addon.disabled && <small>Disabled</small>}
                      {canUpdate && addon.latest && (
                        <small>
                          Update available: {addon.latest.versionLabel}
                        </small>
                      )}
                    </div>
                  </div>
                  <div className="tw:flex tw:shrink-0 tw:gap-2">
                    <Button
                      variant="secondary"
                      onClick={() => void toggleAddon(addon)}
                      disabled={!canManage || !isStopped || actionKey !== null}
                      aria-label={`${addon.disabled ? 'Enable' : 'Disable'} ${addon.projectName || addon.name}`}
                      title={addon.disabled ? 'Enable' : 'Disable'}
                    >
                      {actionKey === `toggle-${addon.id}` ? (
                        <Loader2 size={14} className="tw:animate-spin" />
                      ) : (
                        <Power size={14} />
                      )}
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={() => void updateAddon(addon)}
                      disabled={
                        !canManage ||
                        !isStopped ||
                        !canUpdate ||
                        actionKey !== null
                      }
                      aria-label={`Update ${addon.projectName || addon.name}`}
                    >
                      {actionKey === `update-${addon.id}` ? (
                        <Loader2 size={14} className="tw:animate-spin" />
                      ) : (
                        <Download size={14} />
                      )}
                    </Button>
                    <Button
                      variant="danger"
                      onClick={() => void deleteAddon(addon)}
                      disabled={!canManage || !isStopped || actionKey !== null}
                      aria-label={`Delete ${addon.projectName || addon.name}`}
                    >
                      {actionKey === `delete-${addon.id}` ? (
                        <Loader2 size={14} className="tw:animate-spin" />
                      ) : (
                        <Trash2 size={14} />
                      )}
                    </Button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <Modal
        isOpen={isInstallSummaryOpen}
        onClose={() => setIsInstallSummaryOpen(false)}
        title={`Install Summary (${selectedInstallCount})`}
        contentClassName="tw:max-w-[760px] tw:max-h-[78vh]"
      >
        {loadingSummaryVersions && (
          <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
            Loading compatible versions...
          </p>
        )}
        <ul className="tw:m-0 tw:flex tw:list-none tw:flex-col tw:gap-2 tw:p-0 tw:[&_li]:rounded-lg tw:[&_li]:border tw:[&_li]:border-border tw:[&_li]:p-2 tw:[&_small]:block tw:[&_small]:text-text-muted">
          {selectedInstallEntries.map(([key, result]) => {
            const version = resolveChosenVersion(key, result);
            return (
              <li key={key}>
                <div className="tw:mb-2 tw:flex tw:items-center tw:gap-2.5">
                  {result.iconUrl ? (
                    <img
                      src={result.iconUrl}
                      alt={`${result.projectName} icon`}
                      className="tw:h-7 tw:w-7 tw:rounded-md tw:object-cover"
                    />
                  ) : (
                    <div className="tw:h-7 tw:w-7 tw:rounded-md tw:bg-white/8" />
                  )}
                  <div>
                    <strong>{result.projectName}</strong>
                    <small>
                      {result.source} • {version?.versionLabel || 'latest'}
                    </small>
                  </div>
                </div>
                <select
                  className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
                  value={selectedVersionByKey[key] || ''}
                  onChange={(e) => {
                    setSummaryVersionsError(null);
                    setSelectedVersionByKey((prev) => ({
                      ...prev,
                      [key]: e.target.value,
                    }));
                  }}
                >
                  {(result.versions || []).length === 0 && (
                    <option value="">
                      {result.latest ? 'Latest' : 'No versions'}
                    </option>
                  )}
                  {(result.versions || []).map((option) => (
                    <option key={option.versionId} value={option.versionId}>
                      {option.versionLabel} ({option.releaseType})
                    </option>
                  ))}
                </select>
              </li>
            );
          })}
        </ul>
        {includeDependencies && (
          <section className="tw:mt-4 tw:border-t tw:border-border tw:pt-[14px]">
            <div className="tw:mb-2">
              <strong>
                Required dependencies ({previewDependencies.length})
              </strong>
            </div>
            {!installDependencyError &&
              (loadingSummaryVersions ||
                loadingInstallPreview ||
                !summaryVersionsReady) && (
                <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
                  Checking required dependencies...
                </p>
              )}
            {installDependencyError && (
              <p className="tw:m-0 tw:text-red-200">{installDependencyError}</p>
            )}
            {!installDependencyError &&
              !loadingSummaryVersions &&
              !loadingInstallPreview &&
              summaryVersionsReady &&
              previewDependencies.length === 0 && (
                <p className="tw:col-span-full tw:m-0 tw:text-[0.85rem] tw:text-text-muted">
                  All required dependencies are already installed.
                </p>
              )}
            {previewDependencies.length > 0 && !installDependencyError && (
              <ul className="tw:m-0 tw:flex tw:list-none tw:flex-col tw:gap-1.5 tw:p-0">
                {previewDependencies.map((dependency) => (
                  <li
                    key={`${dependency.source}:${dependency.projectId}`}
                    className="tw:flex tw:items-center tw:gap-2.5 tw:rounded-lg tw:border tw:border-border tw:bg-white/2 tw:px-2.5 tw:py-2 tw:[&_div]:min-w-0 tw:[&_small]:block tw:[&_small]:text-text-muted tw:[&_small]:[overflow-wrap:anywhere]"
                  >
                    {dependency.iconUrl ? (
                      <img
                        src={dependency.iconUrl}
                        alt={`${dependency.name || dependency.projectId} icon`}
                        className="tw:h-7 tw:w-7 tw:rounded-md tw:object-cover"
                      />
                    ) : (
                      <div className="tw:h-7 tw:w-7 tw:rounded-md tw:bg-white/8" />
                    )}
                    <div>
                      <strong>{dependency.name || dependency.projectId}</strong>
                      <small>
                        {dependency.source} •{' '}
                        {dependency.versionLabel || 'compatible version'}
                      </small>
                      {dependency.filename && (
                        <small>{dependency.filename}</small>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}
        <div className="tw:mt-3 tw:grid tw:grid-cols-[1fr_auto] tw:gap-2.5">
          <Button
            variant="secondary"
            onClick={() => setIsInstallSummaryOpen(false)}
          >
            Back
          </Button>
          <Button
            onClick={() => {
              const bulkInstallKey = `install-bulk-${Date.now()}`;
              void runAction(bulkInstallKey, async () => {
                for (const [key, result] of selectedInstallEntries) {
                  const version = resolveChosenVersion(key, result);
                  if (!version) continue;
                  await api.installAddon(server.id, {
                    source: result.source as Exclude<AddonSource, 'manual'>,
                    projectId: result.projectId,
                    versionId:
                      result.source === 'modrinth'
                        ? version.versionId
                        : undefined,
                    fileId:
                      result.source === 'curseforge'
                        ? version.fileId
                        : undefined,
                    includeDependencies,
                  });
                }
                setIsInstallSummaryOpen(false);
                setIsInstallOpen(false);
                setSelectedInstalls({});
                setSelectedVersionByKey({});
              });
            }}
            disabled={
              selectedInstallCount === 0 ||
              actionKey !== null ||
              !selectedVersionsReady ||
              (includeDependencies &&
                (loadingInstallPreview || Boolean(installDependencyError)))
            }
          >
            Install selected
          </Button>
        </div>
      </Modal>
    </div>
  );
};

export default AddonsPanel;
