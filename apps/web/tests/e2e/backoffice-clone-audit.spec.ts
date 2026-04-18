import { expect, test, type Page, type Response } from "@playwright/test"

import { loginAsSeededDev } from "./helpers"

const cloneAuditEnabled = process.env.PLAYWRIGHT_CLONE_AUDIT === "true"

type RuntimeAudit = {
  pageErrors: string[]
  consoleErrors: string[]
  keyWarnings: string[]
  serverErrors: string[]
}

function installRuntimeAudit(page: Page): RuntimeAudit {
  const audit: RuntimeAudit = {
    pageErrors: [],
    consoleErrors: [],
    keyWarnings: [],
    serverErrors: [],
  }

  page.on("pageerror", (error) => {
    audit.pageErrors.push(error.message)
  })

  page.on("console", (message) => {
    const text = message.text()
    if (message.type() === "error") {
      audit.consoleErrors.push(text)
      return
    }

    if (text.includes("Each child in a list should have a unique \"key\" prop")) {
      audit.keyWarnings.push(text)
    }
  })

  page.on("response", (response: Response) => {
    if (response.status() < 500) {
      return
    }

    const url = response.url()
    if (!url.startsWith("http://localhost:5173") && !url.startsWith("http://localhost:8080")) {
      return
    }

    audit.serverErrors.push(`${response.status()} ${url}`)
  })

  return audit
}

async function expectCleanRuntime(audit: RuntimeAudit) {
  expect(
    audit.pageErrors,
    `Unexpected page errors:\n${audit.pageErrors.join("\n")}`,
  ).toEqual([])
  expect(
    audit.consoleErrors,
    `Unexpected console errors:\n${audit.consoleErrors.join("\n")}`,
  ).toEqual([])
  expect(
    audit.keyWarnings,
    `Unexpected React key warnings:\n${audit.keyWarnings.join("\n")}`,
  ).toEqual([])
  expect(
    audit.serverErrors,
    `Unexpected 5xx responses:\n${audit.serverErrors.join("\n")}`,
  ).toEqual([])
}

test.describe("backoffice clone audit", () => {
  test.skip(
    !cloneAuditEnabled,
    "Requires explicit PLAYWRIGHT_CLONE_AUDIT=true with production-clone DB and upstream-ready env.",
  )

  test.beforeEach(async ({ page }) => {
    await loginAsSeededDev(page)
  })

  test("core pages render cleanly on production clone data", async ({ page }) => {
    const audit = installRuntimeAudit(page)

    const pages = [
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

    for (const entry of pages) {
      await page.goto(entry.path)
      await expect(page).toHaveURL(new RegExp(entry.path.replace(/\//g, "\\/")))
      await expect(page.getByText(entry.text, { exact: false }).first()).toBeVisible()
    }

    await expectCleanRuntime(audit)
  })

  test("transactions detail and export work against clone data", async ({ page }) => {
    const audit = installRuntimeAudit(page)

    await page.goto("/backoffice/transactions")
    await expect(page.getByRole("table")).toBeVisible()

    const detailButton = page.getByRole("button", { name: "Detail" }).first()
    await expect(detailButton).toBeVisible()
    await detailButton.click()

    const detailDialog = page.getByRole("dialog", { name: "Detail Transaksi" })
    await expect(detailDialog).toBeVisible()
    await expect(detailDialog.getByText("Payload audit", { exact: false })).toBeVisible()
    await page.keyboard.press("Escape")
    await expect(detailDialog).not.toBeVisible()

    const downloadPromise = page.waitForEvent("download")
    await page.getByRole("button", { name: "Export" }).click()
    await page.getByRole("menuitem", { name: "Export CSV" }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/^transactions-.*\.csv$/)

    await expectCleanRuntime(audit)
  })

  test("topup and withdrawal bootstrap show real toko and bank labels", async ({ page }) => {
    const audit = installRuntimeAudit(page)

    await page.goto("/backoffice/nexusggr-topup")
    await expect(page.getByText("Generate QRIS", { exact: false }).first()).toBeVisible()

    await page.getByRole("combobox").first().click()
    const topupOptions = page.getByRole("option")
    const topupCount = await topupOptions.count()
    expect(topupCount).toBeGreaterThan(0)
    for (let index = 0; index < topupCount; index += 1) {
      await expect(topupOptions.nth(index)).not.toHaveText(/^\s*\(\s*\)\s*$/)
    }
    await page.keyboard.press("Escape")

    await page.goto("/backoffice/withdrawal")
    await expect(page.getByText("Withdrawal wizard", { exact: false })).toBeVisible()
    await page.getByRole("button", { name: "Lanjut ke rekening tujuan" }).click()

    await page.getByRole("combobox").first().click()
    const bankOptions = page.getByRole("option")
    const bankCount = await bankOptions.count()
    expect(bankCount).toBeGreaterThan(0)
    for (let index = 0; index < bankCount; index += 1) {
      await expect(bankOptions.nth(index)).not.toHaveText(/^\s*$/)
    }

    await expectCleanRuntime(audit)
  })

  test("providers, games, and call-management stay usable with clone data", async ({ page }) => {
    const audit = installRuntimeAudit(page)

    await page.goto("/backoffice/providers")
    await expect(page.getByText("Providers", { exact: false }).first()).toBeVisible()
    const providerEvidence = page
      .getByText("Code:", { exact: false })
      .first()
      .or(page.getByText("Belum ada provider", { exact: false }).first())
    await expect(providerEvidence).toBeVisible()

    await page.goto("/backoffice/games")
    await expect(page.getByText("Games", { exact: false }).first()).toBeVisible()
    await expect(
      page.getByText(/Game tidak ditemukan|Standard names|Localized names/, { exact: false }).first(),
    ).toBeVisible()

    await page.goto("/backoffice/call-management")
    await expect(page.getByText("Call management", { exact: false }).first()).toBeVisible()
    await expect(page.getByText("Active players", { exact: false }).first()).toBeVisible()

    await expectCleanRuntime(audit)
  })
})
