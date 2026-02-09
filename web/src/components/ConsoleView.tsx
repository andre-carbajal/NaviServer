import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

import React, { useEffect, useRef } from 'react';

interface ConsoleViewProps {
  logs: string[];
}

const ConsoleView: React.FC<ConsoleViewProps> = ({ logs }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

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

    term.open(terminalRef.current);
    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    const performFit = () => {
      if (!xtermRef.current || !terminalRef.current) return;

      if (terminalRef.current.clientHeight === 0) return;

      fitAddon.fit();
    };

    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(() => performFit());
    });
    resizeObserver.observe(terminalRef.current);

    const handleWindowResize = () => performFit();
    window.addEventListener('resize', handleWindowResize);

    setTimeout(() => performFit(), 50);

    return () => {
      window.removeEventListener('resize', handleWindowResize);
      resizeObserver.disconnect();

      xtermRef.current = null;
      fitAddonRef.current = null;

      try {
        term.dispose();
      } catch (e) {
        console.warn('Error disposing terminal:', e);
      }
    };
  }, []);

  useEffect(() => {
    const term = xtermRef.current;
    if (!term) return;

    const newLogs = logs.slice(lastLogIndexRef.current);
    if (newLogs.length > 0) {
      newLogs.forEach((line) => term.writeln(line));

      lastLogIndexRef.current = logs.length;
    }
  }, [logs]);

  return (
    <div className="console-view">
      <div ref={terminalRef} className="console-view-terminal" />
    </div>
  );
};

export default ConsoleView;
