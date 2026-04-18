import { expect, test, type Page } from "@playwright/test"
import * as OTPAuth from "otpauth"

const seededUsername = process.env.E2E_MFA_USERNAME ?? "justqiumfa"
const seededPassword = process.env.E2E_MFA_PASSWORD ?? "justqiu"
const seededEmail = process.env.E2E_MFA_EMAIL ?? "justqiumfa@local.test"

test.describe("backoffice auth + mfa", () => {
  test("seeded dev can enable MFA and finish login challenge", async ({ page }) => {
    await loginWithPassword(page)
    await expect(page).toHaveURL(/\/backoffice$/)

    await page.goto("/backoffice/profile")
    await expect(page.getByTestId("mfa-start")).toBeVisible()

    await page.getByTestId("mfa-start").click()

    const manualSecret = (await page.getByTestId("mfa-manual-secret").textContent())?.trim()
    if (!manualSecret) {
      throw new Error("MFA manual secret was not rendered")
    }

    await page.getByTestId("mfa-confirm-code").fill(await freshTotp(page, manualSecret))
    await page.getByTestId("mfa-confirm").click()

    await expect(
      page.getByText(/Recovery codes/i).first(),
    ).toBeVisible()

    const firstRecoveryCode = (await page
      .getByTestId("mfa-recovery-code")
      .first()
      .textContent())?.trim()
    if (!firstRecoveryCode) {
      throw new Error("Recovery code was not rendered after enabling MFA")
    }

    await logoutFromSidebar(page)
    await expect(page).toHaveURL(/\/login$/)

    await loginWithPassword(page)
    await expect(
      page.getByRole("heading", {
        name: /Verifikasi MFA untuk menyelesaikan login operator\./i,
      }),
    ).toBeVisible()

    await page
      .getByLabel("MFA code")
      .fill(await freshTotp(page, manualSecret))
    await page.getByRole("button", { name: /^Verify MFA$/ }).click()

    await expect(page).toHaveURL(/\/backoffice$/)

    await page.goto("/backoffice/profile")
    await page.getByTestId("mfa-disable-code").fill(firstRecoveryCode)
    await page.getByTestId("mfa-disable").click()

    await expect(page.getByTestId("mfa-start")).toBeVisible()
  })
})

async function loginWithPassword(page: Page) {
  await page.goto("/login")
  await page.getByLabel("Username atau Email").fill(seededUsername)
  await page.getByLabel("Password").fill(seededPassword)
  await page.getByRole("button", { name: /^Login$/ }).click()
}

async function logoutFromSidebar(page: Page) {
  await page.getByTestId("nav-user-trigger").click()
  await page.getByTestId("nav-user-logout").click()
}

async function freshTotp(page: Page, secret: string) {
  const totp = new OTPAuth.TOTP({
    issuer: "JustQiu Control",
    label: seededEmail,
    algorithm: "SHA1",
    digits: 6,
    period: 30,
    secret,
  })

  const remaining = totp.remaining()
  if (remaining < 1500) {
    await page.waitForTimeout(remaining + 750)
  }

  return totp.generate()
}
