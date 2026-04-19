import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  applyCall,
  cancelCall,
  controlRtp,
  controlUsersRtp,
  getActivePlayers,
  getCallHistory,
  getCallList,
  getCallManagementBootstrap,
} from "@/features/call-management/api"

export function useCallManagementBootstrapQuery() {
  return useQuery({
    queryKey: ["backoffice", "call-management", "bootstrap"],
    queryFn: getCallManagementBootstrap,
    staleTime: 60_000,
  })
}

export function useActivePlayersQuery() {
  return useQuery({
    queryKey: ["backoffice", "call-management", "active-players"],
    queryFn: getActivePlayers,
    refetchInterval: 5_000,
    staleTime: 10_000,
  })
}

export function useCallListQuery(providerCode: string, gameCode: string) {
  return useQuery({
    queryKey: ["backoffice", "call-management", "call-list", providerCode, gameCode],
    queryFn: () => getCallList(providerCode, gameCode),
    enabled: providerCode.length > 0 && gameCode.length > 0,
    staleTime: 30_000,
  })
}

export function useCallHistoryQuery() {
  return useQuery({
    queryKey: ["backoffice", "call-management", "history"],
    queryFn: () => getCallHistory(0, 100),
    refetchInterval: 10_000,
    staleTime: 5_000,
  })
}

export function useApplyCallMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: applyCall,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "call-management", "active-players"],
      })
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "call-management", "history"],
      })
    },
  })
}

export function useCancelCallMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: cancelCall,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "call-management", "history"],
      })
    },
  })
}

export function useControlRtpMutation() {
  return useMutation({
    mutationFn: controlRtp,
  })
}

export function useControlUsersRtpMutation() {
  return useMutation({
    mutationFn: controlUsersRtp,
  })
}
