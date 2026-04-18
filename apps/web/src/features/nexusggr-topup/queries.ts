import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  checkTopupStatus,
  generateTopup,
  getTopupBootstrap,
  type TopupBootstrapResponse,
  type TopupMutationResponse,
} from "@/features/nexusggr-topup/api"

function topupBootstrapQueryKey(tokoId?: number | null) {
  return ["backoffice", "nexusggr-topup", "bootstrap", tokoId ?? null] as const
}

export function useTopupBootstrapQuery(tokoId?: number | null) {
  return useQuery({
    queryKey: topupBootstrapQueryKey(tokoId),
    queryFn: () => getTopupBootstrap(tokoId),
    staleTime: 5_000,
  })
}

function syncTopupBootstrapCache(
  previous: TopupBootstrapResponse | undefined,
  response: TopupMutationResponse,
): TopupBootstrapResponse {
  const previousData = previous?.data
  const nextPendingTopup =
    response.data.pendingTopup ?? (response.data.status === "pending" ? previousData?.pendingTopup ?? null : null)

  return {
    data: {
      tokos: previousData?.tokos ?? [],
      selectedToko: response.data.selectedToko ?? previousData?.selectedToko ?? null,
      topupRatio: response.data.topupRatio ?? previousData?.topupRatio ?? 0,
      pendingTopup:
        response.data.status && response.data.status !== "pending"
          ? null
          : nextPendingTopup,
    },
  }
}

export function useGenerateTopupMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      tokoId,
      amount,
    }: {
      tokoId: number
      amount: number
    }) => generateTopup(tokoId, amount),
    onSuccess: (response, variables) => {
      queryClient.setQueryData<TopupBootstrapResponse>(
        topupBootstrapQueryKey(null),
        (previous) => syncTopupBootstrapCache(previous, response),
      )
      queryClient.setQueryData<TopupBootstrapResponse>(
        topupBootstrapQueryKey(variables.tokoId),
        (previous) => syncTopupBootstrapCache(previous, response),
      )
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "nexusggr-topup", "bootstrap"],
      })
    },
  })
}

export function useCheckTopupStatusMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      tokoId,
      transactionCode,
    }: {
      tokoId: number
      transactionCode: string
    }) => checkTopupStatus(tokoId, transactionCode),
    onSuccess: (response, variables) => {
      queryClient.setQueryData<TopupBootstrapResponse>(
        topupBootstrapQueryKey(null),
        (previous) => syncTopupBootstrapCache(previous, response),
      )
      queryClient.setQueryData<TopupBootstrapResponse>(
        topupBootstrapQueryKey(variables.tokoId),
        (previous) => syncTopupBootstrapCache(previous, response),
      )
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "nexusggr-topup", "bootstrap"],
      })
    },
  })
}
