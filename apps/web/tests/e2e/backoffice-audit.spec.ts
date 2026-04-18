import { expect, test } from "@playwright/test"

import { loginAsSeededDev } from "./helpers"

test.describe("backoffice audit and finance shells", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsSeededDev(page)
  })

  test("seeded transaction can be opened in detail drawer", async ({ page }) => {
    await page.goto("/backoffice/transactions")
    await page.getByPlaceholder("Cari toko, owner, player, external, atau reference").fill("QRIS-DEPOSIT-001")
    const targetRow = page.getByRole("row").filter({ hasText: "QRIS-DEPOSIT-001" }).first()
    await expect(targetRow).toBeVisible()
    await targetRow.getByRole("button", { name: "Detail" }).click()
    const detailDialog = page.getByRole("dialog", { name: "Detail Transaksi" })
    await expect(detailDialog.getByText("Payload audit", { exact: false })).toBeVisible()
    await expect(detailDialog.getByText("QRIS-DEPOSIT-001", { exact: false })).toBeVisible()
  })

  test("transactions export downloads filtered csv", async ({ page }) => {
    await page.goto("/backoffice/transactions")
    await page.getByPlaceholder("Cari toko, owner, player, external, atau reference").fill("QRIS-DEPOSIT-001")
    const downloadPromise = page.waitForEvent("download")
    await page.getByRole("button", { name: "Export" }).click()
    await page.getByRole("menuitem", { name: "Export CSV" }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/^transactions-.*\.csv$/)
  })

  test("nexusggr topup bootstrap renders seeded toko option", async ({ page }) => {
    await page.goto("/backoffice/nexusggr-topup")
    await expect(page.getByText("Generate QRIS", { exact: false }).first()).toBeVisible()
    await expect(page.getByText("Demo Toko", { exact: false }).first()).toBeVisible()
    await expect(page.getByRole("button", { name: "Generate QRIS" })).toBeVisible()
  })

  test("withdrawal wizard bootstrap renders seeded toko and bank", async ({ page }) => {
    await page.goto("/backoffice/withdrawal")
    await expect(page.getByText("Withdrawal wizard", { exact: false })).toBeVisible()
    await expect(page.getByText("Demo Toko", { exact: false }).first()).toBeVisible()
    await page.getByRole("button", { name: "Lanjut ke rekening tujuan" }).click()
    await expect(page.getByText("BCA - 9876543210", { exact: false }).first()).toBeVisible()
  })
})
