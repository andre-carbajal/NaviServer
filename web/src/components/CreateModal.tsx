import { v4 as uuidv4 } from 'uuid';

import React, { useEffect, useMemo, useState } from 'react';

import { api } from '../services/api.ts';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

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

const CreateModal: React.FC<CreateModalProps> = ({ isOpen, onClose, onCreate }) => {
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
    setMcVersion('');
    setBuildVersion('');
    setLoaderVersion('');
    setIncludeSnapshots(false);
    setIncludeUnstable(false);
    setMetadata({});
  }, [loader]);

  useEffect(() => {
    if (!loader || !isOpen) return;
    api
      .getLoaderMetadata(loader, { mcVersion, includeSnapshots, includeUnstable })
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
  }, [loader, isOpen, mcVersion, includeSnapshots, includeUnstable]);

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
        <div className="form-group">
          <label>Server Name</label>
          <input className="form-input" value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Loader</label>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <select className="form-select" value={loader} onChange={(e) => setLoader(e.target.value)}>
              {loaders.map((l) => <option key={l} value={l}>{l}</option>)}
            </select>
            <img src={loaderLogoMap[loader]} alt={`${loader} logo`} style={{ width: 32, height: 32 }} />
          </div>
        </div>
        <div className="form-group">
          <label>Minecraft Version</label>
          <select className="form-select" value={mcVersion} onChange={(e) => setMcVersion(e.target.value)}>
            {(metadata.minecraftVersions || []).map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </div>
        {loader === 'vanilla' && (
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={includeSnapshots}
              onChange={(e) => setIncludeSnapshots(e.target.checked)}
            />{' '}
            Show snapshots
          </label>
        )}
        {showUnstableToggle && (
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={includeUnstable}
              onChange={(e) => setIncludeUnstable(e.target.checked)}
            />{' '}
            Show unstable
          </label>
        )}
        {loader === 'paper' && (
          <div className="form-group">
            <label>Build version</label>
            <select className="form-select" value={buildVersion} onChange={(e) => setBuildVersion(e.target.value)}>
              {(metadata.buildVersions || []).map((v) => <option key={v} value={v}>{v}</option>)}
            </select>
          </div>
        )}
        {['fabric', 'forge', 'neoforge'].includes(loader) && (
          <div className="form-group">
            <label>Loader version</label>
            <select className="form-select" value={loaderVersion} onChange={(e) => setLoaderVersion(e.target.value)}>
              {(metadata.loaderVersions || []).map((v) => <option key={v} value={v}>{v}</option>)}
            </select>
          </div>
        )}
        <div className="form-group">
          <label>RAM (MB)</label>
          <input type="number" className="form-input" value={ram} onChange={(e) => setRam(Number(e.target.value))} min="1024" step="512" />
        </div>
        <div className="modal-actions">
          <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
          <Button type="submit">Create Server</Button>
        </div>
      </form>
    </Modal>
  );
};

export default CreateModal;
