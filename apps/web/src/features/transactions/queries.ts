import { keepPreviousData, useQuery } from "@tanstack/react-query"

import {
  getTransactionDetail,
  listTransactions,
  type TransactionListParams,
} from "@/features/transactions/api"

export function useTransactionsQuery(params: TransactionListParams) {
  return useQuery({
    queryKey: ["backoffice", "transactions", params],
    queryFn: () => listTransactions(params),
    placeholderData: keepPreviousData,
  })
}

export function useTransactionDetailQuery(transactionID: number | null) {
  return useQuery({
    queryKey: ["backoffice", "transactions", "detail", transactionID],
    queryFn: () => getTransactionDetail(transactionID as number),
    enabled: transactionID !== null,
  })
}
