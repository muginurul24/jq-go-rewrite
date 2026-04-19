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

  test("notification center and notifications page render seeded database notifications", async ({ page }) => {
    await page.goto("/backoffice")
    await page.getByTestId("notification-center-trigger").click()
    await expect(page.getByText("Withdrawal Pending", { exact: false }).first()).toBeVisible()
    await page.getByRole("link", { name: "View all" }).click()
    await expect(page).toHaveURL(/\/backoffice\/notifications$/)
    await expect(page.getByText("Notification Center", { exact: false }).first()).toBeVisible()
    await expect(page.getByText("Toko callback gagal", { exact: false }).first()).toBeVisible()
  })

  test("call management mirrors legacy table flow and actions", async ({ page }) => {
    await page.route("**/backoffice/api/catalog/providers", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          success: true,
          providers: [
            { code: "PG", name: "PG Soft", status: 1 },
            { code: "DISABLED", name: "Disabled Provider", status: 0 },
          ],
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/bootstrap", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            managedPlayers: [
              {
                id: 11,
                username: "shareduser",
                userLabel: "shareduser (Toko Alpha)",
                tokoName: "Toko Alpha",
              },
            ],
          },
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/active-players", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [
            {
              playerId: 11,
              username: "shareduser",
              userLabel: "shareduser (Toko Alpha)",
              tokoName: "Toko Alpha",
              providerCode: "PG",
              gameCode: "G1",
              bet: 1200,
              balance: 45000,
              totalDebit: 7000,
              totalCredit: 5800,
              targetRtp: 92,
              realRtp: 83,
            },
          ],
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/call-list", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [
            {
              rtp: 77,
              callType: "buy_bonus_free",
              callTypeValue: 2,
            },
          ],
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/history", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [
            {
              id: 901,
              playerId: 11,
              username: "shareduser",
              userLabel: "shareduser (Toko Alpha)",
              tokoName: "Toko Alpha",
              providerCode: "PG",
              gameCode: "G1",
              bet: 1200,
              expect: 10000,
              real: 8800,
              missed: 1200,
              rtp: 77,
              type: "buy_bonus_free",
              status: 0,
              statusLabel: "Waiting",
              canCancel: true,
              createdAt: "2026-04-19T12:00:00Z",
            },
            {
              id: 902,
              playerId: 11,
              username: "shareduser",
              userLabel: "shareduser (Toko Alpha)",
              tokoName: "Toko Alpha",
              providerCode: "PG",
              gameCode: "G2",
              bet: 2200,
              expect: 12000,
              real: 11800,
              missed: 200,
              rtp: 95,
              type: "common_free",
              status: 2,
              statusLabel: "Finished",
              canCancel: false,
              createdAt: "2026-04-19T11:45:00Z",
            },
          ],
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/apply", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          message: "Call applied",
          data: {
            calledMoney: 25000,
          },
        }),
      })
    })

    await page.route("**/backoffice/api/call-management/cancel", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          message: "Call canceled",
          data: {
            canceledMoney: 12500,
          },
        }),
      })
    })

    await page.goto("/backoffice/call-management")

    await expect(page.getByText("Active players", { exact: false }).first()).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "Total Debit" })).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "Total Credit" })).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "Target RTP" })).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "Real RTP" })).toBeVisible()

    await page.getByRole("button", { name: "View Calls" }).click()
    await expect(page.getByText("Call List", { exact: false }).first()).toBeVisible()
    await expect(page.getByText("shareduser (Toko Alpha) - PG / G1", { exact: false })).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "RTP" }).first()).toBeVisible()
    await expect(page.getByRole("columnheader", { name: "Type" }).first()).toBeVisible()

    await page.getByRole("button", { name: "Apply" }).click()
    const applyDialog = page.getByRole("dialog", { name: "Apply Call" })
    await expect(applyDialog).toBeVisible()
    await expect(applyDialog.locator("input").first()).toHaveValue("shareduser (Toko Alpha)")
    await expect(applyDialog.getByText("Buy Bonus Free", { exact: false })).toBeVisible()
    await applyDialog.getByRole("button", { name: "Apply Call" }).click()
    await expect(page.getByText("Called money: 25000", { exact: false })).toBeVisible()

    await page.getByRole("button", { name: "Control RTP" }).click()
    const controlDialog = page.getByRole("dialog", { name: "Control RTP" })
    await expect(controlDialog).toBeVisible()
    await controlDialog.getByRole("combobox").first().click()
    await expect(page.getByRole("option", { name: "PG Soft" })).toBeVisible()
    await expect(page.getByRole("option", { name: "Disabled Provider" })).toHaveCount(0)
    await page.keyboard.press("Escape")
    await controlDialog.getByRole("button", { name: "Batal" }).click()

    await expect(page.getByText("Waiting", { exact: false }).first()).toBeVisible()
    await expect(page.getByText("Finished", { exact: false }).first()).toBeVisible()
    await page.getByRole("button", { name: "Cancel" }).click()
    const cancelDialog = page.getByRole("dialog", { name: "Cancel Call" })
    await expect(cancelDialog).toBeVisible()
    await cancelDialog.getByRole("button", { name: "Confirm Cancel" }).click()
    await expect(page.getByText("Canceled money: 12500", { exact: false })).toBeVisible()
  })
})
