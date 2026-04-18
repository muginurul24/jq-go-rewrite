export type TransactionFilterOption = {
  id: number
  name: string
  ownerUsername: string
}

export type TransactionListItem = {
  id: number
  tokoId: number
  tokoName: string
  ownerUsername: string
  player?: string | null
  externalPlayer?: string | null
  category: string
  categoryLabel: string
  type: string
  typeLabel: string
  status: string
  statusLabel: string
  amount: number
  code?: string | null
  noteSummary?: string | null
  createdAt: string
  updatedAt: string
  canEdit: boolean
  canDelete: boolean
}

export type TransactionDetail = TransactionListItem & {
  notePayload: string
}

export type TransactionListResponse = {
  data: TransactionListItem[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  summary: {
    totalAmount: number
  }
  filters: {
    tokos: TransactionFilterOption[]
  }
}

export type TransactionDetailResponse = {
  data: TransactionDetail
}

export type TransactionListParams = {
  search?: string
  category?: string
  type?: string
  status?: string
  tokoId?: string
  dateFrom?: string
  dateUntil?: string
  amountMin?: string
  amountMax?: string
  page: number
  perPage: number
}

export type TransactionExportFormat = "csv" | "xlsx"

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

    const error = new Error(payload?.message ?? "Request failed") as Error & {
      status: number
    }
    error.status = response.status
    throw error
  }

  return response.json() as Promise<T>
}

export async function listTransactions(params: TransactionListParams) {
  const searchParams = new URLSearchParams()

  if (params.search) {
    searchParams.set("search", params.search)
  }
  if (params.category) {
    searchParams.set("categories", params.category)
  }
  if (params.type) {
    searchParams.set("types", params.type)
  }
  if (params.status) {
    searchParams.set("statuses", params.status)
  }
  if (params.tokoId) {
    searchParams.set("toko_ids", params.tokoId)
  }
  if (params.dateFrom) {
    searchParams.set("date_from", params.dateFrom)
  }
  if (params.dateUntil) {
    searchParams.set("date_until", params.dateUntil)
  }
  if (params.amountMin) {
    searchParams.set("amount_min", params.amountMin)
  }
  if (params.amountMax) {
    searchParams.set("amount_max", params.amountMax)
  }

  searchParams.set("page", String(params.page))
  searchParams.set("per_page", String(params.perPage))

  return request<TransactionListResponse>(
    `/backoffice/api/transactions?${searchParams.toString()}`,
  )
}

export async function getTransactionDetail(transactionID: number) {
  return request<TransactionDetailResponse>(
    `/backoffice/api/transactions/${transactionID}`,
  )
}

export async function downloadTransactionsExport(
  params: TransactionListParams,
  format: TransactionExportFormat,
) {
  const searchParams = new URLSearchParams()

  if (params.search) {
    searchParams.set("search", params.search)
  }
  if (params.category) {
    searchParams.set("categories", params.category)
  }
  if (params.type) {
    searchParams.set("types", params.type)
  }
  if (params.status) {
    searchParams.set("statuses", params.status)
  }
  if (params.tokoId) {
    searchParams.set("toko_ids", params.tokoId)
  }
  if (params.dateFrom) {
    searchParams.set("date_from", params.dateFrom)
  }
  if (params.dateUntil) {
    searchParams.set("date_until", params.dateUntil)
  }
  if (params.amountMin) {
    searchParams.set("amount_min", params.amountMin)
  }
  if (params.amountMax) {
    searchParams.set("amount_max", params.amountMax)
  }

  searchParams.set("format", format)

  const response = await fetch(
    `/backoffice/api/transactions/export?${searchParams.toString()}`,
    {
      credentials: "include",
      headers: {
        Accept:
          format === "xlsx"
            ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            : "text/csv",
      },
    },
  )

  if (!response.ok) {
    const payload =
      ((await response.json().catch(() => null)) as { message?: string } | null) ??
      null

    const error = new Error(payload?.message ?? "Export failed") as Error & {
      status: number
    }
    error.status = response.status
    throw error
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement("a")
  const disposition = response.headers.get("Content-Disposition")
  const filenameMatch = disposition?.match(/filename="?([^";]+)"?/)
  anchor.href = url
  anchor.download = filenameMatch?.[1] ?? `transactions.${format}`
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
