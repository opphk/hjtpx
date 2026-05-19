import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : 2,
  reporter: [
    ['html', { 
      open: process.env.CI ? 'never' : 'on-first-failure',
      outputFolder: 'test-results/html'
    }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list'],
    ['junit', { outputFile: 'test-results/junit.xml' }]
  ],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    viewport: { width: 1280, height: 720 },
    actionTimeout: 30000,
    navigationTimeout: 60000,
    ignoreHTTPSErrors: true,
    launchOptions: {
      slowMo: process.env.SLOW_MO ? parseInt(process.env.SLOW_MO) : 0,
    },
  },

  projects: [
    // 桌面浏览器测试
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], headless: true },
    },
    {
      name: 'chromium-headful',
      use: { ...devices['Desktop Chrome'], headless: false },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'], headless: true },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'], headless: true },
    },
    
    // 移动设备测试
    {
      name: 'iPhone 12',
      use: { ...devices['iPhone 12'], headless: true },
    },
    {
      name: 'iPhone 12 landscape',
      use: { ...devices['iPhone 12 landscape'], headless: true },
    },
    {
      name: 'iPad Pro 11',
      use: { ...devices['iPad Pro 11'], headless: true },
    },
    {
      name: 'Pixel 5',
      use: { ...devices['Pixel 5'], headless: true },
    },
    {
      name: 'Samsung Galaxy S10',
      use: { ...devices['Samsung Galaxy S10'], headless: true },
    },
  ],

  outputDir: 'test-results',
  
  timeout: 60000,
  
  expect: {
    timeout: 10000,
    toMatchSnapshot: {
      maxDiffPixels: 10,
    },
  },

  globalSetup: async () => {
    // 全局设置
    console.log('Running global setup...');
  },

  globalTeardown: async () => {
    // 全局清理
    console.log('Running global teardown...');
  },
});
