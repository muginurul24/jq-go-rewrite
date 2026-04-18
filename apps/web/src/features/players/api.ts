import { backofficeRequest } from "@/lib/backoffice-api"

export type PlayerFilterOption = {
  id: number
  name: string
}

export type PlayerRecord = {
  id: number
  username: string
  extUsername: string
  tokoId: number
  tokoName: string
  ownerUsername: string
  createdAt: string
  updatedAt: string
}

export type PlayersListResponse = {
  data: PlayerRecord[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  filters: {
    tokos: PlayerFilterOption[]
  }
}

export type PlayerMoneyInfoResponse = {
  data: {
    playerId: number
    username: string
    extUsername: string
    tokoName: string
    balance: number
    checkedAt: string
  }
}

export type PlayersListParams = {
  search?: string
  tokoId?: string
  page: number
  perPage: number
}

export async function listPlayers(params: PlayersListParams) {
  const searchParams = new URLSearchParams()

  if (params.search) {
    searchParams.set("search", params.search)
  }
  if (params.tokoId) {
    searchParams.set("toko_id", params.tokoId)
  }

  searchParams.set("page", String(params.page))
  searchParams.set("per_page", String(params.perPage))

  return backofficeRequest<PlayersListResponse>(
    `/backoffice/api/players?${searchParams.toString()}`,
  )
}

export async function getPlayerMoneyInfo(playerId: number) {
  return backofficeRequest<PlayerMoneyInfoResponse>(
    `/backoffice/api/players/${playerId}/money-info`,
  )
}
