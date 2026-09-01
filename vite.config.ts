import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  },
  build: {
    outDir: 'web/dist',
    emptyOutDir: true
  },
  test: {
    environment: 'node',
    include: ['web/src/**/*.test.ts']
  }
});
