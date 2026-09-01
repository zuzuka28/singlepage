import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

declare const process: { env: Record<string, string | undefined> };

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  },
  build: {
    outDir: process.env.SINGLEPAGE_TARGET === 'app'
      ? 'cmd/app/internal/app/frontend/dist'
      : 'internal/handler/frontend/dist',
    emptyOutDir: false,
    rollupOptions: process.env.SINGLEPAGE_TARGET === 'app'
      ? { external: ['/wails/runtime.js'] }
      : undefined
  },
  test: {
    environment: 'node',
    include: ['web/src/**/*.test.ts']
  }
});
