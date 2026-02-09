import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const isDev = env.NAVIGER_DEV === 'true' || env.NAVIGER_DEV === '1';
  const apiPort = isDev ? 23009 : 23008;

  return {
    plugins: [react()],
    build: {
      chunkSizeWarningLimit: 1000,
    },
    server: {
      port: 5173,
    },
    define: {
      'import.meta.env.VITE_API_PORT': JSON.stringify(apiPort),
    },
  };
});
