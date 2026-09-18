import { v4 as uuidv4 } from 'uuid';

import React, { useEffect, useMemo, useState } from 'react';

import { api } from '../../services/api';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';

const loaderLogoMap: Record<string, string> = {
  paper: '/loaders/paper.webp',
  vanilla: '/loaders/vanilla.webp',
  fabric: '/loaders/fabric.webp',
  forge: '/loaders/forge.webp',
  neoforge: '/loaders/neoforge.webp',
};

interface CreateModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (data: {
    name: string;
    loader: string;
    version?: string;
    ram: number;
    requestId?: string;
    loaderOptions?: {
      mcVersion?: string;
      includeSnapshots?: boolean;
      includeUnstable?: boolean;
      buildVersion?: string;
      loaderVersion?: string;
    };
  }) => void;
}

const CreateModal: React.FC<CreateModalProps> = ({
  isOpen,
  onClose,
  onCreate,
}) => {
  const [name, setName] = useState('');
  const [loader, setLoader] = useState('vanilla');
  const [ram, setRam] = useState(2048);
  const [loaders, setLoaders] = useState<string[]>([]);
  const [mcVersion, setMcVersion] = useState('');
  const [includeSnapshots, setIncludeSnapshots] = useState(false);
  const [includeUnstable, setIncludeUnstable] = useState(false);
  const [buildVersion, setBuildVersion] = useState('');
  const [loaderVersion, setLoaderVersion] = useState('');
  const [metadata, setMetadata] = useState<{
    latestVersion?: string;
    minecraftVersions?: string[];
    buildVersions?: string[];
    loaderVersions?: string[];
  }>({});

  useEffect(() => {
    if (!isOpen) return;
    api.getLoaders().then((response) => {
      setLoaders(response.data);
      if (response.data.length > 0) setLoader(response.data[0]);
    });
  }, [isOpen]);

  useEffect(() => {
    const resetTimer = window.setTimeout(() => {
      setMcVersion('');
      setBuildVersion('');
      setLoaderVersion('');
      setIncludeSnapshots(false);
      setIncludeUnstable(false);
      setMetadata({});
    }, 0);

    return () => window.clearTimeout(resetTimer);
  }, [loader]);

  useEffect(() => {
    if (!loader || !isOpen) return;
    api
      .getLoaderMetadata(loader, {
        mcVersion,
        includeSnapshots,
        includeUnstable,
      })
      .then((response) => {
        const md = response.data;
        setMetadata(md);
        if (
          md.latestVersion &&
          (!mcVersion || !(md.minecraftVersions || []).includes(mcVersion))
        ) {
          setMcVersion(md.latestVersion);
        }
        if (
          md.buildVersions?.length &&
          (!buildVersion || !md.buildVersions.includes(buildVersion))
        ) {
          setBuildVersion(md.buildVersions[0]);
        }
        if (
          md.loaderVersions?.length &&
          (!loaderVersion || !md.loaderVersions.includes(loaderVersion))
        ) {
          setLoaderVersion(md.loaderVersions[0]);
        }
      });
  }, [
    loader,
    isOpen,
    mcVersion,
    includeSnapshots,
    includeUnstable,
    buildVersion,
    loaderVersion,
  ]);

  const showUnstableToggle = useMemo(
    () => ['fabric', 'neoforge'].includes(loader),
    [loader],
  );

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const newRequestId = uuidv4();
    onCreate({
      name,
      loader,
      ram,
      requestId: newRequestId,
      loaderOptions: {
        mcVersion,
        includeSnapshots,
        includeUnstable,
        buildVersion: loader === 'paper' ? buildVersion : undefined,
        loaderVersion: ['fabric', 'forge', 'neoforge'].includes(loader)
          ? loaderVersion
          : undefined,
      },
      version: mcVersion,
    });
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create New Server">
      <form onSubmit={handleSubmit}>
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="create-server-name"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            Server Name
          </label>
          <input
            id="create-server-name"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="create-server-loader"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            Loader
          </label>
          <div className="tw:flex tw:items-center tw:gap-3">
            <select
              id="create-server-loader"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={loader}
              onChange={(e) => setLoader(e.target.value)}
            >
              {loaders.map((l) => (
                <option key={l} value={l}>
                  {l}
                </option>
              ))}
            </select>
            <img
              src={loaderLogoMap[loader]}
              alt={`${loader} logo`}
              className="tw:h-8 tw:w-8"
            />
          </div>
        </div>
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="create-server-mc-version"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            Minecraft Version
          </label>
          <select
            id="create-server-mc-version"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={mcVersion}
            onChange={(e) => setMcVersion(e.target.value)}
          >
            {(metadata.minecraftVersions || []).map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
        </div>
        {loader === 'vanilla' && (
          <label className="tw:mb-2.5 tw:inline-flex tw:cursor-pointer tw:items-center tw:gap-2 tw:text-[0.9rem] tw:text-gray-200">
            <input
              type="checkbox"
              className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
              checked={includeSnapshots}
              onChange={(e) => setIncludeSnapshots(e.target.checked)}
            />{' '}
            Show snapshots
          </label>
        )}
        {showUnstableToggle && (
          <label className="tw:mb-2.5 tw:inline-flex tw:cursor-pointer tw:items-center tw:gap-2 tw:text-[0.9rem] tw:text-gray-200">
            <input
              type="checkbox"
              className="tw:grid tw:h-5 tw:w-5 tw:shrink-0 tw:cursor-pointer tw:appearance-none tw:place-content-center tw:rounded tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:p-0 tw:before:block tw:before:h-[0.65em] tw:before:w-[0.65em] tw:before:scale-0 tw:before:origin-center tw:before:content-['✓'] tw:before:text-xs tw:before:font-bold tw:before:text-white tw:checked:border-blue-500 tw:checked:bg-blue-500 tw:checked:before:scale-100 tw:disabled:cursor-not-allowed tw:disabled:opacity-60"
              checked={includeUnstable}
              onChange={(e) => setIncludeUnstable(e.target.checked)}
            />{' '}
            Show unstable
          </label>
        )}
        {loader === 'paper' && (
          <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
            <label
              htmlFor="create-server-build-version"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
            >
              Build version
            </label>
            <select
              id="create-server-build-version"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={buildVersion}
              onChange={(e) => setBuildVersion(e.target.value)}
            >
              {(metadata.buildVersions || []).map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </div>
        )}
        {['fabric', 'forge', 'neoforge'].includes(loader) && (
          <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
            <label
              htmlFor="create-server-loader-version"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
            >
              Loader version
            </label>
            <select
              id="create-server-loader-version"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={loaderVersion}
              onChange={(e) => setLoaderVersion(e.target.value)}
            >
              {(metadata.loaderVersions || []).map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          </div>
        )}
        <div className="tw:mb-[15px] tw:max-[769px]:mb-3">
          <label
            htmlFor="create-server-ram"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted tw:max-[769px]:mb-1.5 tw:max-[769px]:text-[0.85rem]"
          >
            RAM (MB)
          </label>
          <input
            id="create-server-ram"
            type="number"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={ram}
            onChange={(e) => setRam(Number(e.target.value))}
            min="1024"
            step="512"
          />
        </div>
        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit">Create Server</Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateModal;
