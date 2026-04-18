import { useDeferredValue, useMemo, useState } from "react"

import { format } from "date-fns"
import {
  DownloadIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  EyeIcon,
  RefreshCcwIcon,
  SearchIcon,
} from "lucide-react"
import { toast } from "sonner"

import { useIsMobile } from "@/hooks/use-mobile"
import { useAuthBootstrap } from "@/features/auth/queries"
import { downloadTransactionsExport } from "@/features/transactions/api"
import {
  useTransactionDetailQuery,
  useTransactionsQuery,
} from "@/features/transactions/queries"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

type TransactionFilters = {
  search: string
  category: string
  type: string
  status: string
  tokoId: string
  dateFrom: string
  dateUntil: string
  amountMin: string
  amountMax: string
  page: number
  perPage: number
}

const defaultFilters: TransactionFilters = {
  search: "",
  category: "",
  type: "",
  status: "",
  tokoId: "",
  dateFrom: "",
  dateUntil: "",
  amountMin: "",
  amountMax: "",
  page: 1,
  perPage: 25,
}

export function TransactionsPage() {
  const isMobile = useIsMobile()
  const authBootstrap = useAuthBootstrap()
  const [filters, setFilters] = useState<TransactionFilters>(defaultFilters)
  const [selectedTransactionID, setSelectedTransactionID] = useState<number | null>(
    null,
  )
  const [isExporting, setIsExporting] = useState(false)

  const deferredSearch = useDeferredValue(filters.search)
  const queryParams = useMemo(
    () => ({
      ...filters,
      search: deferredSearch,
    }),
    [deferredSearch, filters],
  )

  const transactionsQuery = useTransactionsQuery(queryParams)
  const detailQuery = useTransactionDetailQuery(selectedTransactionID)

  const isGlobalRole =
    authBootstrap.data?.user?.role === "dev" ||
    authBootstrap.data?.user?.role === "superadmin"

  function updateFilter<Key extends keyof TransactionFilters>(
    key: Key,
    value: TransactionFilters[Key],
  ) {
    setFilters((current) => ({
      ...current,
      [key]: value,
      page: key === "page" || key === "perPage" ? current.page : 1,
    }))
  }

  function updatePage(nextPage: number) {
    setFilters((current) => ({
      ...current,
      page: nextPage,
    }))
  }

  async function handleExport(format: "csv" | "xlsx") {
    try {
      setIsExporting(true)
      await downloadTransactionsExport(queryParams, format)
      toast.success(`Export ${format.toUpperCase()} dimulai.`)
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Gagal mengekspor transaksi."
      toast.error(message)
    } finally {
      setIsExporting(false)
    }
  }

  const totalAmount = transactionsQuery.data?.summary.totalAmount ?? 0
  const totalRecords = transactionsQuery.data?.meta.total ?? 0
  const page = transactionsQuery.data?.meta.page ?? filters.page
  const totalPages = transactionsQuery.data?.meta.totalPages ?? 1

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 lg:grid-cols-2 xl:grid-cols-[minmax(0,1.6fr)_minmax(260px,0.7fr)_minmax(220px,0.7fr)]">
        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader>
            <CardTitle>Transaction audit center</CardTitle>
            <CardDescription>
              Filter, cari, dan audit transaksi QRIS maupun NexusGGR dengan owner
              scoping yang sama seperti legacy Filament.
            </CardDescription>
          </CardHeader>
        </Card>
        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader>
            <CardDescription>Total nominal terfilter</CardDescription>
            <CardTitle className="text-2xl">
              {currencyFormatter.format(totalAmount)}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader>
            <CardDescription>Total records</CardDescription>
            <CardTitle className="text-2xl">{totalRecords}</CardTitle>
          </CardHeader>
        </Card>
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Filters</CardTitle>
              <CardDescription>
                Search, kategori, tipe, status, toko, tanggal, dan nominal.
              </CardDescription>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="rounded-xl"
                    disabled={isExporting || transactionsQuery.isLoading}
                  >
                    <DownloadIcon className="size-4" />
                    {isExporting ? "Exporting..." : "Export"}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-40 rounded-xl">
                  <DropdownMenuItem onClick={() => void handleExport("csv")}>
                    Export CSV
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => void handleExport("xlsx")}>
                    Export XLSX
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <Button
                variant="outline"
                className="rounded-xl"
                onClick={() => setFilters(defaultFilters)}
              >
                <RefreshCcwIcon className="size-4" />
                Reset filter
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <div className="relative xl:col-span-2">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={filters.search}
              onChange={(event) => updateFilter("search", event.target.value)}
              placeholder="Cari toko, owner, player, external, atau reference"
              className="pl-9"
            />
          </div>

          <Select
            value={filters.category || "all"}
            onValueChange={(value) =>
              updateFilter("category", value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Semua kategori" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua kategori</SelectItem>
              <SelectItem value="qris">QRIS</SelectItem>
              <SelectItem value="nexusggr">NexusGGR</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={filters.type || "all"}
            onValueChange={(value) =>
              updateFilter("type", value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Semua tipe" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua tipe</SelectItem>
              <SelectItem value="deposit">Deposit</SelectItem>
              <SelectItem value="withdrawal">Withdrawal</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={filters.status || "all"}
            onValueChange={(value) =>
              updateFilter("status", value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Semua status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua status</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="success">Success</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={filters.tokoId || "all"}
            onValueChange={(value) =>
              updateFilter("tokoId", value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Semua toko" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua toko</SelectItem>
              {transactionsQuery.data?.filters.tokos.map((option) => (
                <SelectItem key={option.id} value={String(option.id)}>
                  {isGlobalRole
                    ? `${option.name} (${option.ownerUsername})`
                    : option.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Input
            type="date"
            value={filters.dateFrom}
            onChange={(event) => updateFilter("dateFrom", event.target.value)}
          />

          <Input
            type="date"
            value={filters.dateUntil}
            onChange={(event) => updateFilter("dateUntil", event.target.value)}
          />

          <Input
            type="number"
            inputMode="numeric"
            placeholder="Min nominal"
            value={filters.amountMin}
            onChange={(event) => updateFilter("amountMin", event.target.value)}
          />

          <Input
            type="number"
            inputMode="numeric"
            placeholder="Max nominal"
            value={filters.amountMax}
            onChange={(event) => updateFilter("amountMax", event.target.value)}
          />

          <Select
            value={String(filters.perPage)}
            onValueChange={(value) =>
              setFilters((current) => ({
                ...current,
                perPage: Number(value),
                page: 1,
              }))
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Rows" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="25">25 rows</SelectItem>
              <SelectItem value="50">50 rows</SelectItem>
              <SelectItem value="100">100 rows</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Transactions</CardTitle>
          <CardDescription>
            Default sort mengikuti legacy: terbaru di atas.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {transactionsQuery.isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 8 }).map((_, index) => (
                <Skeleton key={index} className="h-12 w-full rounded-xl" />
              ))}
            </div>
          ) : transactionsQuery.isError ? (
            <div className="rounded-[1.25rem] border border-destructive/25 bg-destructive/8 px-4 py-6 text-sm text-destructive">
              Gagal memuat transaksi. Coba refresh atau periksa session login.
            </div>
          ) : transactionsQuery.data && transactionsQuery.data.data.length > 0 ? (
            <>
              <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
              <Table className={isGlobalRole ? "min-w-[86rem]" : "min-w-[72rem]"}>
                <TableHeader>
                  <TableRow>
                    <TableHead>#</TableHead>
                    <TableHead>Toko</TableHead>
                    {isGlobalRole ? <TableHead>Owner</TableHead> : null}
                    <TableHead>Player</TableHead>
                    {isGlobalRole ? <TableHead>External</TableHead> : null}
                    <TableHead>Kategori</TableHead>
                    <TableHead>Tipe</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Nominal</TableHead>
                    <TableHead>Reference</TableHead>
                    <TableHead>Dibuat</TableHead>
                    <TableHead className="text-right">Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactionsQuery.data.data.map((transaction, index) => (
                    <TableRow key={transaction.id}>
                      <TableCell>
                        {(page - 1) * filters.perPage + index + 1}
                      </TableCell>
                      <TableCell>
                        <div className="grid gap-1">
                          <span className="font-medium">{transaction.tokoName}</span>
                          {transaction.noteSummary ? (
                            <span className="max-w-[240px] truncate text-xs text-muted-foreground">
                              {transaction.noteSummary}
                            </span>
                          ) : null}
                        </div>
                      </TableCell>
                      {isGlobalRole ? (
                        <TableCell>@{transaction.ownerUsername}</TableCell>
                      ) : null}
                      <TableCell>{transaction.player ?? "-"}</TableCell>
                      {isGlobalRole ? (
                        <TableCell>{transaction.externalPlayer ?? "-"}</TableCell>
                      ) : null}
                      <TableCell>
                        <TransactionBadge value={transaction.categoryLabel} />
                      </TableCell>
                      <TableCell>
                        <TransactionBadge value={transaction.typeLabel} />
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={transaction.status} label={transaction.statusLabel} />
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {currencyFormatter.format(transaction.amount)}
                      </TableCell>
                      <TableCell>{transaction.code ?? "-"}</TableCell>
                      <TableCell>
                        {format(new Date(transaction.createdAt), "dd MMM yyyy HH:mm")}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          className="rounded-xl"
                          onClick={() => setSelectedTransactionID(transaction.id)}
                        >
                          <EyeIcon className="size-4" />
                          Detail
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              </div>

              <div className="flex flex-col gap-3 border-t pt-4 md:flex-row md:items-center md:justify-between">
                <p className="text-sm text-muted-foreground">
                  Page {page} of {Math.max(totalPages, 1)}
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="rounded-xl"
                    disabled={page <= 1 || transactionsQuery.isFetching}
                    onClick={() => updatePage(page - 1)}
                  >
                    <ChevronLeftIcon className="size-4" />
                    Prev
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="rounded-xl"
                    disabled={page >= totalPages || transactionsQuery.isFetching}
                    onClick={() => updatePage(page + 1)}
                  >
                    Next
                    <ChevronRightIcon className="size-4" />
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-10 text-center">
              <p className="font-medium">Belum ada transaksi</p>
              <p className="mt-2 text-sm text-muted-foreground">
                Transaksi dari QRIS dan NexusGGR akan muncul di sini untuk audit
                dan monitoring.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <TransactionDetailOverlay
        isMobile={isMobile}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedTransactionID(null)
          }
        }}
        open={selectedTransactionID !== null}
        detail={detailQuery.data?.data ?? null}
        isLoading={detailQuery.isLoading || detailQuery.isFetching}
      />
    </main>
  )
}

function TransactionBadge({ value }: { value: string }) {
  return (
    <Badge variant="outline" className="rounded-full px-2.5">
      {value}
    </Badge>
  )
}

function StatusBadge({
  status,
  label,
}: {
  status: string
  label: string
}) {
  const className =
    status === "success"
      ? "border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
      : status === "pending"
        ? "border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300"
        : status === "failed"
          ? "border-rose-500/20 bg-rose-500/10 text-rose-700 dark:text-rose-300"
          : "border-border/70 bg-muted/60"

  return (
    <Badge variant="outline" className={`rounded-full px-2.5 ${className}`}>
      {label}
    </Badge>
  )
}

type TransactionDetailOverlayProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  detail: {
    tokoName: string
    ownerUsername: string
    player?: string | null
    externalPlayer?: string | null
    categoryLabel: string
    typeLabel: string
    statusLabel: string
    amount: number
    code?: string | null
    createdAt: string
    updatedAt: string
    notePayload: string
  } | null
  isLoading: boolean
  isMobile: boolean
}

function TransactionDetailOverlay({
  open,
  onOpenChange,
  detail,
  isLoading,
  isMobile,
}: TransactionDetailOverlayProps) {
  const content = (
    <>
      {isLoading ? (
        <div className="space-y-3 p-4">
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : detail ? (
        <div className="space-y-4 px-4 pb-4">
          <div className="grid gap-3 md:grid-cols-2">
            <DetailCard label="Toko" value={detail.tokoName} />
            <DetailCard label="Owner" value={`@${detail.ownerUsername}`} />
            <DetailCard label="Player" value={detail.player ?? "-"} />
            <DetailCard label="External Player" value={detail.externalPlayer ?? "-"} />
            <DetailCard label="Kategori" value={detail.categoryLabel} />
            <DetailCard label="Tipe" value={detail.typeLabel} />
            <DetailCard label="Status" value={detail.statusLabel} />
            <DetailCard
              label="Nominal"
              value={currencyFormatter.format(detail.amount)}
            />
            <DetailCard label="Reference" value={detail.code ?? "-"} />
            <DetailCard
              label="Created"
              value={format(new Date(detail.createdAt), "dd MMM yyyy HH:mm:ss")}
            />
            <DetailCard
              label="Updated"
              value={format(new Date(detail.updatedAt), "dd MMM yyyy HH:mm:ss")}
            />
          </div>

          <Card className="rounded-[1.25rem] border-border/70 bg-background/70">
            <CardHeader>
              <CardTitle className="text-base">Payload audit</CardTitle>
              <CardDescription>
                Pretty-printed note payload untuk kebutuhan investigasi.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="overflow-x-auto rounded-xl border border-border/70 bg-background p-4 font-mono text-xs leading-6 whitespace-pre-wrap">
                {detail.notePayload}
              </pre>
            </CardContent>
          </Card>
        </div>
      ) : null}
    </>
  )

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={onOpenChange}>
        <DrawerContent className="max-h-[92svh] overflow-hidden">
          <DrawerHeader>
            <DrawerTitle>Detail Transaksi</DrawerTitle>
            <DrawerDescription>
              Audit payload, pemain, dan metadata transaksi.
            </DrawerDescription>
          </DrawerHeader>
          <div className="min-h-0 overflow-y-auto">{content}</div>
        </DrawerContent>
      </Drawer>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="grid max-h-[92svh] grid-rows-[auto_minmax(0,1fr)] overflow-hidden p-0 sm:max-w-5xl">
        <DialogHeader className="p-4 pb-0">
          <DialogTitle>Detail Transaksi</DialogTitle>
          <DialogDescription>
            Audit payload, pemain, dan metadata transaksi.
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 overflow-y-auto">{content}</div>
      </DialogContent>
    </Dialog>
  )
}

function DetailCard({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div className="rounded-[1.25rem] border border-border/70 bg-background/60 p-4">
      <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
        {label}
      </p>
      <p className="mt-2 break-words font-medium">{value}</p>
    </div>
  )
}
