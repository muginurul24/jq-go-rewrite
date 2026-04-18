import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createUser,
  getUserDetail,
  listUsers,
  updateUser,
  type UserPayload,
  type UsersListParams,
} from "@/features/users/api"

export function useUsersQuery(params: UsersListParams, enabled = true) {
  return useQuery({
    queryKey: ["backoffice", "users", params],
    queryFn: () => listUsers(params),
    enabled,
    staleTime: 30_000,
  })
}

export function useUserDetailQuery(userId: number | null, enabled = true) {
  return useQuery({
    queryKey: ["backoffice", "users", "detail", userId],
    queryFn: () => getUserDetail(userId as number),
    enabled: enabled && userId != null,
    staleTime: 30_000,
  })
}

export function useCreateUserMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: UserPayload) => createUser(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["backoffice", "users"] })
    },
  })
}

export function useUpdateUserMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      userId,
      payload,
    }: {
      userId: number
      payload: UserPayload
    }) => updateUser(userId, payload),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({ queryKey: ["backoffice", "users"] })
      void queryClient.invalidateQueries({
        queryKey: ["backoffice", "users", "detail", variables.userId],
      })
    },
  })
}
