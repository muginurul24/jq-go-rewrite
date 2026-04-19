import { useDeferredValue, useMemo, useState } from "react"

import { format } from "date-fns"
import {
  CrosshairIcon,
  PhoneCallIcon,
  RefreshCcwIcon,
  SearchIcon,
  ShieldCheckIcon,
  UserRoundCogIcon,
  UsersRoundIcon,
  XCircleIcon,
} from "lucide-react"
import { toast } from "sonner"

import type { ActivePlayerRecord } from "@/features/call-management/api"
import {
  useActivePlayersQuery,
  useApplyCallMutation,
  useCallHistoryQuery,
  useCallListQuery,
  useCallManagementBootstrapQuery,
  useCancelCallMutation,
  useControlRtpMutation,
  useControlUsersRtpMutation,
} from "@/features/call-management/queries"
import { useProvidersQuery } from "@/features/catalog/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import { cn } from "@/lib/utils"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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

export function CallManagementPage() {
  const [playerSearch, setPlayerSearch] = useState("")
  const [historySearch, setHistorySearch] = useState("")
  const [selectedPlayerKey, setSelectedPlayerKey] = useState<string | null>(null)
  const [applyDialogOpen, setApplyDialogOpen] = useState(false)
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)
  const [pendingCancelCallId, setPendingCancelCallId] = useState<number | null>(null)
  const [applyCallTypeValue, setApplyCallTypeValue] = useState("1")
  const [applyCallRtpValue, setApplyCallRtpValue] = useState<number | null>(null)
  const [controlDialogOpen, setControlDialogOpen] = useState(false)
  const [controlUsersDialogOpen, setControlUsersDialogOpen] = useState(false)
  const [controlPlayerId, setControlPlayerId] = useState<string>("")
  const [controlProviderCode, setControlProviderCode] = useState("")
  const [controlRtpValue, setControlRtpValue] = useState("95")
  const [controlUsersRtpValue, setControlUsersRtpValue] = useState("95")

  const deferredPlayerSearch = useDeferredValue(playerSearch)
  const deferredHistorySearch = useDeferredValue(historySearch)

  const bootstrapQuery = useCallManagementBootstrapQuery()
  const activePlayersQuery = useActivePlayersQuery()
  const historyQuery = useCallHistoryQuery()
  const providersQuery = useProvidersQuery()
  const providerOptions = useMemo(() => {
    const providers = providersQuery.data?.providers ?? []
    const activeProviders = providers.filter((provider) => Number(provider.status) === 1)
    return activeProviders.length ? activeProviders : providers
  }, [providersQuery.data?.providers])
  const selectedPlayer = useMemo(() => {
    if (selectedPlayerKey == null) {
      return null
    }

    return activePlayersQuery.data?.data.find((item) => playerKey(item) === selectedPlayerKey) ?? null
  }, [activePlayersQuery.data?.data, selectedPlayerKey])
  const callListQuery = useCallListQuery(
    selectedPlayer?.providerCode ?? "",
    selectedPlayer?.gameCode ?? "",
  )
  const applyMutation = useApplyCallMutation()
  const cancelMutation = useCancelCallMutation()
  const controlRtpMutation = useControlRtpMutation()
  const controlUsersRtpMutation = useControlUsersRtpMutation()

  const activePlayers = useMemo(() => {
    const items = activePlayersQuery.data?.data ?? []
    const term = deferredPlayerSearch.trim().toLowerCase()
    if (!term) {
      return items
    }

    return items.filter((item) => {
      return [
        item.userLabel,
        item.tokoName,
        item.providerCode,
        item.gameCode,
      ].some((value) => value.toLowerCase().includes(term))
    })
  }, [activePlayersQuery.data?.data, deferredPlayerSearch])

  const historyRecords = useMemo(() => {
    const items = historyQuery.data?.data ?? []
    const term = deferredHistorySearch.trim().toLowerCase()
    if (!term) {
      return items
    }

    return items.filter((item) => {
      return [
        item.userLabel,
        item.tokoName,
        String(item.providerCode ?? ""),
        String(item.gameCode ?? ""),
        item.statusLabel,
      ].some((value) => value.toLowerCase().includes(term))
    })
  }, [deferredHistorySearch, historyQuery.data?.data])

  function openControlRtpDialog() {
    const defaultPlayerId =
      selectedPlayer?.playerId != null
        ? String(selectedPlayer.playerId)
        : bootstrapQuery.data?.data.managedPlayers[0]?.id != null
          ? String(bootstrapQuery.data.data.managedPlayers[0].id)
          : ""
    const defaultProvider =
      selectedPlayer?.providerCode ||
      providerOptions[0]?.code ||
      ""

    setControlPlayerId(defaultPlayerId)
    setControlProviderCode(defaultProvider)
    setControlRtpValue("95")
    setControlDialogOpen(true)
  }

  function openApplyCallDialog(callTypeValue: number, callRtp: number) {
    if (!selectedPlayer) {
      toast.error("Pilih active player terlebih dahulu.")
      return
    }

    setApplyCallTypeValue(String(callTypeValue))
    setApplyCallRtpValue(callRtp)
    setApplyDialogOpen(true)
  }

  async function handleApply() {
    if (!selectedPlayer) {
      toast.error("Pilih active player terlebih dahulu.")
      return
    }
    if (applyCallRtpValue == null) {
      toast.error("Call RTP belum dipilih.")
      return
    }

    try {
      const response = await applyMutation.mutateAsync({
        playerId: selectedPlayer.playerId,
        providerCode: selectedPlayer.providerCode,
        gameCode: selectedPlayer.gameCode,
        callRtp: applyCallRtpValue,
        callTypeValue: Number(applyCallTypeValue),
      })
      toast.success(`Call berhasil di-apply. Called money: ${String(response.data.calledMoney ?? 0)}`)
      setApplyDialogOpen(false)
      void activePlayersQuery.refetch()
      void historyQuery.refetch()
      void callListQuery.refetch()
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal apply call.",
      )
    }
  }

  function openCancelDialog(callId: number) {
    setPendingCancelCallId(callId)
    setCancelDialogOpen(true)
  }

  async function handleCancel() {
    if (pendingCancelCallId == null) {
      toast.error("Call yang akan dibatalkan tidak valid.")
      return
    }

    try {
      const response = await cancelMutation.mutateAsync(pendingCancelCallId)
      toast.success(`Call dibatalkan. Canceled money: ${String(response.data.canceledMoney ?? 0)}`)
      setCancelDialogOpen(false)
      setPendingCancelCallId(null)
      void historyQuery.refetch()
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal membatalkan call.",
      )
    }
  }

  async function handleControlRtp() {
    const parsedRtp = Number(controlRtpValue)
    if (parsedRtp < 0 || parsedRtp > 95) {
      toast.error("RTP harus di antara 0 sampai 95.")
      return
    }

    try {
      const response = await controlRtpMutation.mutateAsync({
        playerId: Number(controlPlayerId),
        providerCode: controlProviderCode,
        rtp: parsedRtp,
      })
      toast.success(`RTP berhasil diubah ke ${String(response.data.changedRtp ?? controlRtpValue)}.`)
      setControlDialogOpen(false)
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal mengubah RTP player.",
      )
    }
  }

  async function handleControlUsersRtp() {
    const parsedRtp = Number(controlUsersRtpValue)
    if (parsedRtp < 0 || parsedRtp > 95) {
      toast.error("RTP harus di antara 0 sampai 95.")
      return
    }

    try {
      const response = await controlUsersRtpMutation.mutateAsync(parsedRtp)
      toast.success(`RTP mass update berhasil: ${String(response.data.changedRtp ?? controlUsersRtpValue)}.`)
      setControlUsersDialogOpen(false)
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal mengubah RTP semua player.",
      )
    }
  }

  const stats = [
    {
      title: "Active players",
      value: String(activePlayersQuery.data?.data.length ?? 0),
      description: "Player aktif dari upstream yang sukses dipetakan ke player lokal yang visible untuk actor.",
      icon: PhoneCallIcon,
    },
    {
      title: "Managed players",
      value: String(bootstrapQuery.data?.data.managedPlayers.length ?? 0),
      description: "Semua player lokal yang bisa dipakai untuk RTP control satuan maupun mass update.",
      icon: UsersRoundIcon,
    },
    {
      title: "History rows",
      value: String(historyQuery.data?.data.length ?? 0),
      description: "Riwayat call yang sudah disanitasi dan dipoll setiap 10 detik seperti legacy.",
      icon: CrosshairIcon,
    },
  ]

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 xl:grid-cols-3">
        {stats.map(({ title, value, description, icon: Icon }) => (
          <Card key={title} className="rounded-[1.75rem] border-border/70 bg-card/90">
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
        ))}
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>Call management</CardTitle>
            <CardDescription>
              Active player, call option, cancel history, dan RTP control tanpa membuka payload upstream mentah.
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              className="w-full rounded-xl sm:w-auto"
              onClick={() => {
                void activePlayersQuery.refetch()
                void historyQuery.refetch()
              }}
            >
              <RefreshCcwIcon className="size-4" />
              Refresh
            </Button>
            <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={openControlRtpDialog}>
              <UserRoundCogIcon className="size-4" />
              Control RTP
            </Button>
            <Button className="w-full rounded-xl sm:w-auto" onClick={() => setControlUsersDialogOpen(true)}>
              <ShieldCheckIcon className="size-4" />
              Control Users RTP
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-6">
          <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="text-base">Active players</CardTitle>
              <CardDescription>
                Pilih row player aktif untuk me-load call list berdasarkan provider dan game yang sedang aktif.
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4">
              <div className="relative">
                <SearchIcon className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={playerSearch}
                  onChange={(event) => setPlayerSearch(event.target.value)}
                  placeholder="Cari player, toko, provider, atau game..."
                  className="pl-9"
                />
              </div>

              {activePlayersQuery.isLoading ? (
                <Skeleton className="h-[24rem] rounded-[1.25rem]" />
              ) : (
                <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
                  <Table className="min-w-[74rem]">
                    <TableHeader>
                      <TableRow>
                        <TableHead>User</TableHead>
                        <TableHead>Provider</TableHead>
                        <TableHead>Game</TableHead>
                        <TableHead>Bet</TableHead>
                        <TableHead>Balance</TableHead>
                        <TableHead>Total Debit</TableHead>
                        <TableHead>Total Credit</TableHead>
                        <TableHead>Target RTP</TableHead>
                        <TableHead>Real RTP</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {activePlayers.length ? activePlayers.map((item) => {
                        const selected = selectedPlayerKey === playerKey(item)

                        return (
                          <TableRow key={playerKey(item)} data-state={selected ? "selected" : undefined}>
                            <TableCell>
                              <div className="flex flex-col">
                                <span className="font-medium">{item.userLabel}</span>
                                <span className="text-xs text-muted-foreground">{item.tokoName}</span>
                              </div>
                            </TableCell>
                            <TableCell>{item.providerCode}</TableCell>
                            <TableCell>{item.gameCode}</TableCell>
                            <TableCell>{String(item.bet ?? "-")}</TableCell>
                            <TableCell>{String(item.balance ?? "-")}</TableCell>
                            <TableCell>{String(item.totalDebit ?? "-")}</TableCell>
                            <TableCell>{String(item.totalCredit ?? "-")}</TableCell>
                            <TableCell>{String(item.targetRtp ?? "-")}</TableCell>
                            <TableCell>{String(item.realRtp ?? "-")}</TableCell>
                            <TableCell className="text-right">
                              <Button
                                variant={selected ? "default" : "outline"}
                                className="rounded-xl"
                                onClick={() => setSelectedPlayerKey(playerKey(item))}
                              >
                                View Calls
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      }) : (
                        <TableRow>
                          <TableCell colSpan={10} className="py-12 text-center text-sm text-muted-foreground">
                            Tidak ada active player yang bisa dipetakan ke scope actor ini.
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>

          {selectedPlayer ? (
            <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
              <CardHeader>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <CardTitle className="text-base">Call List</CardTitle>
                    <CardDescription>
                      {selectedPlayer.userLabel} - {selectedPlayer.providerCode} / {selectedPlayer.gameCode}
                    </CardDescription>
                  </div>
                  <Button
                    variant="outline"
                    className="w-full rounded-xl sm:w-auto"
                    onClick={() => setSelectedPlayerKey(null)}
                  >
                    Close
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {callListQuery.isLoading ? (
                  <Skeleton className="h-[18rem] rounded-[1.25rem]" />
                ) : callListQuery.data?.data.length ? (
                  <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
                    <Table className="min-w-[28rem]">
                      <TableHeader>
                        <TableRow>
                          <TableHead>RTP</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead className="text-right">Action</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {callListQuery.data.data.map((item, index) => (
                          <TableRow key={`${selectedPlayer.playerId}-${String(item.rtp)}-${item.callTypeValue}-${index}`}>
                            <TableCell>{String(item.rtp ?? "-")}</TableCell>
                            <TableCell>{item.callType}</TableCell>
                            <TableCell className="text-right">
                              <Button
                                className="rounded-xl"
                                disabled={applyMutation.isPending}
                                onClick={() => openApplyCallDialog(item.callTypeValue, Number(item.rtp ?? 0))}
                              >
                                Apply
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                ) : (
                  <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-12 text-center text-sm text-muted-foreground">
                    Tidak ada call option yang tersedia untuk player aktif ini.
                  </div>
                )}
              </CardContent>
            </Card>
          ) : null}

          <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="text-base">Call history</CardTitle>
              <CardDescription>
                Riwayat call disanitasi dan otomatis refresh setiap 10 detik. Hanya record player yang visible untuk actor yang ditampilkan.
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4">
              <div className="relative">
                <SearchIcon className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={historySearch}
                  onChange={(event) => setHistorySearch(event.target.value)}
                  placeholder="Cari user, toko, provider, game, atau status..."
                  className="pl-9"
                />
              </div>

              {historyQuery.isLoading ? (
                <Skeleton className="h-[22rem] rounded-[1.25rem]" />
              ) : (
                <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
                  <Table className="min-w-[78rem]">
                    <TableHeader>
                      <TableRow>
                        <TableHead>ID</TableHead>
                        <TableHead>User</TableHead>
                        <TableHead>Provider</TableHead>
                        <TableHead>Game</TableHead>
                        <TableHead>Bet</TableHead>
                        <TableHead>Expected</TableHead>
                        <TableHead>Real</TableHead>
                        <TableHead>Missed</TableHead>
                        <TableHead>RTP</TableHead>
                        <TableHead>Type</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Created</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {historyRecords.length ? historyRecords.map((item) => (
                        <TableRow key={String(item.id)}>
                          <TableCell>{String(item.id)}</TableCell>
                          <TableCell>
                            <div className="flex flex-col">
                              <span className="font-medium">{item.userLabel}</span>
                              <span className="text-xs text-muted-foreground">{item.tokoName}</span>
                            </div>
                          </TableCell>
                          <TableCell>{String(item.providerCode ?? "-")}</TableCell>
                          <TableCell>{String(item.gameCode ?? "-")}</TableCell>
                          <TableCell>{String(item.bet ?? "-")}</TableCell>
                          <TableCell>{String(item.expect ?? "-")}</TableCell>
                          <TableCell>{String(item.real ?? "-")}</TableCell>
                          <TableCell>{String(item.missed ?? "-")}</TableCell>
                          <TableCell>{String(item.rtp ?? "-")}</TableCell>
                          <TableCell>{formatCallType(item.type)}</TableCell>
                          <TableCell>
                            <Badge
                              variant="outline"
                              className={cn(
                                "rounded-full border px-2.5 font-medium",
                                statusBadgeClass(item.status),
                              )}
                            >
                              {item.statusLabel}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {formatDateTime(item.createdAt)}
                          </TableCell>
                          <TableCell className="text-right">
                            {item.canCancel ? (
                              <Button
                                variant="outline"
                                className="rounded-xl"
                                disabled={cancelMutation.isPending}
                                onClick={() => openCancelDialog(Number(item.id))}
                              >
                                <XCircleIcon className="size-4" />
                                Cancel
                              </Button>
                            ) : (
                              <span className="text-xs text-muted-foreground">Locked</span>
                            )}
                          </TableCell>
                        </TableRow>
                      )) : (
                        <TableRow>
                          <TableCell colSpan={13} className="py-12 text-center text-sm text-muted-foreground">
                            Belum ada history call yang cocok dengan filter saat ini.
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </CardContent>
      </Card>

      <Dialog open={applyDialogOpen} onOpenChange={setApplyDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Apply Call</DialogTitle>
            <DialogDescription>
              Legacy flow tetap mengizinkan operator memilih ulang call type sebelum submit.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="space-y-2">
              <Label>User Code</Label>
              <Input value={selectedPlayer?.userLabel ?? "-"} disabled />
            </div>

            <div className="space-y-2">
              <Label>Call Type</Label>
              <Select value={applyCallTypeValue} onValueChange={setApplyCallTypeValue}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih call type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Common Free</SelectItem>
                  <SelectItem value="2">Buy Bonus Free</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Call RTP</Label>
              <Input value={String(applyCallRtpValue ?? "-")} disabled />
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={() => setApplyDialogOpen(false)}>
                Batal
              </Button>
              <Button
                className="w-full rounded-xl sm:w-auto"
                disabled={applyMutation.isPending}
                onClick={() => void handleApply()}
              >
                Apply Call
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog
        open={cancelDialogOpen}
        onOpenChange={(open) => {
          setCancelDialogOpen(open)
          if (!open) {
            setPendingCancelCallId(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Cancel Call</DialogTitle>
            <DialogDescription>
              Call dengan status waiting akan dibatalkan seperti flow konfirmasi di legacy.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={() => setCancelDialogOpen(false)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              className="w-full rounded-xl sm:w-auto"
              disabled={cancelMutation.isPending}
              onClick={() => void handleCancel()}
            >
              Confirm Cancel
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={controlDialogOpen} onOpenChange={setControlDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Control RTP</DialogTitle>
            <DialogDescription>
              Pilih provider dan player lokal. Backend akan map ke `ext_username` sebelum request ke upstream.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="space-y-2">
              <Label>Provider</Label>
              <Select value={controlProviderCode} onValueChange={setControlProviderCode}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih provider" />
                </SelectTrigger>
                <SelectContent>
                  {providerOptions.map((provider) => (
                    <SelectItem key={provider.code} value={provider.code}>
                      {provider.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Player</Label>
              <Select value={controlPlayerId} onValueChange={setControlPlayerId}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih player" />
                </SelectTrigger>
                <SelectContent>
                  {bootstrapQuery.data?.data.managedPlayers.map((player) => (
                    <SelectItem key={player.id} value={String(player.id)}>
                      {player.userLabel}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>RTP</Label>
              <Input
                inputMode="decimal"
                value={controlRtpValue}
                onChange={(event) => setControlRtpValue(event.target.value)}
                placeholder="95"
                type="number"
                min={0}
                max={95}
              />
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={() => setControlDialogOpen(false)}>
                Batal
              </Button>
              <Button
                className="w-full rounded-xl sm:w-auto"
                disabled={controlRtpMutation.isPending}
                onClick={() => void handleControlRtp()}
              >
                Simpan RTP
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={controlUsersDialogOpen} onOpenChange={setControlUsersDialogOpen}>
        <DialogContent className="max-h-[92svh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Control Users RTP</DialogTitle>
            <DialogDescription>
              Akan menerapkan RTP yang sama ke seluruh player lokal yang visible untuk actor ini.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="space-y-2">
              <Label>RTP mass update</Label>
              <Input
                inputMode="decimal"
                value={controlUsersRtpValue}
                onChange={(event) => setControlUsersRtpValue(event.target.value)}
                placeholder="95"
                type="number"
                min={0}
                max={95}
              />
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={() => setControlUsersDialogOpen(false)}>
                Batal
              </Button>
              <Button
                className="w-full rounded-xl sm:w-auto"
                disabled={controlUsersRtpMutation.isPending}
                onClick={() => void handleControlUsersRtp()}
              >
                Update semua player
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </main>
  )
}

function formatDateTime(value: string | null) {
  if (!value) {
    return "-"
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return format(parsed, "dd MMM yyyy HH:mm")
}

function formatCallType(value: string | number | null) {
  if (value == null) {
    return "-"
  }

  const normalized = String(value).trim()
  if (!normalized) {
    return "-"
  }

  return normalized
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .split(" ")
    .map((token) => token.charAt(0).toUpperCase() + token.slice(1).toLowerCase())
    .join(" ")
}

function statusBadgeClass(status: string | number | null) {
  switch (Number(status)) {
    case 0:
      return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
    case 1:
      return "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300"
    case 2:
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
    case 3:
      return "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300"
    case 4:
      return "border-muted-foreground/20 bg-muted text-muted-foreground"
    default:
      return "border-muted-foreground/20 bg-background text-foreground"
  }
}

function playerKey(player: ActivePlayerRecord) {
  return `${player.playerId}:${player.providerCode}:${player.gameCode}`
}
