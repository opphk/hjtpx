import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [
    ['html'],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list']
  ],
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on',
    screenshot: 'on',
    video: 'off',
    viewport: { width: 1280, height: 720 },
    actionTimeout: 30000,
    navigationTimeout: 60000,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], headless: true },
    },
  ],

  outputDir: 'test-results',
});
