import { backofficeRequest } from "@/lib/backoffice-api"

export type TopupTokoOption = {
  id: number
  name: string
  ownerUsername: string
  nexusggrBalance: number
}

export type PendingTopup = {
  amount: number
  transactionCode: string
  expiresAt?: number | null
  status: "pending" | "success" | "failed" | "expired"
  qrPayload: string
}

export type TopupRateRule = {
  thresholdAmount: number
  belowThresholdRate: number
  aboveThresholdRate: number
}

export type TopupBootstrapResponse = {
  data: {
    tokos: TopupTokoOption[]
    selectedToko?: TopupTokoOption | null
    topupRatio: number
    topupRule: TopupRateRule
    pendingTopup?: PendingTopup | null
  }
}

export type TopupMutationResponse = {
  message?: string
  data: {
    selectedToko?: TopupTokoOption | null
    topupRatio?: number
    topupRule?: TopupRateRule
    pendingTopup?: PendingTopup | null
    status?: string
  }
}

export async function getTopupBootstrap(tokoId?: number | null) {
  const searchParams = new URLSearchParams()
  if (tokoId != null) {
    searchParams.set("toko_id", String(tokoId))
  }

  return backofficeRequest<TopupBootstrapResponse>(
    `/backoffice/api/nexusggr-topup/bootstrap?${searchParams.toString()}`,
  )
}

export async function generateTopup(tokoId: number, amount: number) {
  return backofficeRequest<TopupMutationResponse>(
    "/backoffice/api/nexusggr-topup/generate",
    {
      method: "POST",
      body: JSON.stringify({
        tokoId,
        amount,
      }),
    },
  )
}

export async function checkTopupStatus(tokoId: number, transactionCode: string) {
  return backofficeRequest<TopupMutationResponse>(
    "/backoffice/api/nexusggr-topup/check-status",
    {
      method: "POST",
      body: JSON.stringify({
        tokoId,
        transactionCode,
      }),
    },
  )
}
