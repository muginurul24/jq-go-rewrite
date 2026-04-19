import { Link } from "@tanstack/react-router"
import { formatDistanceToNow } from "date-fns"
import {
  BellRingIcon,
  BanknoteIcon,
  CheckCircle2Icon,
  Clock3Icon,
  DownloadIcon,
  StoreIcon,
  TriangleAlertIcon,
  XCircleIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useMarkAllNotificationsReadMutation,
  useMarkNotificationReadMutation,
  useRecentNotificationsQuery,
} from "@/features/notifications/queries"

function iconForNotification(icon?: string | null) {
  switch (icon) {
    case "heroicon-o-banknotes":
      return BanknoteIcon
    case "heroicon-o-check-circle":
      return CheckCircle2Icon
    case "heroicon-o-clock":
      return Clock3Icon
    case "heroicon-o-arrow-down-tray":
      return DownloadIcon
    case "heroicon-o-building-storefront":
      return StoreIcon
    case "heroicon-o-x-circle":
      return XCircleIcon
    case "heroicon-o-exclamation-triangle":
      return TriangleAlertIcon
    default:
      return BellRingIcon
  }
}

function accentClass(status?: string | null) {
  switch (status) {
    case "success":
      return "border-emerald-500/20 bg-emerald-500/8 text-emerald-600 dark:text-emerald-300"
    case "warning":
      return "border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300"
    case "danger":
      return "border-rose-500/20 bg-rose-500/10 text-rose-700 dark:text-rose-300"
    default:
      return "border-sky-500/15 bg-sky-500/8 text-sky-700 dark:text-sky-300"
  }
}

export function NotificationCenter() {
  const notificationsQuery = useRecentNotificationsQuery()
  const markReadMutation = useMarkNotificationReadMutation()
  const markAllMutation = useMarkAllNotificationsReadMutation()

  const unreadCount = notificationsQuery.data?.summary.unread ?? 0
  const notifications = notificationsQuery.data?.data ?? []

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="icon"
          className="relative size-9 rounded-full"
          data-testid="notification-center-trigger"
        >
          <BellRingIcon className="size-4" />
          {unreadCount > 0 ? (
            <span className="absolute -top-1 -right-1 inline-flex min-w-5 items-center justify-center rounded-full bg-rose-500 px-1.5 text-[10px] font-semibold text-white">
              {unreadCount > 99 ? "99+" : unreadCount}
            </span>
          ) : null}
          <span className="sr-only">Notifications</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        className="w-[22rem] max-w-[calc(100vw-2rem)] rounded-2xl p-0 sm:w-[26rem]"
      >
        <div className="flex items-center justify-between gap-3 px-4 py-3">
          <div>
            <p className="text-sm font-semibold">Notifications</p>
            <p className="text-xs text-muted-foreground">
              {unreadCount > 0
                ? `${unreadCount} unread signal`
                : "Tidak ada signal baru"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-8 rounded-lg px-2.5 text-xs"
              disabled={unreadCount === 0 || markAllMutation.isPending}
              onClick={() => {
                void markAllMutation.mutateAsync()
              }}
            >
              Mark all read
            </Button>
            <Button asChild variant="outline" size="sm" className="h-8 rounded-lg px-2.5 text-xs">
              <Link to="/backoffice/notifications">View all</Link>
            </Button>
          </div>
        </div>
        <Separator />
        <div className="max-h-[26rem] overflow-y-auto p-3">
          {notificationsQuery.isLoading ? (
            <div className="grid gap-3">
              {Array.from({ length: 4 }, (_, index) => (
                <div key={index} className="rounded-xl border border-border/60 p-3">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="mt-2 h-3 w-full" />
                  <Skeleton className="mt-1 h-3 w-4/5" />
                </div>
              ))}
            </div>
          ) : notifications.length > 0 ? (
            <div className="grid gap-3">
              {notifications.map((notification) => {
                const Icon = iconForNotification(notification.icon)
                const isUnread = !notification.readAt

                return (
                  <div
                    key={notification.id}
                    className={`rounded-xl border p-3 ${accentClass(notification.status)}`}
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl border border-current/10 bg-background/80">
                        <Icon className="size-4" />
                      </div>
                      <div className="min-w-0 flex-1 space-y-1">
                        <div className="flex items-start justify-between gap-3">
                          <p className="text-sm font-semibold leading-5">
                            {notification.title}
                          </p>
                          {isUnread ? (
                            <span className="inline-flex size-2.5 shrink-0 rounded-full bg-current" />
                          ) : null}
                        </div>
                        <p className="text-xs leading-5 text-muted-foreground">
                          {notification.body}
                        </p>
                        <div className="flex flex-wrap items-center gap-2 pt-1">
                          <span className="text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
                            {formatDistanceToNow(new Date(notification.createdAt), {
                              addSuffix: true,
                            })}
                          </span>
                          {notification.action?.url ? (
                            <Button asChild variant="link" size="sm" className="h-auto px-0 text-xs">
                              <Link
                                to={notification.action.url}
                                onClick={() => {
                                  if (isUnread) {
                                    void markReadMutation.mutateAsync(notification.id)
                                  }
                                }}
                              >
                                {notification.action.label}
                              </Link>
                            </Button>
                          ) : null}
                          {isUnread ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-auto px-0 text-xs"
                              onClick={() => {
                                void markReadMutation.mutateAsync(notification.id)
                              }}
                            >
                              Mark read
                            </Button>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="rounded-xl border border-dashed border-border/60 px-4 py-8 text-center">
              <p className="text-sm font-medium">Notification inbox bersih</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Notifikasi operasional baru akan muncul di sini saat ada event penting.
              </p>
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
