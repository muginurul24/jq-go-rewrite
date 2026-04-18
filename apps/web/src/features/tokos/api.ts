import { backofficeRequest } from "@/lib/backoffice-api"

export type TokoOwnerOption = {
  id: number
  username: string
  name: string
}

export type TokoRecord = {
  id: number
  userId: number
  ownerUsername: string
  ownerName: string
  name: string
  callbackUrl?: string | null
  callbackHost: string
  token?: string | null
  tokenPreview?: string | null
  isActive: boolean
  balances: {
    pending: number
    settle: number
    nexusggr: number
  }
  createdAt: string
  updatedAt: string
}

export type TokosListResponse = {
  data: TokoRecord[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  summary: {
    totalTokos: number
    activeTokos: number
    totalPending: number
    totalSettle: number
    totalNexusggr: number
  }
  filters: {
    owners: TokoOwnerOption[]
  }
}

export type TokoDetailResponse = {
  data: TokoRecord
}

export type TokoMutationResponse = {
  message: string
  data: TokoRecord
}

export type TokosListParams = {
  search?: string
  status?: string
  ownerId?: string
  page: number
  perPage: number
}

export type TokoPayload = {
  userId?: number
  name: string
  callbackUrl?: string | null
  isActive: boolean
}

export async function listTokos(params: TokosListParams) {
  const searchParams = new URLSearchParams()

  if (params.search) {
    searchParams.set("search", params.search)
  }
  if (params.status) {
    searchParams.set("status", params.status)
  }
  if (params.ownerId) {
    searchParams.set("owner_id", params.ownerId)
  }

  searchParams.set("page", String(params.page))
  searchParams.set("per_page", String(params.perPage))

  return backofficeRequest<TokosListResponse>(
    `/backoffice/api/tokos?${searchParams.toString()}`,
  )
}

export async function getTokoDetail(tokoId: number) {
  return backofficeRequest<TokoDetailResponse>(`/backoffice/api/tokos/${tokoId}`)
}

export async function createToko(payload: TokoPayload) {
  return backofficeRequest<TokoMutationResponse>("/backoffice/api/tokos", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function updateToko(tokoId: number, payload: TokoPayload) {
  return backofficeRequest<TokoMutationResponse>(`/backoffice/api/tokos/${tokoId}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  })
}

export async function regenerateTokoToken(tokoId: number) {
  return backofficeRequest<TokoMutationResponse>(
    `/backoffice/api/tokos/${tokoId}/regenerate-token`,
    {
      method: "POST",
      body: JSON.stringify({}),
    },
  )
}
