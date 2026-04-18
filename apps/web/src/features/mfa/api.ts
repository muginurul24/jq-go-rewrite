import { backofficeRequest } from "@/lib/backoffice-api"

export type MfaStatusResponse = {
  enabled: boolean
  pendingSetup: boolean
  secret?: string
  otpauthUrl?: string
}

export type MfaSetupResponse = {
  enabled: boolean
  pendingSetup: boolean
  secret?: string
  otpauthUrl?: string
}

export type MfaConfirmResponse = {
  enabled: boolean
  recoveryCodes?: string[]
}

export async function getMfaStatus() {
  return backofficeRequest<MfaStatusResponse>("/backoffice/api/auth/mfa", {
    method: "GET",
  })
}

export async function beginMfaSetup() {
  return backofficeRequest<MfaSetupResponse>("/backoffice/api/auth/mfa/setup", {
    method: "POST",
    body: JSON.stringify({}),
  })
}

export async function confirmMfaSetup(code: string) {
  return backofficeRequest<MfaConfirmResponse>("/backoffice/api/auth/mfa/confirm", {
    method: "POST",
    body: JSON.stringify({ code }),
  })
}

export async function disableMfa(code: string) {
  return backofficeRequest<MfaStatusResponse>("/backoffice/api/auth/mfa/disable", {
    method: "POST",
    body: JSON.stringify({ code }),
  })
}
