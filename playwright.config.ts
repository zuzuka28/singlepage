import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './web/e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  expect: { timeout: 8_000 },
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'retain-on-failure'
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], channel: 'chrome' } }],
  webServer: {
    command: "npm run build && SINGLEPAGE_HTTP_LISTEN=127.0.0.1:4173 SINGLEPAGE_METRICS_LISTEN=127.0.0.1:4174 SINGLEPAGE_SQLITE_DSN='file:e2e?mode=memory&cache=shared' go run ./cmd/web",
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: false,
    timeout: 120_000
  }
});
