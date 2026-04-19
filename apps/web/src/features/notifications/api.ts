import { backofficeRequest } from "@/lib/backoffice-api"

export type NotificationRecord = {
  id: string
  type: string
  title: string
  body: string
  icon?: string | null
  iconColor?: "info" | "success" | "warning" | "danger" | null
  status?: "info" | "success" | "warning" | "danger" | null
  createdAt: string
  readAt?: string | null
  action?: {
    label: string
    url?: string | null
  } | null
}

export type NotificationsResponse = {
  data: NotificationRecord[]
  meta: {
    page: number
    perPage: number
    total: number
    totalPages: number
  }
  summary: {
    total: number
    unread: number
    unreadCritical: number
    unreadWarnings: number
    unreadSuccess: number
  }
}

export function getNotifications(scope: "all" | "unread", page = 1, perPage = 20) {
  const params = new URLSearchParams({
    scope,
    page: String(page),
    perPage: String(perPage),
  })

  return backofficeRequest<NotificationsResponse>(
    `/backoffice/api/notifications?${params.toString()}`,
  )
}

export function markNotificationRead(notificationId: string) {
  return backofficeRequest<{ message: string }>(
    `/backoffice/api/notifications/${notificationId}/read`,
    {
      method: "POST",
      body: JSON.stringify({}),
    },
  )
}

export function markAllNotificationsRead() {
  return backofficeRequest<{ message: string; updatedCount: number }>(
    "/backoffice/api/notifications/read-all",
    {
      method: "POST",
      body: JSON.stringify({}),
    },
  )
}
