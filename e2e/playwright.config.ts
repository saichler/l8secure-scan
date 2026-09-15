import { defineConfig, devices } from '@playwright/test';
import { BASE_URL } from './fixtures/env';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  reporter: 'html',
  use: {
    baseURL: BASE_URL,
    ignoreHTTPSErrors: true,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium-desktop',
      testDir: './tests/desktop',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'chromium-mobile',
      testDir: './tests/mobile',
      // Real mobile bundle (m/app.html), not a resize of the desktop one --
      // see plans/playwright-e2e-testing.md Phase 5.
      use: { ...devices['iPhone 13'] },
    },
  ],
});
