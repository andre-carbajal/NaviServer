import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

import React, { useEffect, useRef, useState } from 'react';

interface ConsoleViewProps {
  logs: string[];
}

const ConsoleView: React.FC<ConsoleViewProps> = ({ logs }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [terminalReady, setTerminalReady] = useState(false);

  const lastLogIndexRef = useRef(0);

  useEffect(() => {
    if (!terminalRef.current) return;

    const term = new Terminal({
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
        cursor: '#ffffff',
      },
      fontSize: 14,
      fontFamily: 'Consolas, "Courier New", monospace',
      cursorBlink: true,
      convertEol: true,
      disableStdin: true,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    let isOpened = false;

    const openTerminal = () => {
      if (isOpened || !terminalRef.current) return;

      if (
        terminalRef.current.clientWidth > 0 &&
        terminalRef.current.clientHeight > 0
      ) {
        try {
          term.open(terminalRef.current);
          fitAddon.fit();
          xtermRef.current = term;
          fitAddonRef.current = fitAddon;
          setTerminalReady(true);
          isOpened = true;
        } catch (e) {
          console.error('Failed to open terminal:', e);
        }
      }
    };

    const ro = new ResizeObserver(() => {
      if (!isOpened) {
        openTerminal();
      } else {
        try {
          fitAddon.fit();
        } catch {
          // Ignore fit errors on hidden elements
        }
      }
    });
    ro.observe(terminalRef.current);

    const timer = setTimeout(openTerminal, 50);

    return () => {
      clearTimeout(timer);
      ro.disconnect();
      xtermRef.current = null;
      fitAddonRef.current = null;
      try {
        term.dispose();
      } catch {
        // Ignore terminal disposal errors
      }
    };
  }, []);

  useEffect(() => {
    const term = xtermRef.current;
    if (!term || !terminalReady) return;

    if (logs.length > lastLogIndexRef.current) {
      const newLogs = logs.slice(lastLogIndexRef.current);
      newLogs.forEach((line) => {
        term.writeln(line);
      });
      lastLogIndexRef.current = logs.length;
      term.scrollToBottom();
    }
  }, [logs, terminalReady]);

  return (
    <div className="console-view">
      <div ref={terminalRef} className="console-view-terminal" />
    </div>
  );
};

export default ConsoleView;
