export type ProviderRecord = {
  code: string
  name: string
  status: string | number
}

export type GameRecord = {
  id: string | number
  game_code: string
  game_name: string
  banner?: string | null
  status?: string | number | null
}

export type ProviderListResponse = {
  success: boolean
  providers: ProviderRecord[]
}

export type GameListResponse = {
  success: boolean
  provider_code: string
  games: GameRecord[]
}

async function request<T>(path: string) {
  const response = await fetch(path, {
    credentials: "include",
    headers: {
      Accept: "application/json",
    },
  })

  if (!response.ok) {
    const payload =
      ((await response.json().catch(() => null)) as { message?: string } | null) ??
      null

    throw new Error(payload?.message ?? "Request failed")
  }

  return response.json() as Promise<T>
}

export async function getProviders() {
  return request<ProviderListResponse>("/backoffice/api/catalog/providers")
}

export async function getGames(providerCode: string, localized: boolean) {
  const params = new URLSearchParams({
    provider_code: providerCode,
    localized: String(localized),
  })

  return request<GameListResponse>(`/backoffice/api/catalog/games?${params.toString()}`)
}
