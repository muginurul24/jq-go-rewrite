import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { useEffect } from "react"

import {
  bootstrapAuth,
  getCurrentUser,
  login,
  logout,
  register,
  verifyLoginMfa,
  type LoginMfaPayload,
  type LoginPayload,
  type RegisterPayload,
} from "@/features/auth/api"
import { setBackofficeCSRFToken } from "@/lib/backoffice-api"

export const authQueryKey = ["auth", "bootstrap"] as const

export function authBootstrapQueryOptions() {
  return queryOptions({
    queryKey: authQueryKey,
    queryFn: bootstrapAuth,
    staleTime: 30_000,
  })
}

export function useAuthBootstrap() {
  const query = useQuery(authBootstrapQueryOptions())
  const csrfToken = query.data?.csrfToken

  useEffect(() => {
    if (csrfToken !== undefined) {
      setBackofficeCSRFToken(csrfToken ?? "")
    }
  }, [csrfToken])

  return query
}

export function useRefreshCurrentUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: getCurrentUser,
    onSuccess: (response) => {
      queryClient.setQueryData(authQueryKey, {
        csrfToken: response.csrfToken ?? "",
        user: response.user ?? null,
        mfaPending: Boolean(response.requiresMfa),
      })
    },
  })
}

export function useLoginMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: LoginPayload) => login(payload),
    onSuccess: (response) => {
      queryClient.setQueryData(authQueryKey, {
        csrfToken: response.csrfToken ?? "",
        user: response.user ?? null,
        mfaPending: Boolean(response.requiresMfa),
      })
    },
  })
}

export function useVerifyLoginMfaMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: LoginMfaPayload) => verifyLoginMfa(payload),
    onSuccess: (response) => {
      queryClient.setQueryData(authQueryKey, {
        csrfToken: response.csrfToken ?? "",
        user: response.user ?? null,
        mfaPending: false,
      })
    },
  })
}

export function useRegisterMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: RegisterPayload) => register(payload),
    onSuccess: (response) => {
      queryClient.setQueryData(authQueryKey, {
        csrfToken: response.csrfToken ?? "",
        user: response.user,
      })
    },
  })
}

export function useLogoutMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: logout,
    onSuccess: (response) => {
      queryClient.setQueryData(authQueryKey, {
        csrfToken: response.csrfToken ?? "",
        user: null,
        mfaPending: false,
      })
    },
  })
}
