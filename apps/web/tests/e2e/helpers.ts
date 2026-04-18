import { expect, type Page } from "@playwright/test"

export const seededUsername = process.env.E2E_DEV_USERNAME ?? "justqiu"
export const seededPassword = process.env.E2E_DEV_PASSWORD ?? "justqiu"

export async function loginAsSeededDev(page: Page) {
  await page.goto("/login")
  await page.getByLabel("Username atau Email").fill(seededUsername)
  await page.getByLabel("Password").fill(seededPassword)
  await page.getByRole("button", { name: /^Login$/ }).click()
  await expect(page).toHaveURL(/\/backoffice$/)
}

export async function openSelectOption(page: Page, trigger: Parameters<Page["getByRole"]>[1], optionName: RegExp | string) {
  await page.getByRole("combobox", trigger).click()
  await page.getByRole("option", { name: optionName }).click()
}
