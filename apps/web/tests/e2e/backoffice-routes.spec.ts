import { expect, test } from "@playwright/test"

import { loginAsSeededDev } from "./helpers"

const routeChecks = [
  { path: "/backoffice", text: "Recent transactions" },
  { path: "/backoffice/operational-pulse", text: "Operational pulse" },
  { path: "/backoffice/users", text: "User management" },
  { path: "/backoffice/tokos", text: "Tokos control" },
  { path: "/backoffice/banks", text: "Banks" },
  { path: "/backoffice/players", text: "Players registry" },
  { path: "/backoffice/transactions", text: "Transaction audit center" },
  { path: "/backoffice/providers", text: "Providers" },
  { path: "/backoffice/games", text: "Games" },
  { path: "/backoffice/call-management", text: "Call management" },
  { path: "/backoffice/nexusggr-topup", text: "NexusGGR topup" },
  { path: "/backoffice/withdrawal", text: "Withdrawal wizard" },
  { path: "/backoffice/api-docs", text: "Autentikasi" },
  { path: "/backoffice/profile", text: "Multi-factor authentication" },
] as const

test.describe("backoffice route smoke", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsSeededDev(page)
  })

  for (const check of routeChecks) {
    test(`loads ${check.path}`, async ({ page }) => {
      await page.goto(check.path)
      await expect(page).toHaveURL(new RegExp(check.path.replace(/\//g, "\\/")))
      await expect(page.getByText(check.text, { exact: false }).first()).toBeVisible()
    })
  }
})
