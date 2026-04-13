import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      // packages/ui internal aliases must come first (more specific)
      // ui components use @/lib/utils, @/components/ui/*, @/hooks/*
      {
        find: /^@\/(lib|components|hooks)(\/.*)?$/,
        replacement: path.resolve(__dirname, '../packages/ui/src/$1$2'),
      },
      // web/ src alias for everything else
      { find: '@', replacement: path.resolve(__dirname, './src') },
    ],
  },
  server: {
    port: parseInt(process.env.VITE_SERVER_PORT || '3001', 10),
    host: '0.0.0.0',
    allowedHosts: true,
    proxy: {
      '/api': {
        target: process.env.VITE_API_SERVER_URL ?? 'http://localhost:8080',
        changeOrigin: true,
        headers: {
          'X-Tenant-ID': '00000000-0000-0000-0000-000000000001',
          'X-User-ID': 'dev-user',
        },
      },
      '/health': {
        target: process.env.VITE_API_SERVER_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
      '^/repo(/|$)': {
        target: process.env.VITE_API_SERVER_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
