import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  workers: 2,
  timeout: 30000,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:5174',
    channel: 'chrome',
    viewport: { width: 1280, height: 1000 },
    locale: 'ru-RU',
    timezoneId: 'Europe/Moscow',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'npm run dev -- --port 5174',
    env: { VITE_ENABLE_DEMO_TOOLS: 'true', VITE_CLIENT_ID: 'anya', VITE_API_BASE_URL: '' },
    url: 'http://127.0.0.1:5174',
    reuseExistingServer: false,
  },
});
