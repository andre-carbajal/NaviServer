import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const envApiPort = Number(env.VITE_API_PORT);
  const apiPort = Number.isFinite(envApiPort) && envApiPort > 0 ? envApiPort : 23008;

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
