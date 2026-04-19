import { backofficeRequest } from "@/lib/backoffice-api"

export type DashboardOverviewResponse = {
  data: {
    generatedAt: string
    role: "dev" | "superadmin" | "admin" | "user"
    stats: {
      pendingBalance: number
      settleBalance: number
      nexusggrBalance: number
      platformIncome?: number | null
      externalQrPending?: number | null
      externalQrSettle?: number | null
      externalAgentBalance?: number | null
      externalAgentCode?: string | null
    }
    alertSummary: {
      unreadNotifications: number
      criticalNotifications: number
      pendingOverdueQris: number
      pendingWithdrawals: number
      lowSettleTokos: number
      lowNexusggrTokos: number
    }
    alerts: Array<{
      key: string
      severity: "info" | "success" | "warning" | "danger"
      title: string
      body: string
      count: number
      href: string
    }>
    recentTransactions: Array<{
      id: number
      code?: string | null
      tokoName: string
      player?: string | null
      category: "qris" | "nexusggr"
      type: "deposit" | "withdrawal"
      status: "pending" | "success" | "failed" | "expired"
      amount: number
      createdAt: string
    }>
  }
}

export type DashboardOperationalPulseResponse = {
  data: {
    generatedAt: string
    role: "dev" | "superadmin" | "admin" | "user"
    stats: {
      pendingTransactions: number
      failedTransactions7d: number
      successfulQris7d: number
      successfulNexusggr7d: number
    }
    qris: Array<{
      date: string
      deposit: number
      withdrawal: number
    }>
    nexusggr: Array<{
      date: string
      deposit: number
      withdrawal: number
    }>
  }
}

export function getDashboardOverview() {
  return backofficeRequest<DashboardOverviewResponse>(
    "/backoffice/api/dashboard/overview",
  )
}

export function getDashboardOperationalPulse() {
  return backofficeRequest<DashboardOperationalPulseResponse>(
    "/backoffice/api/dashboard/operational-pulse",
  )
}
