import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { authQueryKey } from "@/features/auth/queries"
import { beginMfaSetup, confirmMfaSetup, disableMfa, getMfaStatus } from "@/features/mfa/api"

export const mfaStatusQueryKey = ["auth", "mfa"] as const

export function useMfaStatusQuery() {
  return useQuery({
    queryKey: mfaStatusQueryKey,
    queryFn: getMfaStatus,
    staleTime: 15_000,
  })
}

export function useBeginMfaSetupMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: beginMfaSetup,
    onSuccess: (response) => {
      queryClient.setQueryData(mfaStatusQueryKey, response)
    },
  })
}

export function useConfirmMfaSetupMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (code: string) => confirmMfaSetup(code),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: mfaStatusQueryKey })
      void queryClient.invalidateQueries({ queryKey: authQueryKey })
    },
  })
}

export function useDisableMfaMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (code: string) => disableMfa(code),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: mfaStatusQueryKey })
      void queryClient.invalidateQueries({ queryKey: authQueryKey })
    },
  })
}
