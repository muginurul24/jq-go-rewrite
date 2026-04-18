import { useQuery } from "@tanstack/react-query"

import { getGames, getProviders } from "@/features/catalog/api"

export function useProvidersQuery() {
  return useQuery({
    queryKey: ["backoffice", "catalog", "providers"],
    queryFn: getProviders,
    staleTime: 60_000,
  })
}

export function useGamesQuery(providerCode: string, localized: boolean) {
  return useQuery({
    queryKey: ["backoffice", "catalog", "games", providerCode, localized],
    queryFn: () => getGames(providerCode, localized),
    enabled: providerCode.length > 0,
    staleTime: 60_000,
  })
}
