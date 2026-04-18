import { backofficeRequest } from "@/lib/backoffice-api"

export type ManagedPlayerOption = {
  id: number
  username: string
  userLabel: string
  tokoName: string
}

export type ActivePlayerRecord = {
  playerId: number
  username: string
  userLabel: string
  tokoName: string
  providerCode: string
  gameCode: string
  bet: string | number | null
  balance: string | number | null
  totalDebit: string | number | null
  totalCredit: string | number | null
  targetRtp: string | number | null
  realRtp: string | number | null
}

export type CallOption = {
  rtp: string | number | null
  callType: string
  callTypeValue: number
}

export type CallHistoryRecord = {
  id: string | number
  playerId: number
  username: string
  userLabel: string
  tokoName: string
  providerCode: string | number | null
  gameCode: string | number | null
  bet: string | number | null
  userPrev: string | number | null
  userAfter: string | number | null
  agentPrev: string | number | null
  agentAfter: string | number | null
  expect: string | number | null
  missed: string | number | null
  real: string | number | null
  rtp: string | number | null
  type: string | number | null
  status: string | number | null
  statusLabel: string
  canCancel: boolean
  createdAt: string | null
  updatedAt: string | null
}

export type CallManagementBootstrapResponse = {
  data: {
    managedPlayers: ManagedPlayerOption[]
  }
}

export type ActivePlayersResponse = {
  data: ActivePlayerRecord[]
}

export type CallListResponse = {
  data: CallOption[]
}

export type CallHistoryResponse = {
  data: CallHistoryRecord[]
}

export type CallMutationResponse = {
  message?: string
  data: {
    calledMoney?: string | number | null
    canceledMoney?: string | number | null
    changedRtp?: string | number | null
  }
}

export async function getCallManagementBootstrap() {
  return backofficeRequest<CallManagementBootstrapResponse>("/backoffice/api/call-management/bootstrap")
}

export async function getActivePlayers() {
  return backofficeRequest<ActivePlayersResponse>("/backoffice/api/call-management/active-players")
}

export async function getCallList(providerCode: string, gameCode: string) {
  return backofficeRequest<CallListResponse>("/backoffice/api/call-management/call-list", {
    method: "POST",
    body: JSON.stringify({
      providerCode,
      gameCode,
    }),
  })
}

export async function getCallHistory(offset = 0, limit = 100) {
  return backofficeRequest<CallHistoryResponse>("/backoffice/api/call-management/history", {
    method: "POST",
    body: JSON.stringify({
      offset,
      limit,
    }),
  })
}

export async function applyCall(payload: {
  playerId: number
  providerCode: string
  gameCode: string
  callRtp: number
  callTypeValue: number
}) {
  return backofficeRequest<CallMutationResponse>("/backoffice/api/call-management/apply", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function cancelCall(callId: number) {
  return backofficeRequest<CallMutationResponse>("/backoffice/api/call-management/cancel", {
    method: "POST",
    body: JSON.stringify({
      callId,
    }),
  })
}

export async function controlRtp(payload: {
  playerId: number
  providerCode: string
  rtp: number
}) {
  return backofficeRequest<CallMutationResponse>("/backoffice/api/call-management/control-rtp", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function controlUsersRtp(rtp: number) {
  return backofficeRequest<CallMutationResponse>("/backoffice/api/call-management/control-users-rtp", {
    method: "POST",
    body: JSON.stringify({
      rtp,
    }),
  })
}
