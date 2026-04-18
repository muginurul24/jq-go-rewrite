import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  getWithdrawalBootstrap,
  runWithdrawalInquiry,
  submitWithdrawal,
} from "@/features/withdrawal/api"

export function useWithdrawalBootstrapQuery(tokoId?: number | null) {
  return useQuery({
    queryKey: ["backoffice", "withdrawal", "bootstrap", tokoId ?? null],
    queryFn: () => getWithdrawalBootstrap(tokoId),
    staleTime: 5_000,
  })
}

export function useWithdrawalInquiryMutation() {
  return useMutation({
    mutationFn: ({
      tokoId,
      bankId,
      amount,
    }: {
      tokoId: number
      bankId: number
      amount: number
    }) => runWithdrawalInquiry(tokoId, bankId, amount),
  })
}

export function useWithdrawalSubmitMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      tokoId,
      bankId,
      amount,
      inquiryId,
    }: {
      tokoId: number
      bankId: number
      amount: number
      inquiryId: number
    }) => submitWithdrawal(tokoId, bankId, amount, inquiryId),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "withdrawal", "bootstrap", variables.tokoId],
      })
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "transactions"],
      })
    },
  })
}
