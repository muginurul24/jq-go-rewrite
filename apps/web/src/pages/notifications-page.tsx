import { useState } from "react"

import { Link } from "@tanstack/react-router"
import { format } from "date-fns"
import {
  BellRingIcon,
  CheckCircle2Icon,
  Clock3Icon,
  DownloadIcon,
  StoreIcon,
  TriangleAlertIcon,
  XCircleIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  useMarkAllNotificationsReadMutation,
  useMarkNotificationReadMutation,
  useNotificationsQuery,
} from "@/features/notifications/queries"

function iconForNotification(icon?: string | null) {
  switch (icon) {
    case "heroicon-o-banknotes":
      return Clock3Icon
    case "heroicon-o-check-circle":
      return CheckCircle2Icon
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

function statusVariant(status?: string | null) {
  switch (status) {
    case "success":
      return "default" as const
    case "warning":
      return "secondary" as const
    case "danger":
      return "destructive" as const
    default:
      return "outline" as const
  }
}

export function NotificationsPage() {
  const [scope, setScope] = useState<"all" | "unread">("all")
  const [page, setPage] = useState(1)
  const notificationsQuery = useNotificationsQuery(scope, page, 20)
  const markReadMutation = useMarkNotificationReadMutation()
  const markAllMutation = useMarkAllNotificationsReadMutation()

  const payload = notificationsQuery.data

  return (
    <main className="flex flex-1 flex-col gap-6 px-4 py-4 pb-6 lg:px-6">
      <section className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Notification Center</CardTitle>
              <CardDescription>
                Database notifications parity dengan legacy, ditambah signal operasional yang memang berguna di production.
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Tabs
                value={scope}
                onValueChange={(value) => {
                  setScope(value as "all" | "unread")
                  setPage(1)
                }}
              >
                <TabsList className="rounded-xl">
                  <TabsTrigger value="all">All</TabsTrigger>
                  <TabsTrigger value="unread">Unread</TabsTrigger>
                </TabsList>
              </Tabs>
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={(payload?.summary.unread ?? 0) === 0 || markAllMutation.isPending}
                onClick={() => {
                  void markAllMutation.mutateAsync()
                }}
              >
                Mark all read
              </Button>
            </div>
          </CardHeader>
        </Card>

        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader>
            <CardTitle className="text-base">Signal Summary</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <div className="rounded-xl border border-border/60 p-3">
              <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Unread</p>
              <p className="mt-1 text-2xl font-semibold">{payload?.summary.unread ?? 0}</p>
            </div>
            <div className="rounded-xl border border-rose-500/20 bg-rose-500/8 p-3">
              <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Critical</p>
              <p className="mt-1 text-2xl font-semibold">{payload?.summary.unreadCritical ?? 0}</p>
            </div>
            <div className="rounded-xl border border-amber-500/20 bg-amber-500/8 p-3">
              <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Warnings</p>
              <p className="mt-1 text-2xl font-semibold">{payload?.summary.unreadWarnings ?? 0}</p>
            </div>
          </CardContent>
        </Card>
      </section>

      <section>
        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardContent className="p-4">
            {notificationsQuery.isLoading ? (
              <div className="grid gap-3">
                {Array.from({ length: 6 }, (_, index) => (
                  <div key={index} className="rounded-2xl border border-border/60 p-4">
                    <Skeleton className="h-4 w-44" />
                    <Skeleton className="mt-2 h-3 w-full" />
                    <Skeleton className="mt-1 h-3 w-3/4" />
                  </div>
                ))}
              </div>
            ) : payload && payload.data.length > 0 ? (
              <div className="grid gap-3">
                {payload.data.map((notification) => {
                  const Icon = iconForNotification(notification.icon)

                  return (
                    <div
                      key={notification.id}
                      className="rounded-2xl border border-border/60 bg-background/70 p-4"
                    >
                      <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
                        <div className="flex size-11 shrink-0 items-center justify-center rounded-2xl border border-border/60 bg-muted/60">
                          <Icon className="size-5" />
                        </div>
                        <div className="min-w-0 flex-1 space-y-2">
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <div className="space-y-1">
                              <div className="flex flex-wrap items-center gap-2">
                                <h3 className="text-sm font-semibold">{notification.title}</h3>
                                <Badge variant={statusVariant(notification.status)} className="rounded-full">
                                  {notification.status ?? "info"}
                                </Badge>
                                {!notification.readAt ? (
                                  <Badge variant="outline" className="rounded-full">
                                    unread
                                  </Badge>
                                ) : null}
                              </div>
                              <p className="text-sm leading-6 text-muted-foreground">
                                {notification.body}
                              </p>
                            </div>
                            <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                              {format(new Date(notification.createdAt), "dd MMM yyyy HH:mm")}
                            </p>
                          </div>
                          <div className="flex flex-wrap items-center gap-2">
                            {notification.action?.url ? (
                              <Button asChild variant="outline" size="sm" className="rounded-xl">
                                <Link
                                  to={notification.action.url}
                                  onClick={() => {
                                    if (!notification.readAt) {
                                      void markReadMutation.mutateAsync(notification.id)
                                    }
                                  }}
                                >
                                  {notification.action.label}
                                </Link>
                              </Button>
                            ) : null}
                            {!notification.readAt ? (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="rounded-xl"
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
              <div className="rounded-2xl border border-dashed border-border/60 px-6 py-12 text-center">
                <p className="text-sm font-medium">Belum ada notifikasi untuk filter ini</p>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  Notification center akan terisi saat ada event toko, callback, atau operasional yang perlu Anda tindak lanjuti.
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </section>

      {payload ? (
        <section className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            Menampilkan halaman {payload.meta.page} dari {Math.max(payload.meta.totalPages, 1)}.
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              className="rounded-xl"
              disabled={payload.meta.page <= 1}
              onClick={() => {
                setPage((current) => Math.max(current - 1, 1))
              }}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              className="rounded-xl"
              disabled={payload.meta.page >= payload.meta.totalPages}
              onClick={() => {
                setPage((current) => current + 1)
              }}
            >
              Next
            </Button>
          </div>
        </section>
      ) : null}
    </main>
  )
}
