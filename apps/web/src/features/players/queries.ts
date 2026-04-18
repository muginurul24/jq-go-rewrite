import { useQuery } from "@tanstack/react-query"

import {
  getPlayerMoneyInfo,
  listPlayers,
  type PlayersListParams,
} from "@/features/players/api"

export function usePlayersQuery(params: PlayersListParams) {
  return useQuery({
    queryKey: ["backoffice", "players", params],
    queryFn: () => listPlayers(params),
    staleTime: 30_000,
  })
}

export function usePlayerMoneyInfoQuery(playerId: number | null, enabled: boolean) {
  return useQuery({
    queryKey: ["backoffice", "players", "money-info", playerId],
    queryFn: () => getPlayerMoneyInfo(playerId as number),
    enabled: enabled && playerId != null,
    staleTime: 0,
  })
}
