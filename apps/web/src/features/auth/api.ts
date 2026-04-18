import {
  backofficeRequest,
  isBackofficeRequestError,
  setBackofficeCSRFToken,
  type BackofficeErrorPayload,
} from "@/lib/backoffice-api"

export type AuthUser = {
  id: number
  username: string
  name: string
  email: string
  role: "dev" | "superadmin" | "admin" | "user"
  isActive?: boolean
  mfaEnabled?: boolean
}

export type AuthBootstrapResponse = {
  csrfToken: string
  user: AuthUser | null
  mfaPending?: boolean
}

export type AuthSuccessResponse = {
  csrfToken?: string
  user?: AuthUser
  requiresMfa?: boolean
}

export type AuthErrorResponse = {
  message: string
  errors?: Record<string, string>
}

export type LogoutResponse = {
  message: string
  csrfToken: string
}

export type LoginMfaPayload = {
  code: string
}

export type LoginPayload = {
  login: string
  password: string
  remember: boolean
}

export type RegisterPayload = {
  username: string
  name: string
  email: string
  password: string
  password_confirmation: string
}

export async function bootstrapAuth() {
  const response = await backofficeRequest<AuthBootstrapResponse>(
    "/backoffice/api/auth/bootstrap",
    { method: "GET" },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export async function login(payload: LoginPayload) {
  const response = await backofficeRequest<AuthSuccessResponse>(
    "/backoffice/api/auth/login",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export async function register(payload: RegisterPayload) {
  const response = await backofficeRequest<AuthSuccessResponse>(
    "/backoffice/api/auth/register",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export async function getCurrentUser() {
  const response = await backofficeRequest<AuthSuccessResponse>(
    "/backoffice/api/auth/me",
    { method: "GET" },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export async function logout() {
  const response = await backofficeRequest<LogoutResponse>(
    "/backoffice/api/auth/logout",
    {
      method: "POST",
      body: JSON.stringify({}),
    },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export async function verifyLoginMfa(payload: LoginMfaPayload) {
  const response = await backofficeRequest<AuthSuccessResponse>(
    "/backoffice/api/auth/mfa/login/verify",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  )

  setBackofficeCSRFToken(response.csrfToken)
  return response
}

export function isAuthRequestError(
  error: unknown,
): error is Error & { status: number; payload: BackofficeErrorPayload } {
  return isBackofficeRequestError(error)
}
