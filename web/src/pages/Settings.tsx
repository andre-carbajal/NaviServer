import React from 'react';

import { Button } from '../components/ui/Button';
import {
  BYTES_PER_LINE_ESTIMATE,
  useGlobalSettings,
} from '../hooks/useGlobalSettings';
import { useModalDialog } from '../hooks/useModalDialog';
import { humanSize } from '../utils/format';

const Settings: React.FC = () => {
  const { showAlert, showConfirm, modalDialog } = useModalDialog();
  const settings = useGlobalSettings(showAlert, showConfirm);
  const {
    loading,
    portRange,
    setPortRange,
    portsChanged: hasChanges,
    isSaving,
    publicIP,
    setPublicIP,
    initialPublicIP,
    networkInterfaces,
    publicIPWarning,
    isSavingPublicIP,
    curseForgeKey,
    setCurseForgeKey,
    curseForgeKeyStatus,
    isSavingCurseForgeKey,
    isClearingCurseForgeKey,
    logBufferSize,
    initialLogBufferSize,
    logBufferError,
    isSavingLogBuffer,
    isRestarting,
  } = settings;
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setPortRange({
      ...portRange,
      [e.target.name]: Number.parseInt(e.target.value, 10),
    });
  const estimatedBytes = logBufferSize * BYTES_PER_LINE_ESTIMATE;
  const handleLogBufferChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    settings.changeLogBuffer(e.target.value);
  const handleSave = settings.savePorts;
  const handleSaveLogBuffer = settings.saveLogBuffer;
  const handleSavePublicIP = settings.savePublicIP;
  const handleRestart = settings.restart;
  const handleSaveCurseForgeKey = settings.saveCurseForgeKey;
  const handleClearCurseForgeKey = settings.clearCurseForgeKey;

  if (loading) return <div>Loading settings...</div>;

  return (
    <div className="tw:flex tw:flex-col tw:gap-5">
      {modalDialog}
      <h1 className="tw:m-0">Settings</h1>

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2>Network Configuration</h2>
        <p>
          Define the range of ports that the manager can assign to new servers.
        </p>

        <div>
          <div className="tw:mb-[15px]">
            <label
              htmlFor="settings-start-port"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
            >
              Start Port
            </label>
            <input
              id="settings-start-port"
              type="number"
              name="start"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={portRange.start}
              onChange={handleChange}
            />
          </div>
          <div className="tw:mb-[15px]">
            <label
              htmlFor="settings-end-port"
              className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
            >
              End Port
            </label>
            <input
              id="settings-end-port"
              type="number"
              name="end"
              className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
              value={portRange.end}
              onChange={handleChange}
            />
          </div>
        </div>

        <div>
          <Button onClick={handleSave} disabled={!hasChanges || isSaving}>
            {isSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </div>
      </div>

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2>Public Address</h2>
        <p>
          Configure the IP address or hostname displayed for server connections.
          This address is used in the server list and public share links,
          allowing users to connect using a specific network interface (e.g.,
          VPN, public IP).
        </p>

        {publicIPWarning && (
          <div className="tw:mb-4 tw:rounded-md tw:bg-yellow-500/15 tw:px-3.5 tw:py-2.5 tw:text-[0.9rem] tw:text-yellow-500">
            ⚠️ {publicIPWarning}
          </div>
        )}

        <div className="tw:mb-[15px]">
          <label
            htmlFor="settings-public-address"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
          >
            Public Address
          </label>
          <select
            id="settings-public-address"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={publicIP}
            onChange={(e) => setPublicIP(e.target.value)}
          >
            <option value="localhost">localhost (default)</option>
            {networkInterfaces.map((ip) => (
              <option key={ip} value={ip}>
                {ip}
              </option>
            ))}
            {publicIP !== 'localhost' &&
              !networkInterfaces.includes(publicIP) && (
                <option value={publicIP}>{publicIP} (unavailable)</option>
              )}
          </select>
        </div>
        <div>
          <Button
            onClick={handleSavePublicIP}
            disabled={isSavingPublicIP || publicIP === initialPublicIP}
          >
            {isSavingPublicIP ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </div>

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2>CurseForge API</h2>
        <p>
          Configure an optional custom CurseForge API key. If set, it overrides
          the embedded build key. Modrinth remains the default source.
        </p>
        <div className="tw:mb-[15px]">
          <label
            htmlFor="settings-curseforge-key"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
          >
            Custom CurseForge API Key
          </label>
          <input
            id="settings-curseforge-key"
            type="password"
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={curseForgeKey}
            onChange={(e) => setCurseForgeKey(e.target.value)}
            placeholder={
              curseForgeKeyStatus.hasCustomKey
                ? '••••••••••••••••'
                : 'Enter your CurseForge API key'
            }
          />
        </div>
        <div className="tw:mb-3 tw:text-[0.9rem] tw:text-gray-400">
          <div>
            Embedded key available:{' '}
            {curseForgeKeyStatus.hasEmbeddedKey ? 'Yes' : 'No'}
          </div>
          <div>Effective source: {curseForgeKeyStatus.effectiveSource}</div>
        </div>
        <div className="tw:flex tw:flex-wrap tw:gap-2.5">
          <Button
            onClick={handleSaveCurseForgeKey}
            disabled={isSavingCurseForgeKey || !curseForgeKey.trim()}
          >
            {isSavingCurseForgeKey ? 'Saving...' : 'Save Custom Key'}
          </Button>
          <Button
            variant="secondary"
            onClick={handleClearCurseForgeKey}
            disabled={
              isClearingCurseForgeKey || !curseForgeKeyStatus.hasCustomKey
            }
          >
            {isClearingCurseForgeKey ? 'Clearing...' : 'Clear Custom Key'}
          </Button>
        </div>
      </div>

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2>Console Log Buffer</h2>
        <p>
          Define how many lines of console logs should be kept in memory per
          server while it is running.
        </p>
        <div className="tw:mb-[15px]">
          <label
            htmlFor="settings-log-buffer-size"
            className="tw:mb-2 tw:block tw:text-[0.9rem] tw:text-text-muted"
          >
            Lines to keep in memory{' '}
            <small className="tw:font-normal">(use 0 to disable)</small>
          </label>
          <input
            id="settings-log-buffer-size"
            type="number"
            min={0}
            step={1}
            className="tw:box-border tw:w-full tw:rounded-lg tw:border tw:border-[rgb(32,36,43)] tw:bg-bg-dark tw:px-4 tw:py-3 tw:text-[0.95rem] tw:text-gray-200 tw:outline-none tw:transition-all tw:duration-200 tw:ease-[ease] tw:focus:border-bg-dark tw:focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] tw:placeholder:text-gray-400 tw:max-[769px]:px-3 tw:max-[769px]:py-2.5 tw:max-[769px]:text-base"
            value={logBufferSize || ''}
            onChange={handleLogBufferChange}
          />
          {logBufferError && (
            <div className="tw:mt-1.5 tw:text-red-500">{logBufferError}</div>
          )}
          <div className="tw:mt-2 tw:text-[#555]">
            <strong>Estimated memory usage:</strong> {humanSize(estimatedBytes)}
            <div className="tw:mt-1 tw:text-xs">
              (Based on ~{BYTES_PER_LINE_ESTIMATE} bytes per line. This is an
              estimate and represents the memory used by the buffer in RAM while
              the server is running.)
            </div>
          </div>
        </div>
        <div>
          <Button
            onClick={handleSaveLogBuffer}
            disabled={
              isSavingLogBuffer ||
              logBufferSize === initialLogBufferSize ||
              !!logBufferError
            }
          >
            {isSavingLogBuffer ? 'Saving...' : 'Save'}
          </Button>
        </div>
      </div>

      <div className="tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-5">
        <h2>System</h2>
        <p>Manage the NaviServer Daemon process.</p>
        <div>
          <Button
            variant="danger"
            onClick={handleRestart}
            disabled={isRestarting}
          >
            {isRestarting ? 'Restarting...' : 'Restart Daemon'}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default Settings;
