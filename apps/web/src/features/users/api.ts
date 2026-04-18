import { backofficeRequest } from "@/lib/backoffice-api"

export type UserRecord = {
  id: number
  username: string
  name: string
  email: string
  role: "dev" | "superadmin" | "admin" | "user"
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export type UsersListParams = {
  search?: string
  role?: string
  status?: string
  page: number
  perPage: number
}

export type UsersListResponse = {
  data: UserRecord[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  summary: {
    totalUsers: number
    activeUsers: number
    adminUsers: number
    endUsers: number
  }
}

export type UserDetailResponse = {
  data: UserRecord
}

export type UserMutationResponse = {
  message: string
  data: UserRecord
}

export type UserPayload = {
  username: string
  name: string
  email: string
  role: string
  isActive: boolean
  password?: string
}

export async function listUsers(params: UsersListParams) {
  const searchParams = new URLSearchParams()

  if (params.search) {
    searchParams.set("search", params.search)
  }
  if (params.role) {
    searchParams.set("role", params.role)
  }
  if (params.status) {
    searchParams.set("status", params.status)
  }

  searchParams.set("page", String(params.page))
  searchParams.set("per_page", String(params.perPage))

  return backofficeRequest<UsersListResponse>(
    `/backoffice/api/users?${searchParams.toString()}`,
  )
}

export async function getUserDetail(userId: number) {
  return backofficeRequest<UserDetailResponse>(`/backoffice/api/users/${userId}`)
}

export async function createUser(payload: UserPayload) {
  return backofficeRequest<UserMutationResponse>("/backoffice/api/users", {
    method: "POST",
    body: JSON.stringify(payload),
  })
}

export async function updateUser(userId: number, payload: UserPayload) {
  return backofficeRequest<UserMutationResponse>(`/backoffice/api/users/${userId}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
  })
}
