import { useEffect, useMemo, useState } from "react"

import { QrCodeIcon, RefreshCcwIcon, RotateCcwIcon, WalletCardsIcon } from "lucide-react"
import { QRCodeSVG } from "qrcode.react"
import { toast } from "sonner"

import { isBackofficeRequestError } from "@/lib/backoffice-api"
import {
  useCheckTopupStatusMutation,
  useGenerateTopupMutation,
  useTopupBootstrapQuery,
} from "@/features/nexusggr-topup/queries"
import type { PendingTopup } from "@/features/nexusggr-topup/api"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

export function NexusggrTopupPage() {
  const [manualSelectedTokoId, setManualSelectedTokoId] = useState<number | null>(null)
  const [amount, setAmount] = useState("100000")
  const [dismissedCode, setDismissedCode] = useState<string | null>(null)
  const [currentEpochSecond, setCurrentEpochSecond] = useState(() => Math.floor(Date.now() / 1000))

  const bootstrapQuery = useTopupBootstrapQuery(manualSelectedTokoId)
  const generateMutation = useGenerateTopupMutation()
  const checkStatusMutation = useCheckTopupStatusMutation()

  const tokoOptions = useMemo(() => {
    const source = bootstrapQuery.data?.data.tokos ?? []
    const seen = new Set<number>()
    return source.filter((toko) => {
      if (seen.has(toko.id)) {
        return false
      }
      seen.add(toko.id)
      return true
    })
  }, [bootstrapQuery.data?.data.tokos])

  const selectedTokoId = manualSelectedTokoId
    ?? bootstrapQuery.data?.data.selectedToko?.id
    ?? tokoOptions[0]?.id
    ?? null

  const pendingTopup = useMemo(() => {
    const current = bootstrapQuery.data?.data.pendingTopup ?? null
    if (!current) {
      return null
    }
    if (dismissedCode && current.transactionCode === dismissedCode) {
      return null
    }
    return current
  }, [bootstrapQuery.data?.data.pendingTopup, dismissedCode])

  useEffect(() => {
    if (!pendingTopup?.expiresAt || pendingTopup.status !== "pending") {
      return
    }

    const interval = window.setInterval(() => {
      setCurrentEpochSecond(Math.floor(Date.now() / 1000))
    }, 1000)
    return () => window.clearInterval(interval)
  }, [pendingTopup?.expiresAt, pendingTopup?.status])

  const countdown = useMemo(() => {
    if (!pendingTopup?.expiresAt || pendingTopup.status !== "pending") {
      return ""
    }

    const remaining = Math.max(0, pendingTopup.expiresAt - currentEpochSecond)
    const minutes = Math.floor(remaining / 60)
    const seconds = remaining % 60
    return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`
  }, [currentEpochSecond, pendingTopup?.expiresAt, pendingTopup?.status])

  const topupRule = bootstrapQuery.data?.data.topupRule ?? {
    thresholdAmount: 1_000_000,
    belowThresholdRate: 7,
    aboveThresholdRate: 6,
  }
  const selectedToko = tokoOptions.find((item) => item.id === selectedTokoId)
    ?? bootstrapQuery.data?.data.selectedToko
    ?? null
  const parsedAmount = Number(amount.replace(/[^\d]/g, "")) || 0
  const effectiveTopupRatio = parsedAmount > topupRule.thresholdAmount
    ? topupRule.aboveThresholdRate
    : topupRule.belowThresholdRate
  const estimatedCredit =
    parsedAmount > 0 && effectiveTopupRatio > 0
      ? Math.round((parsedAmount * 100) / effectiveTopupRatio)
      : 0
  const estimatedBalance = (selectedToko?.nexusggrBalance ?? 0) + estimatedCredit

  async function handleGenerate() {
    if (selectedTokoId == null) {
      toast.error("Pilih toko terlebih dahulu.")
      return
    }

    try {
      await generateMutation.mutateAsync({
        tokoId: selectedTokoId,
        amount: parsedAmount,
      })
      setDismissedCode(null)
      toast.success("QRIS topup berhasil dibuat.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal generate QRIS topup.",
      )
    }
  }

  async function handleCheckStatus(code: string) {
    if (selectedTokoId == null) {
      return
    }

    try {
      const response = await checkStatusMutation.mutateAsync({
        tokoId: selectedTokoId,
        transactionCode: code,
      })
      const status = response.data.status ?? response.data.pendingTopup?.status

      if (status === "success") {
        toast.success("Pembayaran berhasil. Saldo NexusGGR sudah ter-update jika webhook sudah masuk.")
      } else if (status === "expired") {
        toast.warning("QRIS sudah expired. Silakan generate ulang.")
      } else {
        toast.message("Pembayaran masih pending.")
      }
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal cek status topup.",
      )
    }
  }

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr_0.8fr]">
        <TopupStat
          title="Topup ratio"
          value={effectiveTopupRatio > 0 ? `${effectiveTopupRatio}%` : "-"}
          description={`Rate aktif untuk nominal saat ini. <= ${currencyFormatter.format(topupRule.thresholdAmount)} memakai ${topupRule.belowThresholdRate}%, di atasnya ${topupRule.aboveThresholdRate}%.`}
          icon={QrCodeIcon}
        />
        <TopupStat
          title="Estimated credit"
          value={currencyFormatter.format(estimatedCredit)}
          description="Estimasi saldo NexusGGR yang akan ditambahkan jika callback sukses."
          icon={WalletCardsIcon}
        />
        <TopupStat
          title="Estimated balance"
          value={currencyFormatter.format(estimatedBalance)}
          description="Proyeksi NexusGGR balance toko setelah topup berhasil diproses."
          icon={RefreshCcwIcon}
        />
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>NexusGGR topup</CardTitle>
            <CardDescription>
              Generate QRIS topup, restore pending transaction, dan cek status lokal seperti flow Filament legacy.
            </CardDescription>
          </div>
          <Button
            variant="outline"
            className="w-full rounded-xl sm:w-auto"
            onClick={() => bootstrapQuery.refetch()}
          >
            <RefreshCcwIcon className="size-4" />
            Refresh
          </Button>
        </CardHeader>
        <CardContent className="grid gap-4 xl:grid-cols-[1fr_1fr]">
          <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="text-base">Generate QRIS</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4">
              {bootstrapQuery.isLoading ? (
                <Skeleton className="h-48 rounded-[1.25rem]" />
              ) : (
                <>
                  <div className="space-y-2">
                    <Label>Toko</Label>
                    <Select
                      value={selectedTokoId != null ? String(selectedTokoId) : ""}
                      onValueChange={(value) => {
                        setManualSelectedTokoId(Number(value))
                        setDismissedCode(null)
                      }}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Pilih toko" />
                      </SelectTrigger>
                      <SelectContent>
                        {tokoOptions.map((toko, index) => (
                          <SelectItem key={`topup-toko-${toko.id}-${index}`} value={String(toko.id)}>
                            {toko.name} ({toko.ownerUsername})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>Nominal topup</Label>
                    <Input
                      inputMode="numeric"
                      value={amount}
                      onChange={(event) => setAmount(event.target.value)}
                      placeholder="100000"
                      disabled={pendingTopup != null}
                    />
                    <p className="text-xs text-muted-foreground">
                      Minimum topup Rp 1.000. QRIS akan aktif selama 5 menit.
                    </p>
                  </div>

                  <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
                    <InfoLine
                      label="Current NexusGGR"
                      value={currencyFormatter.format(selectedToko?.nexusggrBalance ?? 0)}
                    />
                    <InfoLine
                      label="Applied rate"
                      value={`${effectiveTopupRatio}%`}
                    />
                    <InfoLine
                      label="Estimated credit"
                      value={currencyFormatter.format(estimatedCredit)}
                    />
                    <InfoLine
                      label="Estimated final balance"
                      value={currencyFormatter.format(estimatedBalance)}
                    />
                  </div>

                  <Button
                    className="w-full rounded-xl"
                    onClick={handleGenerate}
                    disabled={generateMutation.isPending || pendingTopup != null}
                  >
                    Generate QRIS
                  </Button>
                </>
              )}
            </CardContent>
          </Card>

          <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="text-base">Pending topup</CardTitle>
              <CardDescription>
                State ini direstore dari transaksi lokal pending dengan `purpose=nexusggr_topup`.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {bootstrapQuery.isLoading ? (
                <Skeleton className="h-[28rem] rounded-[1.25rem]" />
              ) : pendingTopup ? (
                <PendingTopupCard
                  pendingTopup={pendingTopup}
                  countdown={countdown}
                  onCheckStatus={() => handleCheckStatus(pendingTopup.transactionCode)}
                  onResetView={() => setDismissedCode(pendingTopup.transactionCode)}
                  isChecking={checkStatusMutation.isPending}
                />
              ) : (
                <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-12 text-center text-sm text-muted-foreground">
                  Tidak ada pending topup aktif untuk toko yang dipilih.
                </div>
              )}
            </CardContent>
          </Card>
        </CardContent>
      </Card>
    </main>
  )
}

function TopupStat({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: typeof QrCodeIcon
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

function PendingTopupCard({
  pendingTopup,
  countdown,
  onCheckStatus,
  onResetView,
  isChecking,
}: {
  pendingTopup: PendingTopup
  countdown: string
  onCheckStatus: () => Promise<void>
  onResetView: () => void
  isChecking: boolean
}) {
  return (
    <div className="grid gap-4">
      <div className="overflow-x-auto rounded-[1.25rem] border border-border/70 bg-white p-5">
        <QRCodeSVG
          value={pendingTopup.qrPayload}
          size={220}
          level="M"
          includeMargin
        />
      </div>

      <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
        <InfoLine label="Transaction code" value={pendingTopup.transactionCode} mono />
        <InfoLine label="Amount" value={currencyFormatter.format(pendingTopup.amount)} />
        <InfoLine label="Status" value={pendingTopup.status.toUpperCase()} />
        <InfoLine label="Countdown" value={countdown || "-"} />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button className="w-full rounded-xl sm:w-auto" onClick={() => void onCheckStatus()} disabled={isChecking}>
          <RefreshCcwIcon className="size-4" />
          Check payment status
        </Button>
        <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={onResetView}>
          <RotateCcwIcon className="size-4" />
          Reset view
        </Button>
      </div>
    </div>
  )
}

function InfoLine({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className={mono ? "break-all font-mono text-xs sm:text-right" : "break-words font-medium sm:text-right"}>
        {value}
      </span>
    </div>
  )
}
