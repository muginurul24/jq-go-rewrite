import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createToko,
  getTokoDetail,
  listTokos,
  regenerateTokoToken,
  updateToko,
  type TokosListParams,
  type TokoPayload,
} from "@/features/tokos/api"

export function useTokosQuery(params: TokosListParams) {
  return useQuery({
    queryKey: ["backoffice", "tokos", params],
    queryFn: () => listTokos(params),
    staleTime: 30_000,
  })
}

export function useTokoDetailQuery(tokoId: number | null) {
  return useQuery({
    queryKey: ["backoffice", "tokos", "detail", tokoId],
    queryFn: () => getTokoDetail(tokoId as number),
    enabled: tokoId != null,
    staleTime: 30_000,
  })
}

export function useCreateTokoMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: TokoPayload) => createToko(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["backoffice", "tokos"] })
    },
  })
}

export function useUpdateTokoMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      tokoId,
      payload,
    }: {
      tokoId: number
      payload: TokoPayload
    }) => updateToko(tokoId, payload),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({ queryKey: ["backoffice", "tokos"] })
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "tokos", "detail", variables.tokoId],
      })
    },
  })
}

export function useRegenerateTokoTokenMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (tokoId: number) => regenerateTokoToken(tokoId),
    onSuccess: (_, tokoId) => {
      void queryClient.invalidateQueries({ queryKey: ["backoffice", "tokos"] })
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "tokos", "detail", tokoId],
      })
    },
  })
}
