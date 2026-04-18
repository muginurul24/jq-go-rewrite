import { backofficeRequest } from "@/lib/backoffice-api"

export type BankOwnerOption = {
  id: number
  username: string
  name: string
}

export type BankRecord = {
  id: number
  userId: number
  ownerUsername: string
  ownerName: string
  bankCode: string
  bankName: string
  accountNumber: string
  accountName: string
  createdAt: string
  updatedAt: string
}

export type BanksResponse = {
  data: BankRecord[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  filters: {
    owners: BankOwnerOption[]
  }
}

export type BankMutationResponse = {
  message?: string
  data: BankRecord
}

export type BankInquiryResponse = {
  message?: string
  data: {
    bankCode: string
    bankName: string
    accountNumber: string
    accountName: string
  }
}

export type BankListQuery = {
  search?: string
  ownerId?: string
  page?: number
  perPage?: number
}

export async function getBanks(query: BankListQuery) {
  const searchParams = new URLSearchParams()
  if (query.search) {
    searchParams.set("search", query.search)
  }
  if (query.ownerId) {
    searchParams.set("owner_id", query.ownerId)
  }
  if (query.page) {
    searchParams.set("page", String(query.page))
  }
  if (query.perPage) {
    searchParams.set("per_page", String(query.perPage))
  }

  const suffix = searchParams.toString()
  return backofficeRequest<BanksResponse>(
    `/backoffice/api/banks${suffix ? `?${suffix}` : ""}`,
  )
}

export async function createBank(payload: {
  userId?: number
  bankCode: string
  accountNumber: string
  accountName: string
}) {
  return backofficeRequest<BankMutationResponse>("/backoffice/api/banks", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function updateBank(
  bankId: number,
  payload: {
    userId?: number
    bankCode: string
    accountNumber: string
    accountName: string
  },
) {
  return backofficeRequest<BankMutationResponse>(`/backoffice/api/banks/${bankId}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  })
}

export async function inquiryBank(payload: {
  userId?: number
  bankCode: string
  accountNumber: string
}) {
  return backofficeRequest<BankInquiryResponse>("/backoffice/api/banks/inquiry", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}
