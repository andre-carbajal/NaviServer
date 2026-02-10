import React, { useEffect, useState } from 'react';

import { Button } from '../components/ui/Button';
import { api } from '../services/api';

const BYTES_PER_LINE_ESTIMATE = 200;

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(2)} KB`;
  const mb = kb / 1024;
  if (mb < 1024) return `${mb.toFixed(2)} MB`;
  const gb = mb / 1024;
  return `${gb.toFixed(2)} GB`;
}

const Settings: React.FC = () => {
  const [portRange, setPortRange] = useState({ start: 0, end: 0 });
  const [initialPortRange, setInitialPortRange] = useState({
    start: 0,
    end: 0,
  });
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [logBufferSize, setLogBufferSize] = useState(1000);
  const [initialLogBufferSize, setInitialLogBufferSize] = useState(1000);
  const [isSavingLogBuffer, setIsSavingLogBuffer] = useState(false);
  const [logBufferError, setLogBufferError] = useState<string | null>(null);
  const [publicIP, setPublicIP] = useState('localhost');
  const [initialPublicIP, setInitialPublicIP] = useState('localhost');
  const [networkInterfaces, setNetworkInterfaces] = useState<string[]>([]);
  const [isSavingPublicIP, setIsSavingPublicIP] = useState(false);
  const [publicIPWarning, setPublicIPWarning] = useState<string | null>(null);

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const res = await api.getPortRange();
        setPortRange(res.data);
        setInitialPortRange(res.data);
        const lb = await api.getLogBufferSize();
        const size = lb.data?.log_buffer_size ?? 1000;
        setLogBufferSize(size);
        setInitialLogBufferSize(size);

        const [publicIPRes, interfacesRes] = await Promise.all([
          api.getPublicIP(),
          api.getNetworkInterfaces(),
        ]);
        const savedIP = publicIPRes.data?.public_ip ?? 'localhost';
        setPublicIP(savedIP);
        setInitialPublicIP(savedIP);

        const ifaces = interfacesRes.data?.interfaces ?? [];
        setNetworkInterfaces(ifaces);

        if (savedIP !== 'localhost' && !ifaces.includes(savedIP)) {
          setPublicIPWarning(
            `The configured IP "${savedIP}" is not currently available on any network interface.`,
          );
        }
      } catch (err) {
        console.error('Failed to fetch settings:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchSettings();
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    const newPortRange = { ...portRange, [name]: parseInt(value, 10) };
    setPortRange(newPortRange);
    setHasChanges(
      JSON.stringify(newPortRange) !== JSON.stringify(initialPortRange),
    );
  };

  const handleLogBufferChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    const parsed = parseInt(raw, 10);
    if (raw === '') {
      setLogBufferSize(0);
      setLogBufferError(null);
      return;
    }
    if (isNaN(parsed)) {
      setLogBufferSize(0);
      setLogBufferError('The value must be an integer >= 0');
      return;
    }
    if (parsed < 0) {
      setLogBufferSize(parsed);
      setLogBufferError('The value cannot be negative');
      return;
    }
    setLogBufferSize(parsed);
    setLogBufferError(null);
  };

  const estimatedBytes = logBufferSize * BYTES_PER_LINE_ESTIMATE;

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.updatePortRange(portRange);
      setInitialPortRange(portRange);
      setHasChanges(false);
    } catch (err) {
      console.error('Failed to save settings:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleSaveLogBuffer = async () => {
    if (isNaN(logBufferSize) || logBufferSize < 0) {
      setLogBufferError('The value must be an integer >= 0');
      return;
    }
    setLogBufferError(null);
    setIsSavingLogBuffer(true);
    try {
      await api.updateLogBufferSize({ log_buffer_size: logBufferSize });
      setInitialLogBufferSize(logBufferSize);
    } catch (err) {
      console.error('Failed to save log buffer size:', err);
      setLogBufferError('The save operation failed. Please try again.');
    } finally {
      setIsSavingLogBuffer(false);
    }
  };

  const handleSavePublicIP = async () => {
    setIsSavingPublicIP(true);
    try {
      await api.updatePublicIP({ public_ip: publicIP });
      setInitialPublicIP(publicIP);
      if (publicIP !== 'localhost' && !networkInterfaces.includes(publicIP)) {
        setPublicIPWarning(
          `The configured IP "${publicIP}" is not currently available on any network interface.`,
        );
      } else {
        setPublicIPWarning(null);
      }
    } catch (err) {
      console.error('Failed to save public IP:', err);
    } finally {
      setIsSavingPublicIP(false);
    }
  };

  const handleRestart = async () => {
    if (
      !confirm(
        'Are you sure you want to restart the daemon? This will stop all running servers.',
      )
    ) {
      return;
    }
    setIsRestarting(true);
    try {
      await api.restartDaemon();
      alert(
        'Daemon restart command sent. The page may lose connection briefly.',
      );
    } catch (err) {
      alert(
        'Daemon restart command sent. The page may lose connection briefly.',
      );
      console.error('Failed to restart daemon:', err);
    } finally {
      setIsRestarting(false);
    }
  };

  if (loading) return <div>Loading settings...</div>;

  return (
    <div className="settings-page">
      <h1>Settings</h1>

      <div className="card">
        <h2>Network Configuration</h2>
        <p>
          Define the range of ports that the manager can assign to new servers.
        </p>

        <div>
          <div className="form-group">
            <label>Start Port</label>
            <input
              type="number"
              name="start"
              className="form-input"
              value={portRange.start}
              onChange={handleChange}
            />
          </div>
          <div className="form-group">
            <label>End Port</label>
            <input
              type="number"
              name="end"
              className="form-input"
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

      <div className="card" style={{ marginTop: '20px' }}>
        <h2>Public Address</h2>
        <p>
          Configure the IP address or hostname displayed for server connections.
          This address is used in the server list and public share links,
          allowing users to connect using a specific network interface (e.g.,
          VPN, public IP).
        </p>

        {publicIPWarning && (
          <div
            style={{
              backgroundColor: 'rgba(234, 179, 8, 0.15)',
              color: '#eab308',
              padding: '10px 14px',
              borderRadius: '6px',
              marginBottom: '16px',
              fontSize: '0.9rem',
            }}
          >
            ⚠️ {publicIPWarning}
          </div>
        )}

        <div className="form-group">
          <label>Public Address</label>
          <select
            className="form-input"
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

      <div className="card" style={{ marginTop: '20px' }}>
        <h2>Console Log Buffer</h2>
        <p>
          Define how many lines of console logs should be kept in memory per
          server while it is running.
        </p>
        <div className="form-group">
          <label>
            Lines to keep in memory{' '}
            <small style={{ fontWeight: 400 }}>(use 0 to disable)</small>
          </label>
          <input
            type="number"
            min={0}
            step={1}
            className="form-input"
            value={logBufferSize || ''}
            onChange={handleLogBufferChange}
          />
          {logBufferError && (
            <div style={{ color: 'red', marginTop: '6px' }}>
              {logBufferError}
            </div>
          )}
          <div style={{ marginTop: '8px', color: '#555' }}>
            <strong>Estimated memory usage:</strong> {humanSize(estimatedBytes)}
            <div style={{ fontSize: '12px', marginTop: '4px' }}>
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

      <div className="card" style={{ marginTop: '20px' }}>
        <h2>System</h2>
        <p>Manage the Naviger Daemon process.</p>
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
