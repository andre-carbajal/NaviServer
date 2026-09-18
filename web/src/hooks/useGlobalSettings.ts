import { useEffect, useState } from 'react';

import { api } from '../services/api';
import type { AlertOptions, ConfirmOptions } from './useModalDialog';

export const BYTES_PER_LINE_ESTIMATE = 200;

export const useGlobalSettings = (
  showAlert: (options: AlertOptions) => Promise<void>,
  showConfirm: (options: ConfirmOptions) => Promise<boolean>,
) => {
  const [portRange, setPortRange] = useState({ start: 0, end: 0 });
  const [initialPortRange, setInitialPortRange] = useState({
    start: 0,
    end: 0,
  });
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
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
  const [curseForgeKey, setCurseForgeKey] = useState('');
  const [isSavingCurseForgeKey, setIsSavingCurseForgeKey] = useState(false);
  const [isClearingCurseForgeKey, setIsClearingCurseForgeKey] = useState(false);
  const [curseForgeKeyStatus, setCurseForgeKeyStatus] = useState({
    hasCustomKey: false,
    hasEmbeddedKey: false,
    effectiveSource: 'none' as 'custom' | 'embedded' | 'none',
  });

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const [ports, logs, publicAddress, interfaces, curseForge] =
          await Promise.all([
            api.getPortRange(),
            api.getLogBufferSize(),
            api.getPublicIP(),
            api.getNetworkInterfaces(),
            api.getCurseForgeKeyStatus(),
          ]);
        setPortRange(ports.data);
        setInitialPortRange(ports.data);
        const size = logs.data?.log_buffer_size ?? 1000;
        setLogBufferSize(size);
        setInitialLogBufferSize(size);
        const savedIP = publicAddress.data?.public_ip ?? 'localhost';
        const ifaces = interfaces.data?.interfaces ?? [];
        setPublicIP(savedIP);
        setInitialPublicIP(savedIP);
        setNetworkInterfaces(ifaces);
        setCurseForgeKeyStatus(curseForge.data);
        if (savedIP !== 'localhost' && !ifaces.includes(savedIP))
          setPublicIPWarning(
            `The configured IP "${savedIP}" is not currently available on any network interface.`,
          );
      } catch (error) {
        console.error('Failed to fetch settings:', error);
      } finally {
        setLoading(false);
      }
    };
    void fetchSettings();
  }, []);

  const savePorts = async () => {
    setIsSaving(true);
    try {
      await api.updatePortRange(portRange);
      setInitialPortRange(portRange);
    } catch (error) {
      console.error('Failed to save settings:', error);
    } finally {
      setIsSaving(false);
    }
  };
  const changeLogBuffer = (raw: string) => {
    const parsed = Number.parseInt(raw, 10);
    if (raw === '') {
      setLogBufferSize(0);
      setLogBufferError(null);
      return;
    }
    if (Number.isNaN(parsed)) {
      setLogBufferSize(0);
      setLogBufferError('The value must be an integer >= 0');
      return;
    }
    setLogBufferSize(parsed);
    setLogBufferError(parsed < 0 ? 'The value cannot be negative' : null);
  };
  const saveLogBuffer = async () => {
    if (logBufferSize < 0) {
      setLogBufferError('The value must be an integer >= 0');
      return;
    }
    setIsSavingLogBuffer(true);
    try {
      await api.updateLogBufferSize({ log_buffer_size: logBufferSize });
      setInitialLogBufferSize(logBufferSize);
    } catch (error) {
      console.error('Failed to save log buffer size:', error);
      setLogBufferError('The save operation failed. Please try again.');
    } finally {
      setIsSavingLogBuffer(false);
    }
  };
  const savePublicIP = async () => {
    setIsSavingPublicIP(true);
    try {
      await api.updatePublicIP({ public_ip: publicIP });
      setInitialPublicIP(publicIP);
      setPublicIPWarning(
        publicIP !== 'localhost' && !networkInterfaces.includes(publicIP)
          ? `The configured IP "${publicIP}" is not currently available on any network interface.`
          : null,
      );
    } catch (error) {
      console.error('Failed to save public IP:', error);
    } finally {
      setIsSavingPublicIP(false);
    }
  };
  const saveCurseForgeKey = async () => {
    const value = curseForgeKey.trim();
    if (!value) return;
    setIsSavingCurseForgeKey(true);
    try {
      await api.setCurseForgeKey(value);
      setCurseForgeKeyStatus((await api.getCurseForgeKeyStatus()).data);
      setCurseForgeKey('');
    } catch (error) {
      console.error('Failed to save CurseForge API key:', error);
      await showAlert({
        title: 'Save Failed',
        message: 'Failed to save CurseForge API key.',
        variant: 'danger',
      });
    } finally {
      setIsSavingCurseForgeKey(false);
    }
  };
  const clearCurseForgeKey = async () => {
    setIsClearingCurseForgeKey(true);
    try {
      await api.clearCurseForgeKey();
      setCurseForgeKeyStatus((await api.getCurseForgeKeyStatus()).data);
    } catch (error) {
      console.error('Failed to clear CurseForge API key:', error);
      await showAlert({
        title: 'Clear Failed',
        message: 'Failed to clear CurseForge API key.',
        variant: 'danger',
      });
    } finally {
      setIsClearingCurseForgeKey(false);
    }
  };
  const restart = async () => {
    if (
      !(await showConfirm({
        title: 'Restart Daemon',
        message:
          'Are you sure you want to restart the daemon? This will stop all running servers.',
        confirmText: 'Restart',
        variant: 'danger',
      }))
    )
      return;
    setIsRestarting(true);
    try {
      await api.restartDaemon();
      await showAlert({
        title: 'Restart Sent',
        message:
          'Daemon restart command sent. The page may lose connection briefly.',
      });
    } catch (error) {
      console.error('Failed to restart daemon:', error);
      await showAlert({
        title: 'Restart Failed',
        message: 'Failed to send daemon restart command. Please try again.',
        variant: 'danger',
      });
    } finally {
      setIsRestarting(false);
    }
  };

  return {
    loading,
    portRange,
    setPortRange,
    portsChanged:
      JSON.stringify(portRange) !== JSON.stringify(initialPortRange),
    isSaving,
    savePorts,
    publicIP,
    setPublicIP,
    initialPublicIP,
    networkInterfaces,
    publicIPWarning,
    isSavingPublicIP,
    savePublicIP,
    curseForgeKey,
    setCurseForgeKey,
    curseForgeKeyStatus,
    isSavingCurseForgeKey,
    isClearingCurseForgeKey,
    saveCurseForgeKey,
    clearCurseForgeKey,
    logBufferSize,
    initialLogBufferSize,
    logBufferError,
    isSavingLogBuffer,
    changeLogBuffer,
    saveLogBuffer,
    isRestarting,
    restart,
  };
};
