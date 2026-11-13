// Run: cd e2e && npm ci && npx playwright test   (needs ../homeroll built: `make` in the repo root)
// Each run starts the real binary on a fresh data dir and a generated fixture library (made once, cached in .fixtures).
import { defineConfig, devices } from '@playwright/test';

export const BASE = 'http://127.0.0.1:8090';

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  workers: 1, // one server, shared state: specs run in order and clean up after themselves
  fullyParallel: false,
  retries: 0,
  reporter: [['list']],
  globalSetup: './global-setup.ts',
  use: { baseURL: BASE, trace: 'off', screenshot: 'only-on-failure' },
  webServer: {
    command: 'rm -rf .data && mkdir .data && ../homeroll --data-dir .data --listen 127.0.0.1:8090',
    url: BASE + '/api/status',
    reuseExistingServer: false,
    stdout: 'ignore',
    stderr: 'pipe'
  },
  projects: [
    { name: 'setup', testMatch: 'setup.spec.ts', use: { viewport: { width: 1440, height: 900 } } },
    {
      name: 'desktop',
      testIgnore: 'setup.spec.ts',
      dependencies: ['setup'],
      use: { viewport: { width: 1440, height: 900 }, colorScheme: 'light', storageState: '.auth.json' }
    },
    {
      name: 'mobile',
      testIgnore: 'setup.spec.ts',
      grep: /@mobile/,
      dependencies: ['desktop'],
      use: { ...devices['iPhone 13'], defaultBrowserType: 'chromium', colorScheme: 'dark', storageState: '.auth.json' }
    }
  ]
});
