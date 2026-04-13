import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: [
      // packages/ui internal aliases must come first (more specific)
      // ui components use @/lib/utils, @/components/ui/*, @/hooks/*
      {
        find: /^@\/(lib|components|hooks)(\/.*)?$/,
        replacement: path.resolve(__dirname, '../packages/ui/src/$1$2'),
      },
      // web-agent/ src alias for everything else
      { find: '@', replacement: path.resolve(__dirname, './src') },
    ],
  },
  server: {
    port: parseInt(process.env.VITE_AGENT_PORT || '3003', 10),
    host: '0.0.0.0',
    allowedHosts: true,
    watch: {
      // Watch packages/ui so HMR picks up shared component changes
      ignored: ['!**/packages/ui/src/**'],
    },
    proxy: {
      '/api': {
        target: process.env.VITE_API_AGENT_URL || 'http://localhost:8090',
        changeOrigin: true,
      },
      '/health': {
        target: process.env.VITE_API_AGENT_URL || 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
});
