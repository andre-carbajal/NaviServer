import type { AxiosError } from 'axios';
import {
  Download,
  Globe,
  Loader2,
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

import { api } from '../services/api';
import type {
  Addon,
  AddonListResponse,
  AddonSearchResult,
  AddonSource,
  Server,
} from '../types';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface AddonsPanelProps {
  server: Server;
  canManage: boolean;
}

const AddonsPanel: React.FC<AddonsPanelProps> = ({ server, canManage }) => {
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
  const [selectedInstalls, setSelectedInstalls] = useState<
    Record<string, AddonSearchResult>
  >({});
  const [selectedVersionByKey, setSelectedVersionByKey] = useState<
    Record<string, string>
  >({});
  const wasInstallOpen = useRef(false);
  const searchRequestRef = useRef(0);
  const lastBaseSearchKey = useRef('');
  const searchResultsRef = useRef<HTMLDivElement | null>(null);
  const hydratedVersionKeys = useRef(new Set<string>());

  const isStopped = server.status === 'STOPPED';

  const loadAddons = useCallback(
    async (useSync = true) => {
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
    loadAddons(true);
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
        if (searchRequestRef.current !== requestId) {
          return;
        }
        if (append) {
          setSearchingMore(false);
        } else {
          setSearching(false);
        }
      }
    },
    [searchQuery, searchSource, server.id, SEARCH_BATCH_SIZE],
  );

  useEffect(() => {
    if (isInstallOpen && !wasInstallOpen.current) {
      setSearchSource('modrinth');
      setSearchQuery('');
      setSearchResults([]);
      setSearchOffset(0);
      setSearchHasMore(false);
      setSelectedInstalls({});
      setSelectedVersionByKey({});
      hydratedVersionKeys.current.clear();
      void handleSearch({ query: '', source: 'modrinth' });
    }
    wasInstallOpen.current = isInstallOpen;
  }, [handleSearch, isInstallOpen]);

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

  const selectedInstallEntries = Object.entries(selectedInstalls);
  const selectedInstallCount = selectedInstallEntries.length;
  const selectedInstallKey = selectedInstallEntries
    .map(([key]) => key)
    .sort()
    .join('|');
  const installedProjectKeys = useMemo(() => {
    const keys = new Set<string>();
    for (const addon of items) {
      if (!addon.projectId) continue;
      keys.add(`${addon.source}-${addon.projectId}`);
    }
    return keys;
  }, [items]);
  const resolveChosenVersion = (key: string, result: AddonSearchResult) => {
    const versionId = selectedVersionByKey[key];
    if (!versionId) return result.latest || null;
    return (
      result.versions?.find((version) => version.versionId === versionId) ||
      result.latest ||
      null
    );
  };

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
    setLoadingSummaryVersions(true);
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
            hydratedVersionKeys.current.add(update.key);
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
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('Failed to load versions');
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

  return (
    <div className="server-v2-addons-layout">
      <div className="server-v2-settings-card">
        <div className="server-v2-settings-panel-head">
          <div className="server-v2-settings-panel-icon">
            <Upload size={18} />
          </div>
          <div>
            <h3>{title} Manager</h3>
            <p>Install, update and remove {title.toLowerCase()}.</p>
          </div>
        </div>

        {!isStopped && (
          <p className="server-v2-settings-hint">
            Stop the server to install, remove or update {title.toLowerCase()}.
          </p>
        )}

        {!canManage && (
          <p className="server-v2-settings-hint">
            You need console permission to manage {title.toLowerCase()}.
          </p>
        )}

        <div className="server-v2-addons-toolbar">
          <Button
            variant="secondary"
            onClick={() => loadAddons(true)}
            disabled={loading || actionKey !== null}
          >
            {loading ? (
              <Loader2 size={14} className="spin" />
            ) : (
              <RefreshCw size={14} />
            )}
            Sync
          </Button>
          <label className="server-v2-addons-toggle">
            <input
              type="checkbox"
              checked={includeDependencies}
              onChange={(e) => setIncludeDependencies(e.target.checked)}
            />
            Include dependencies
          </label>
          <Button
            variant="secondary"
            onClick={() =>
              runAction('update-all', () =>
                api.updateAllAddons(server.id, {
                  includeDependencies,
                }),
              )
            }
            disabled={!canManage || !isStopped || actionKey !== null}
          >
            {actionKey === 'update-all' ? (
              <Loader2 size={14} className="spin" />
            ) : (
              <Download size={14} />
            )}
            Update all
          </Button>
          <Button
            onClick={() => setIsInstallOpen(true)}
            disabled={!canManage || !isStopped || actionKey !== null}
          >
            Install {title}
          </Button>
        </div>

        {error && <p className="server-v2-addons-error">{error}</p>}
      </div>

      <Modal
        isOpen={isInstallOpen}
        onClose={() => setIsInstallOpen(false)}
        title={`Install ${title}`}
        contentClassName="modal-content-install"
      >
        <div className="server-v2-addons-search">
          <label>
            <Search size={16} />
            <input
              type="text"
              className="form-input"
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
            className="form-input"
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
          {searching && <Loader2 size={18} className="spin" />}
        </div>

        <div
          ref={searchResultsRef}
          className="server-v2-addons-results"
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
            {searchResults.map((result) => (
              <li
                key={`${result.source}-${result.projectId}`}
                className={
                  installedProjectKeys.has(
                    `${result.source}-${result.projectId}`,
                  )
                    ? 'installed'
                    : undefined
                }
              >
                <label className="server-v2-install-row">
                  <input
                    type="checkbox"
                    disabled={installedProjectKeys.has(
                      `${result.source}-${result.projectId}`,
                    )}
                    checked={Boolean(
                      selectedInstalls[`${result.source}-${result.projectId}`],
                    )}
                    onChange={(e) => {
                      const key = `${result.source}-${result.projectId}`;
                      if (installedProjectKeys.has(key)) {
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
                      className="server-v2-install-icon"
                    />
                  ) : (
                    <div className="server-v2-install-icon-placeholder" />
                  )}
                  <div className="server-v2-install-text">
                    <strong>{result.projectName}</strong>
                    <small>
                      {installedProjectKeys.has(
                        `${result.source}-${result.projectId}`,
                      )
                        ? 'Already installed'
                        : result.authorName?.trim()
                          ? `by ${result.authorName}`
                          : result.projectSlug || result.description || ''}
                    </small>
                  </div>
                </label>
                {result.projectUrl && (
                  <a
                    href={result.projectUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="server-v2-addon-link"
                    aria-label={`Open ${result.projectName} page`}
                  >
                    <Globe size={14} />
                  </a>
                )}
              </li>
            ))}
          </ul>

          {searchingMore && (
            <div className="server-v2-addons-pagination">
              <span>Loading more...</span>
            </div>
          )}
          {!searchingMore && !searchHasMore && searchResults.length > 0 && (
            <div className="server-v2-addons-pagination">
              <span>End of results</span>
            </div>
          )}
        </div>

        <div className="server-v2-addons-install-footer">
          <Button
            variant="secondary"
            onClick={() => setIsInstallSummaryOpen(true)}
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

      <div className="server-v2-settings-card">
        <h3>Installed {title}</h3>
        {loading ? (
          <p>Loading...</p>
        ) : items.length === 0 ? (
          <p>No {title.toLowerCase()} found.</p>
        ) : (
          <ul className="server-v2-addon-list">
            {items.map((addon) => {
              const canUpdate = addon.status === 'update_available';
              return (
                <li key={addon.id}>
                  <div className="server-v2-installed-addon-meta">
                    {addon.iconUrl ? (
                      <img
                        src={addon.iconUrl}
                        alt={`${addon.projectName || addon.name} icon`}
                        className="server-v2-install-icon"
                      />
                    ) : (
                      <div className="server-v2-install-icon-placeholder" />
                    )}
                    <div>
                      <strong>{addon.projectName || addon.name}</strong>
                      <small>
                        {addon.source} • {addon.fileName}
                        {addon.versionLabel ? ` • ${addon.versionLabel}` : ''}
                      </small>
                      {canUpdate && addon.latest && (
                        <small>
                          Update available: {addon.latest.versionLabel}
                        </small>
                      )}
                    </div>
                  </div>
                  <div className="server-v2-addon-actions">
                    <Button
                      variant="secondary"
                      onClick={() =>
                        runAction(`update-${addon.id}`, () =>
                          api.updateAddon(server.id, addon.id, {
                            includeDependencies,
                          }),
                        )
                      }
                      disabled={
                        !canManage ||
                        !isStopped ||
                        !canUpdate ||
                        actionKey !== null
                      }
                    >
                      {actionKey === `update-${addon.id}` ? (
                        <Loader2 size={14} className="spin" />
                      ) : (
                        <Download size={14} />
                      )}
                    </Button>
                    <Button
                      variant="danger"
                      onClick={() =>
                        runAction(`delete-${addon.id}`, () =>
                          api.deleteAddon(server.id, addon.id),
                        )
                      }
                      disabled={!canManage || !isStopped || actionKey !== null}
                    >
                      {actionKey === `delete-${addon.id}` ? (
                        <Loader2 size={14} className="spin" />
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
        contentClassName="modal-content-install-summary"
      >
        {loadingSummaryVersions && (
          <p className="server-v2-settings-hint">
            Loading compatible versions...
          </p>
        )}
        <ul className="server-v2-install-summary-list">
          {selectedInstallEntries.map(([key, result]) => {
            const version = resolveChosenVersion(key, result);
            return (
              <li key={key}>
                <div className="server-v2-install-summary-addon">
                  {result.iconUrl ? (
                    <img
                      src={result.iconUrl}
                      alt={`${result.projectName} icon`}
                      className="server-v2-install-icon"
                    />
                  ) : (
                    <div className="server-v2-install-icon-placeholder" />
                  )}
                  <div>
                    <strong>{result.projectName}</strong>
                    <small>
                      {result.source} • {version?.versionLabel || 'latest'}
                    </small>
                  </div>
                </div>
                <select
                  className="form-input"
                  value={selectedVersionByKey[key] || ''}
                  onChange={(e) =>
                    setSelectedVersionByKey((prev) => ({
                      ...prev,
                      [key]: e.target.value,
                    }))
                  }
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
        <div className="server-v2-addons-install-footer">
          <Button
            variant="secondary"
            onClick={() => setIsInstallSummaryOpen(false)}
          >
            Back
          </Button>
          <Button
            onClick={() => {
              void runAction(`install-bulk-${Date.now()}`, async () => {
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
            disabled={selectedInstallCount === 0 || actionKey !== null}
          >
            Install selected
          </Button>
        </div>
      </Modal>
    </div>
  );
};

export default AddonsPanel;
