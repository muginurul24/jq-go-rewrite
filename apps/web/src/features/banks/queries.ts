import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createBank,
  getBanks,
  inquiryBank,
  type BankListQuery,
  updateBank,
} from "@/features/banks/api"

export function useBanksQuery(query: BankListQuery) {
  return useQuery({
    queryKey: ["backoffice", "banks", query],
    queryFn: () => getBanks(query),
    staleTime: 5_000,
  })
}

export function useCreateBankMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createBank,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "banks"],
      })
    },
  })
}

export function useUpdateBankMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      bankId,
      payload,
    }: {
      bankId: number
      payload: Parameters<typeof updateBank>[1]
    }) => updateBank(bankId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "banks"],
      })
    },
  })
}

export function useInquiryBankMutation() {
  return useMutation({
    mutationFn: inquiryBank,
  })
}
