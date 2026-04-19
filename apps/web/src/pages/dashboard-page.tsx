import { Link } from "@tanstack/react-router"
import { format } from "date-fns"
import { motion } from "framer-motion"
import {
  ArrowRightLeftIcon,
  BadgeDollarSignIcon,
  QrCodeIcon,
  ShieldCheckIcon,
  WalletCardsIcon,
} from "lucide-react"

import { MatrixBackdrop } from "@/components/matrix-backdrop"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useDashboardOverviewQuery } from "@/features/dashboard/queries"

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

const statusVariant: Record<
  "pending" | "success" | "failed" | "expired",
  "secondary" | "default" | "destructive" | "outline"
> = {
  pending: "secondary",
  success: "default",
  failed: "destructive",
  expired: "outline",
}

const alertVariant: Record<
  "info" | "success" | "warning" | "danger",
  "outline" | "default" | "secondary" | "destructive"
> = {
  info: "outline",
  success: "default",
  warning: "secondary",
  danger: "destructive",
}

export function DashboardPage() {
  const overviewQuery = useDashboardOverviewQuery()
  const overview = overviewQuery.data?.data

  const cards = [
    {
      title: "Pending Balance",
      value: overview?.stats.pendingBalance ?? 0,
      description: "QRIS pending balance lokal sesuai scope actor.",
      icon: ArrowRightLeftIcon,
    },
    {
      title: "Settle Balance",
      value: overview?.stats.settleBalance ?? 0,
      description: "Saldo settle yang dipakai untuk withdrawal dan pencairan.",
      icon: ShieldCheckIcon,
    },
    {
      title: "NexusGGR Balance",
      value: overview?.stats.nexusggrBalance ?? 0,
      description: "Pool operasional lokal untuk deposit dan withdrawal player.",
      icon: WalletCardsIcon,
    },
    ...(overview?.stats.platformIncome != null
      ? [
          {
            title: "Platform Income",
            value: overview.stats.platformIncome,
            description: "Akumulasi komisi fee transaksi dan withdrawal.",
            icon: BadgeDollarSignIcon,
          },
        ]
      : []),
  ]

  const devExternalCards = overview?.role === "dev"
    ? [
        {
          title: "External QR Pending",
          value: overview.stats.externalQrPending ?? 0,
          description: "Snapshot QR pending dari provider, cache 300 detik.",
          icon: QrCodeIcon,
        },
        {
          title: "External QR Settle",
          value: overview.stats.externalQrSettle ?? 0,
          description: "Snapshot QR settle dari provider, cache 300 detik.",
          icon: ShieldCheckIcon,
        },
        {
          title: "Agent Balance",
          value: overview.stats.externalAgentBalance ?? 0,
          description:
            overview.stats.externalAgentCode != null
              ? `Agent ${overview.stats.externalAgentCode} dari upstream NexusGGR.`
              : "Snapshot agent balance dari upstream NexusGGR.",
          icon: WalletCardsIcon,
        },
      ]
    : []

  return (
    <main className="flex flex-1 flex-col gap-6 pb-6">
      <motion.section
        className="px-4 pt-4 lg:px-6"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, ease: "easeOut" }}
      >
        <div className="relative overflow-hidden rounded-[2rem] border border-border/70 bg-card/90 p-6 shadow-sm backdrop-blur lg:p-8">
          <MatrixBackdrop />
          <div className="relative z-10 grid gap-8 xl:grid-cols-[minmax(0,1.5fr)_minmax(0,28rem)] xl:items-end">
            <div className="space-y-4">
              <div className="flex flex-wrap items-center gap-2">
                <Badge className="rounded-full bg-emerald-500/12 px-3 text-emerald-700 hover:bg-emerald-500/12 dark:text-emerald-300">
                  Overview
                </Badge>
                <Badge variant="outline" className="rounded-full px-3">
                  Live parity data
                </Badge>
              </div>
              <div className="max-w-3xl space-y-3">
                <h2 className="text-3xl font-semibold tracking-tight text-balance lg:text-5xl">
                  Dashboard overview sekarang memakai data nyata yang searah
                  dengan widget legacy.
                </h2>
                <p className="max-w-2xl text-sm leading-6 text-muted-foreground lg:text-base">
                  Balance pending, settle, NexusGGR, platform income, external
                  QR balance, dan agent balance tidak lagi dummy. Semua diambil
                  dari query lokal atau upstream yang sama dengan legacy.
                </p>
              </div>
            </div>
            <div className="grid min-w-0 gap-3 rounded-[1.5rem] border border-border/60 bg-background/75 p-4 backdrop-blur">
              <div className="min-w-0 rounded-2xl border border-emerald-500/15 bg-emerald-500/8 px-4 py-3">
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">
                  generated at
                </p>
                <p className="mt-1 break-words text-sm leading-6 font-semibold sm:text-base">
                  {overview?.generatedAt
                    ? format(new Date(overview.generatedAt), "dd MMM yyyy HH:mm:ss")
                    : "Loading..."}
                </p>
              </div>
              <div className="min-w-0 rounded-2xl border border-sky-500/15 bg-sky-500/8 px-4 py-3">
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">
                  role scope
                </p>
                <p className="mt-1 break-words text-sm leading-6 font-semibold capitalize sm:text-base">
                  {overview?.role ?? "guest"}
                </p>
              </div>
              <div className="flex flex-col gap-3 pt-1 2xl:flex-row">
                <Button asChild className="w-full rounded-xl 2xl:flex-1">
                  <Link to="/backoffice/operational-pulse">
                    Open operational pulse
                  </Link>
                </Button>
                <Button asChild variant="outline" className="w-full rounded-xl 2xl:flex-1">
                  <Link to="/backoffice/transactions">
                    Open transactions
                  </Link>
                </Button>
              </div>
            </div>
          </div>
        </div>
      </motion.section>

      <motion.section
        className="grid gap-4 px-4 lg:grid-cols-2 xl:grid-cols-4 lg:px-6"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, delay: 0.08, ease: "easeOut" }}
      >
        {cards.map((card) => (
          <Card key={card.title} className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
            <CardHeader className="space-y-4">
              <div className="flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <card.icon className="size-5" />
              </div>
              <div>
                <CardDescription>{card.title}</CardDescription>
                {overviewQuery.isLoading ? (
                  <Skeleton className="mt-2 h-9 w-40" />
                ) : (
                  <CardTitle className="mt-2 text-2xl">
                    {currencyFormatter.format(card.value)}
                  </CardTitle>
                )}
              </div>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              {card.description}
            </CardContent>
          </Card>
        ))}
      </motion.section>

      {devExternalCards.length > 0 ? (
        <motion.section
          className="grid gap-4 px-4 lg:grid-cols-3 lg:px-6"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.45, delay: 0.14, ease: "easeOut" }}
        >
          {devExternalCards.map((card) => (
            <Card key={card.title} className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
              <CardHeader className="space-y-4">
                <div className="flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                  <card.icon className="size-5" />
                </div>
                <div>
                  <CardDescription>{card.title}</CardDescription>
                  {overviewQuery.isLoading ? (
                    <Skeleton className="mt-2 h-9 w-40" />
                  ) : (
                    <CardTitle className="mt-2 text-2xl">
                      {currencyFormatter.format(card.value)}
                    </CardTitle>
                  )}
                </div>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                {card.description}
              </CardContent>
            </Card>
          ))}
        </motion.section>
      ) : null}

      <motion.section
        className="grid gap-4 px-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)] lg:px-6"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, delay: 0.17, ease: "easeOut" }}
      >
        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>Operational Alerts</CardTitle>
              <CardDescription>
                Ringkasan anomaly state dan notifikasi yang paling perlu ditindak.
              </CardDescription>
            </div>
            <Button asChild variant="outline" className="rounded-xl">
              <Link to="/backoffice/notifications">Open notification center</Link>
            </Button>
          </CardHeader>
          <CardContent className="grid gap-4">
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              {[
                {
                  title: "Unread",
                  value: overview?.alertSummary.unreadNotifications ?? 0,
                  description: "Database notifications belum dibaca.",
                },
                {
                  title: "Critical",
                  value: overview?.alertSummary.criticalNotifications ?? 0,
                  description: "Signal warning dan danger belum dibaca.",
                },
                {
                  title: "Overdue QRIS",
                  value: overview?.alertSummary.pendingOverdueQris ?? 0,
                  description: "Pending deposit QRIS lebih dari 30 menit.",
                },
                {
                  title: "Pending WD",
                  value: overview?.alertSummary.pendingWithdrawals ?? 0,
                  description: "Withdrawal menunggu callback disbursement.",
                },
              ].map((item) => (
                <div key={item.title} className="rounded-2xl border border-border/60 bg-background/70 p-4">
                  <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">
                    {item.title}
                  </p>
                  {overviewQuery.isLoading ? (
                    <Skeleton className="mt-2 h-8 w-20" />
                  ) : (
                    <p className="mt-2 text-3xl font-semibold tracking-tight">
                      {item.value}
                    </p>
                  )}
                  <p className="mt-2 text-xs leading-5 text-muted-foreground">
                    {item.description}
                  </p>
                </div>
              ))}
            </div>
            <div className="grid gap-3">
              {overviewQuery.isLoading ? (
                Array.from({ length: 3 }, (_, index) => (
                  <div key={index} className="rounded-2xl border border-border/60 p-4">
                    <Skeleton className="h-4 w-44" />
                    <Skeleton className="mt-2 h-3 w-full" />
                  </div>
                ))
              ) : overview?.alerts.length ? (
                overview.alerts.map((alert) => (
                  <div key={alert.key} className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="min-w-0 space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="text-sm font-semibold">{alert.title}</p>
                          <Badge variant={alertVariant[alert.severity]} className="rounded-full">
                            {alert.count}
                          </Badge>
                        </div>
                        <p className="text-sm leading-6 text-muted-foreground">
                          {alert.body}
                        </p>
                      </div>
                      <Button asChild variant="outline" size="sm" className="rounded-xl">
                        <Link to={alert.href}>Open</Link>
                      </Button>
                    </div>
                  </div>
                ))
              ) : (
                <div className="rounded-2xl border border-dashed border-border/60 px-4 py-8 text-center">
                  <p className="text-sm font-medium">Tidak ada alert aktif</p>
                  <p className="mt-1 text-sm leading-6 text-muted-foreground">
                    Dashboard saat ini bersih dari signal yang perlu aksi cepat.
                  </p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader>
            <CardTitle>Low Balance Watch</CardTitle>
            <CardDescription>
              Deteksi toko aktif dengan pool settle atau NexusGGR di bawah ambang operasional.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3">
            {[
              {
                title: "Low settle tokos",
                value: overview?.alertSummary.lowSettleTokos ?? 0,
                description: "Toko aktif dengan settle di bawah Rp 100.000.",
              },
              {
                title: "Low NexusGGR tokos",
                value: overview?.alertSummary.lowNexusggrTokos ?? 0,
                description: "Toko aktif dengan NexusGGR di bawah Rp 100.000.",
              },
            ].map((item) => (
              <div key={item.title} className="rounded-2xl border border-border/60 bg-background/70 p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">
                  {item.title}
                </p>
                {overviewQuery.isLoading ? (
                  <Skeleton className="mt-2 h-8 w-20" />
                ) : (
                  <p className="mt-2 text-3xl font-semibold tracking-tight">{item.value}</p>
                )}
                <p className="mt-2 text-xs leading-5 text-muted-foreground">
                  {item.description}
                </p>
              </div>
            ))}
            <Button asChild variant="outline" className="rounded-xl">
              <Link to="/backoffice/tokos">Review toko balances</Link>
            </Button>
          </CardContent>
        </Card>
      </motion.section>

      <motion.section
        className="px-4 lg:px-6"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, delay: 0.2, ease: "easeOut" }}
      >
        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>Recent transactions</CardTitle>
              <CardDescription>
                Snapshot transaksi terbaru berdasarkan scope actor, bukan dummy.
              </CardDescription>
            </div>
            <Button asChild variant="outline" className="rounded-xl">
              <Link to="/backoffice/transactions">Buka transaksi lengkap</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto rounded-2xl border border-border/60">
              <Table className="min-w-[58rem]">
                <TableHeader>
                  <TableRow className="bg-muted/40">
                    <TableHead>Code</TableHead>
                    <TableHead>Toko</TableHead>
                    <TableHead>Player</TableHead>
                    <TableHead>Flow</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {overviewQuery.isLoading
                    ? Array.from({ length: 5 }, (_, index) => (
                        <TableRow key={index}>
                          {Array.from({ length: 7 }, (_, cellIndex) => (
                            <TableCell key={cellIndex}>
                              <Skeleton className="h-4 w-full max-w-28" />
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    : overview?.recentTransactions.map((transaction) => (
                        <TableRow key={transaction.id}>
                          <TableCell className="font-medium">
                            {transaction.code ?? "-"}
                          </TableCell>
                          <TableCell>{transaction.tokoName}</TableCell>
                          <TableCell className="font-mono text-xs">
                            {transaction.player ?? "-"}
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-col gap-1">
                              <Badge variant="outline" className="w-fit capitalize">
                                {transaction.category}
                              </Badge>
                              <span className="text-xs text-muted-foreground capitalize">
                                {transaction.type}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell className="text-right font-semibold">
                            {currencyFormatter.format(transaction.amount)}
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant={statusVariant[transaction.status]}
                              className="capitalize"
                            >
                              {transaction.status}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {format(new Date(transaction.createdAt), "dd MMM yyyy HH:mm")}
                          </TableCell>
                        </TableRow>
                      ))}
                  {!overviewQuery.isLoading &&
                  (overview?.recentTransactions.length ?? 0) === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={7}
                        className="h-28 text-center text-sm text-muted-foreground"
                      >
                        Belum ada transaksi yang terlihat pada scope actor saat ini.
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      </motion.section>
    </main>
  )
}
