import path from "node:path"
import { fileURLToPath } from "node:url"

import { defineConfig, devices } from "@playwright/test"

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:5173"
const manageServers = process.env.PLAYWRIGHT_MANAGED_SERVERS === "true"
const configDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 90_000,
  workers: 1,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI
    ? [["html", { open: "never" }], ["line"]]
    : "list",
  use: {
    baseURL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    testIdAttribute: "data-testid",
  },
  webServer: manageServers
    ? [
        {
          command: "pnpm dev:api",
          cwd: path.resolve(configDir, "../.."),
          url: "http://127.0.0.1:8080/health/ready",
          reuseExistingServer: !process.env.CI,
          timeout: 180_000,
        },
        {
          command: "pnpm dev:web",
          cwd: path.resolve(configDir, "../.."),
          url: baseURL,
          reuseExistingServer: !process.env.CI,
          timeout: 180_000,
        },
      ]
    : undefined,
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 960 },
      },
    },
  ],
})
