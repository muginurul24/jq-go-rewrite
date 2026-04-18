import { useDeferredValue, useMemo, useState } from "react"

import { format } from "date-fns"
import {
  CoinsIcon,
  EyeIcon,
  RefreshCcwIcon,
  SearchIcon,
  UsersRoundIcon,
  WalletCardsIcon,
} from "lucide-react"
import { toast } from "sonner"

import { useAuthBootstrap } from "@/features/auth/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import { usePlayerMoneyInfoQuery, usePlayersQuery } from "@/features/players/queries"
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

type PlayerFilters = {
  search: string
  tokoId: string
  page: number
  perPage: number
}

const defaultFilters: PlayerFilters = {
  search: "",
  tokoId: "",
  page: 1,
  perPage: 25,
}

export function PlayersPage() {
  const authBootstrap = useAuthBootstrap()
  const isGlobalRole =
    authBootstrap.data?.user?.role === "dev" ||
    authBootstrap.data?.user?.role === "superadmin"

  const [filters, setFilters] = useState<PlayerFilters>(defaultFilters)
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const deferredSearch = useDeferredValue(filters.search)
  const queryParams = useMemo(
    () => ({
      ...filters,
      search: deferredSearch,
    }),
    [deferredSearch, filters],
  )

  const playersQuery = usePlayersQuery(queryParams)
  const moneyInfoQuery = usePlayerMoneyInfoQuery(selectedPlayerId, detailOpen)

  function updateFilter<Key extends keyof PlayerFilters>(
    key: Key,
    value: PlayerFilters[Key],
  ) {
    setFilters((current) => ({
      ...current,
      [key]: value,
      page: key === "page" || key === "perPage" ? current.page : 1,
    }))
  }

  function openMoneyInfo(playerId: number) {
    setSelectedPlayerId(playerId)
    setDetailOpen(true)
  }

  const records = playersQuery.data?.data ?? []
  const tokos = playersQuery.data?.filters.tokos ?? []

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 md:grid-cols-3">
        <MiniStat
          title="Visible players"
          value={String(playersQuery.data?.meta.total ?? 0)}
          description="Jumlah player yang terlihat sesuai owner scoping legacy."
          icon={UsersRoundIcon}
        />
        <MiniStat
          title="Visible tokos"
          value={String(tokos.length)}
          description="Toko yang currently accessible untuk audit player."
          icon={WalletCardsIcon}
        />
        <MiniStat
          title="Upstream checks"
          value={detailOpen && moneyInfoQuery.data ? "1 live check" : "Idle"}
          description="Action money info tetap lewat backend, bukan langsung dari browser."
          icon={CoinsIcon}
        />
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Players registry</CardTitle>
              <CardDescription>
                Mapping username lokal ke ext_username upstream dengan lookup balance real-time
                melalui adapter Go yang sama dengan public API.
              </CardDescription>
            </div>
            <Button
              variant="outline"
              className="rounded-xl"
              onClick={() => playersQuery.refetch()}
            >
              <RefreshCcwIcon className="size-4" />
              Refresh players
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <div className="relative md:col-span-2">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={filters.search}
              onChange={(event) => updateFilter("search", event.target.value)}
              placeholder="Cari username lokal, external, toko, atau owner"
              className="pl-9"
            />
          </div>

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
              {tokos.map((option, index) => (
                <SelectItem key={`player-filter-toko-${option.id}-${index}`} value={String(option.id)}>
                  {option.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Daftar player</CardTitle>
          <CardDescription>
            Username lokal tetap jadi wajah utama. `ext_username` hanya muncul untuk role global
            atau saat audit mendalam diperlukan.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
            <Table className={isGlobalRole ? "min-w-[56rem]" : "min-w-[48rem]"}>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Toko</TableHead>
                  {isGlobalRole ? <TableHead>External Username</TableHead> : null}
                  <TableHead>Owner</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {playersQuery.isLoading
                  ? Array.from({ length: filters.perPage }).map((_, index) => (
                      <TableRow key={`player-skeleton-${index}`}>
                        <TableCell colSpan={isGlobalRole ? 6 : 5}>
                          <Skeleton className="h-10 w-full rounded-xl" />
                        </TableCell>
                      </TableRow>
                    ))
                  : records.map((record, index) => (
                      <TableRow key={`player-row-${record.id}-${record.extUsername}-${index}`}>
                        <TableCell>
                          <div className="space-y-1">
                            <p className="font-medium">{record.username}</p>
                            <p className="text-xs text-muted-foreground">
                              Player #{record.id}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell>{record.tokoName}</TableCell>
                        {isGlobalRole ? (
                          <TableCell className="font-mono text-xs">
                            {record.extUsername}
                          </TableCell>
                        ) : null}
                        <TableCell>{record.ownerUsername}</TableCell>
                        <TableCell>
                          {format(new Date(record.createdAt), "dd MMM yyyy HH:mm")}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="outline"
                            className="rounded-xl"
                            onClick={() => openMoneyInfo(record.id)}
                          >
                            <EyeIcon className="size-4" />
                            Money info
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
              </TableBody>
            </Table>
          </div>

          {!playersQuery.isLoading && records.length === 0 ? (
            <Card className="rounded-[1.25rem] border-dashed border-border/70 bg-background/40">
              <CardContent className="py-10 text-center">
                <p className="font-medium">Player tidak ditemukan</p>
                <p className="mt-2 text-sm text-muted-foreground">
                  Coba ganti toko atau kata kunci pencarian.
                </p>
              </CardContent>
            </Card>
          ) : null}

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">
              Page {playersQuery.data?.meta.page ?? filters.page} dari{" "}
              {playersQuery.data?.meta.totalPages ?? 1} • {playersQuery.data?.meta.total ?? 0} player
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={(playersQuery.data?.meta.page ?? filters.page) <= 1}
                onClick={() =>
                  updateFilter(
                    "page",
                    Math.max(1, (playersQuery.data?.meta.page ?? filters.page) - 1),
                  )
                }
              >
                Previous
              </Button>
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={
                  (playersQuery.data?.meta.page ?? filters.page) >=
                  (playersQuery.data?.meta.totalPages ?? 1)
                }
                onClick={() =>
                  updateFilter(
                    "page",
                    (playersQuery.data?.meta.page ?? filters.page) + 1,
                  )
                }
              >
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={detailOpen}
        onOpenChange={(nextOpen) => {
          setDetailOpen(nextOpen)
          if (!nextOpen) {
            setSelectedPlayerId(null)
          }
        }}
      >
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Player money info</DialogTitle>
            <DialogDescription>
              Snapshot balance upstream terbaru untuk player lokal yang dipilih.
            </DialogDescription>
          </DialogHeader>

          {moneyInfoQuery.isLoading ? (
            <Skeleton className="h-40 rounded-[1.5rem]" />
          ) : moneyInfoQuery.isError ? (
            <Card className="rounded-[1.25rem] border-dashed border-destructive/40 bg-destructive/5">
              <CardContent className="py-8 text-center">
                <p className="font-medium text-destructive">
                  {isBackofficeRequestError(moneyInfoQuery.error)
                    ? moneyInfoQuery.error.payload.message
                    : "Gagal mengambil balance player."}
                </p>
                <Button
                  variant="outline"
                  className="mt-4 rounded-xl"
                  onClick={() => {
                    void moneyInfoQuery.refetch().catch((error: unknown) => {
                      toast.error(
                        isBackofficeRequestError(error)
                          ? error.payload.message
                          : "Retry money info gagal.",
                      )
                    })
                  }}
                >
                  Retry
                </Button>
              </CardContent>
            </Card>
          ) : moneyInfoQuery.data ? (
            <div className="grid gap-3">
              <PlayerInfoLine label="Username lokal" value={moneyInfoQuery.data.data.username} />
              {isGlobalRole ? (
                <PlayerInfoLine
                  label="External username"
                  value={moneyInfoQuery.data.data.extUsername}
                  mono
                />
              ) : null}
              <PlayerInfoLine label="Toko" value={moneyInfoQuery.data.data.tokoName} />
              <PlayerInfoLine
                label="Balance"
                value={currencyFormatter.format(moneyInfoQuery.data.data.balance)}
              />
              <PlayerInfoLine
                label="Checked at"
                value={format(new Date(moneyInfoQuery.data.data.checkedAt), "dd MMM yyyy HH:mm:ss")}
              />
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </main>
  )
}

function MiniStat({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: typeof UsersRoundIcon
}) {
  return (
    <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
      <CardHeader className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <CardDescription>{title}</CardDescription>
            <CardTitle className="text-2xl">{value}</CardTitle>
          </div>
          <span className="inline-flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
            <Icon className="size-5" />
          </span>
        </div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardHeader>
    </Card>
  )
}

function PlayerInfoLine({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between rounded-[1.1rem] border border-border/70 px-3 py-3">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono text-xs" : "font-medium"}>{value}</span>
    </div>
  )
}
