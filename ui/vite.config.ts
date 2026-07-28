import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: false,
        timeout: 600000
      },
      '/rpc': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000
      },
      '/api/channels': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        timeout: 600000
      },
      '/ws': {
        target: 'ws://localhost:28789',
        ws: true,
        changeOrigin: false,
      },
    },
  },
});
