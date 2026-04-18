import { expect, test } from "@playwright/test"

import { loginAsSeededDev } from "./helpers"

test.describe("backoffice local CRUD", () => {
  test("dev can create user, toko, and bank through the UI", async ({ page }) => {
    const unique = Date.now().toString().slice(-6)
    const username = `smoke${unique}`
    const tokoName = `Smoke Toko ${unique}`
    const accountNumber = `9900${unique}`

    await loginAsSeededDev(page)

    await page.goto("/backoffice/users")
    await page.getByRole("button", { name: "User baru" }).click()
    const userDialog = page.getByTestId("users-dialog")
    await userDialog.getByTestId("user-form-username").fill(username)
    await userDialog.getByTestId("user-form-name").fill(`Smoke ${unique}`)
    await userDialog.getByTestId("user-form-email").fill(`${username}@local.test`)
    await userDialog.getByTestId("user-form-password").fill("justqiu123")
    await userDialog.getByTestId("user-form-submit").click()
    await expect(page.getByText(username, { exact: true }).first()).toBeVisible()

    await page.goto("/backoffice/tokos")
    await page.getByRole("button", { name: "Toko baru" }).click()
    const tokoDialog = page.getByTestId("tokos-dialog")
    await tokoDialog.getByTestId("toko-form-owner").click()
    await page.getByRole("option", { name: new RegExp(`${username}.*Smoke ${unique}`, "i") }).click()
    await tokoDialog.getByTestId("toko-form-name").fill(tokoName)
    await tokoDialog.getByTestId("toko-form-callback-url").fill(`${username}.example.test/callback`)
    await tokoDialog.getByTestId("toko-form-submit").click()
    await expect(page.getByText(tokoName, { exact: true }).first()).toBeVisible()
    await expect(page.getByRole("button", { name: "Copy token" })).toBeVisible()

    await page.goto("/backoffice/banks")
    await page.getByRole("button", { name: "Tambah rekening" }).click()
    const bankDialog = page.getByTestId("banks-dialog")
    await bankDialog.getByTestId("bank-form-owner").click()
    await page.getByRole("option", { name: new RegExp(`${username}.*Smoke ${unique}`, "i") }).click()
    await bankDialog.getByTestId("bank-form-bank-code").click()
    await page.getByRole("option", { name: /^BCA$/ }).click()
    await bankDialog.getByTestId("bank-form-account-number").fill(accountNumber)
    await bankDialog.getByTestId("bank-form-account-name").fill(`SMOKE ${unique}`)
    await bankDialog.getByTestId("bank-form-submit").click()
    await expect(page.getByText(accountNumber, { exact: true }).first()).toBeVisible()
  })
})
