import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from "@/features/notifications/api"

export function useNotificationsQuery(
  scope: "all" | "unread",
  page = 1,
  perPage = 20,
) {
  return useQuery({
    queryKey: ["backoffice", "notifications", scope, page, perPage],
    queryFn: () => getNotifications(scope, page, perPage),
    refetchInterval: 30_000,
  })
}

export function useRecentNotificationsQuery() {
  return useNotificationsQuery("all", 1, 8)
}

export function useMarkNotificationReadMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: markNotificationRead,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["backoffice", "notifications"],
      })
      await queryClient.invalidateQueries({
        queryKey: ["backoffice", "dashboard", "overview"],
      })
    },
  })
}

export function useMarkAllNotificationsReadMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["backoffice", "notifications"],
      })
      await queryClient.invalidateQueries({
        queryKey: ["backoffice", "dashboard", "overview"],
      })
    },
  })
}
