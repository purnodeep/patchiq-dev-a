import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
  },
  resolve: {
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
});
