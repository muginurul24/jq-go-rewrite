import { backofficeRequest } from "@/lib/backoffice-api"

export type WithdrawalTokoOption = {
  id: number
  name: string
  ownerUsername: string
  settleBalance: number
}

export type WithdrawalBankOption = {
  id: number
  bankCode: string
  bankName: string
  accountNumber: string
  accountName: string
}

export type WithdrawalInquiry = {
  inquiryId: number
  accountName: string
  bankName: string
  accountNumber: string
  amount: number
  bankFee: number
  platformFee: number
  partnerRefNo: string
  currentSettleBalance: number
  estimatedTotalDeduction: number
  estimatedRemainingSettle: number
  finalTotalDeduction: number
  finalRemainingSettle: number
}

export type WithdrawalBootstrapResponse = {
  data: {
    tokos: WithdrawalTokoOption[]
    selectedToko?: WithdrawalTokoOption | null
    banks: WithdrawalBankOption[]
    feePercentage: number
    minimumAmount: number
  }
}

export type WithdrawalInquiryResponse = {
  message?: string
  data: {
    selectedToko: WithdrawalTokoOption | null
    selectedBank: WithdrawalBankOption | null
    inquiry: WithdrawalInquiry
  }
}

export type WithdrawalSubmitResponse = {
  message?: string
  data: {
    selectedToko: WithdrawalTokoOption | null
    selectedBank: WithdrawalBankOption | null
    inquiry: WithdrawalInquiry
    transaction: {
      id: number
      code: string
      status: string
      amount: number
    }
  }
}

export async function getWithdrawalBootstrap(tokoId?: number | null) {
  const searchParams = new URLSearchParams()
  if (tokoId != null) {
    searchParams.set("toko_id", String(tokoId))
  }

  const suffix = searchParams.toString()
  return backofficeRequest<WithdrawalBootstrapResponse>(
    `/backoffice/api/withdrawal/bootstrap${suffix ? `?${suffix}` : ""}`,
  )
}

export async function runWithdrawalInquiry(tokoId: number, bankId: number, amount: number) {
  return backofficeRequest<WithdrawalInquiryResponse>(
    "/backoffice/api/withdrawal/inquiry",
    {
      method: "POST",
      body: JSON.stringify({
        tokoId,
        bankId,
        amount,
      }),
    },
  )
}

export async function submitWithdrawal(tokoId: number, bankId: number, amount: number, inquiryId: number) {
  return backofficeRequest<WithdrawalSubmitResponse>(
    "/backoffice/api/withdrawal/submit",
    {
      method: "POST",
      body: JSON.stringify({
        tokoId,
        bankId,
        amount,
        inquiryId,
      }),
    },
  )
}
