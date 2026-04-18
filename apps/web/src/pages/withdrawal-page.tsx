import { useDeferredValue, useMemo, useState } from "react"

import {
  ArrowLeftIcon,
  ArrowRightIcon,
  BadgeDollarSignIcon,
  Building2Icon,
  CreditCardIcon,
  RefreshCcwIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { toast } from "sonner"

import type {
  WithdrawalBankOption,
  WithdrawalInquiry,
  WithdrawalSubmitResponse,
  WithdrawalTokoOption,
} from "@/features/withdrawal/api"
import {
  useWithdrawalBootstrapQuery,
  useWithdrawalInquiryMutation,
  useWithdrawalSubmitMutation,
} from "@/features/withdrawal/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import { Badge } from "@/components/ui/badge"
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

type WizardStep = 1 | 2 | 3

export function WithdrawalPage() {
  const [step, setStep] = useState<WizardStep>(1)
  const [manualSelectedTokoId, setManualSelectedTokoId] = useState<number | null>(null)
  const [manualSelectedBankId, setManualSelectedBankId] = useState<number | null>(null)
  const [amountInput, setAmountInput] = useState("25000")
  const [verifiedInquiry, setVerifiedInquiry] = useState<WithdrawalInquiry | null>(null)
  const [submittedResult, setSubmittedResult] = useState<WithdrawalSubmitResponse["data"] | null>(null)

  const deferredTokoId = useDeferredValue(manualSelectedTokoId)
  const bootstrapQuery = useWithdrawalBootstrapQuery(deferredTokoId)
  const inquiryMutation = useWithdrawalInquiryMutation()
  const submitMutation = useWithdrawalSubmitMutation()

  const selectedTokoId =
    manualSelectedTokoId
    ?? bootstrapQuery.data?.data.selectedToko?.id
    ?? bootstrapQuery.data?.data.tokos[0]?.id
    ?? null

  const banks = bootstrapQuery.data?.data.banks ?? []
  const selectedBankId = manualSelectedBankId ?? banks[0]?.id ?? null
  const selectedToko = bootstrapQuery.data?.data.selectedToko
    ?? bootstrapQuery.data?.data.tokos.find((item) => item.id === selectedTokoId)
    ?? null
  const selectedBank = banks.find((item) => item.id === selectedBankId) ?? null
  const feePercentage = bootstrapQuery.data?.data.feePercentage ?? 0
  const minimumAmount = bootstrapQuery.data?.data.minimumAmount ?? 25_000
  const parsedAmount = Number(amountInput.replace(/[^\d]/g, "")) || 0
  const estimatedPlatformFee = parsedAmount > 0
    ? Math.round((parsedAmount * feePercentage) / 100)
    : 0
  const estimatedTotalDeduction = parsedAmount + estimatedPlatformFee
  const estimatedRemainingSettle = (selectedToko?.settleBalance ?? 0) - estimatedTotalDeduction

  const isBusy = inquiryMutation.isPending || submitMutation.isPending

  const summaryCards = useMemo(
    () => [
      {
        title: "Current settle",
        value: currencyFormatter.format(selectedToko?.settleBalance ?? 0),
        description: "Saldo settle toko yang akan dipotong ketika transfer berhasil diproses.",
        icon: Building2Icon,
      },
      {
        title: "Platform fee",
        value: currencyFormatter.format(estimatedPlatformFee),
        description: `Biaya platform withdrawal ${feePercentage}% sesuai konfigurasi income legacy.`,
        icon: BadgeDollarSignIcon,
      },
      {
        title: "Estimated deduction",
        value: currencyFormatter.format(estimatedTotalDeduction),
        description: "Estimasi potongan sebelum admin bank terverifikasi di step inquiry.",
        icon: CreditCardIcon,
      },
    ],
    [estimatedPlatformFee, estimatedTotalDeduction, feePercentage, selectedToko?.settleBalance],
  )

  function resetVerificationState(nextStep: WizardStep = 1) {
    setVerifiedInquiry(null)
    setSubmittedResult(null)
    setStep(nextStep)
  }

  function handleChangeToko(value: string) {
    setManualSelectedTokoId(Number(value))
    setManualSelectedBankId(null)
    resetVerificationState(1)
  }

  function handleChangeBank(value: string) {
    setManualSelectedBankId(Number(value))
    resetVerificationState(step === 3 ? 2 : step)
  }

  function handleAmountChange(value: string) {
    setAmountInput(value)
    resetVerificationState(step === 3 ? 2 : step)
  }

  async function handleRunInquiry() {
    if (selectedTokoId == null) {
      toast.error("Pilih toko terlebih dahulu.")
      return
    }

    if (selectedBankId == null) {
      toast.error("Pilih rekening tujuan terlebih dahulu.")
      return
    }

    try {
      const response = await inquiryMutation.mutateAsync({
        tokoId: selectedTokoId,
        bankId: selectedBankId,
        amount: parsedAmount,
      })
      setVerifiedInquiry(response.data.inquiry)
      setSubmittedResult(null)
      setStep(3)
      toast.success("Rekening berhasil diverifikasi.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal memverifikasi rekening tujuan.",
      )
    }
  }

  async function handleSubmitWithdrawal() {
    if (selectedTokoId == null || selectedBankId == null || verifiedInquiry == null) {
      toast.error("Inquiry belum siap diproses.")
      return
    }

    try {
      const response = await submitMutation.mutateAsync({
        tokoId: selectedTokoId,
        bankId: selectedBankId,
        amount: parsedAmount,
        inquiryId: verifiedInquiry.inquiryId,
      })
      setSubmittedResult(response.data)
      toast.success("Withdrawal berhasil dikirim ke provider.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal memproses withdrawal.",
      )
    }
  }

  function startNewWithdrawal() {
    setAmountInput(String(minimumAmount))
    setManualSelectedBankId(null)
    resetVerificationState(1)
  }

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 xl:grid-cols-3">
        {summaryCards.map(({ title, value, description, icon: Icon }) => (
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
            <CardTitle>Withdrawal wizard</CardTitle>
            <CardDescription>
              Flow ini menjaga parity legacy: pilih toko, pilih rekening + nominal, inquiry saat lanjut, lalu submit final.
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
        <CardContent className="grid gap-6">
          <WizardProgress currentStep={step} />

          {submittedResult ? (
            <SuccessPanel
              result={submittedResult}
              onReset={startNewWithdrawal}
            />
          ) : (
            <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
              <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
                <CardHeader>
                  <CardTitle className="text-base">Wizard form</CardTitle>
                  <CardDescription>
                    Step 1 memilih toko, step 2 inquiry rekening, step 3 verifikasi dan submit.
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-6">
                  {bootstrapQuery.isLoading ? (
                    <Skeleton className="h-[28rem] rounded-[1.25rem]" />
                  ) : bootstrapQuery.data?.data.tokos.length ? (
                    <>
                      {step === 1 ? (
                        <StepChooseToko
                          tokos={bootstrapQuery.data.data.tokos}
                          selectedTokoId={selectedTokoId}
                          selectedToko={selectedToko}
                          onSelectToko={handleChangeToko}
                          onNext={() => setStep(2)}
                        />
                      ) : null}

                      {step === 2 ? (
                        <StepChooseBank
                          selectedToko={selectedToko}
                          selectedBank={selectedBank}
                          banks={banks}
                          selectedBankId={selectedBankId}
                          amountInput={amountInput}
                          minimumAmount={minimumAmount}
                          estimatedPlatformFee={estimatedPlatformFee}
                          estimatedTotalDeduction={estimatedTotalDeduction}
                          estimatedRemainingSettle={estimatedRemainingSettle}
                          isBusy={isBusy}
                          onBack={() => setStep(1)}
                          onAmountChange={handleAmountChange}
                          onSelectBank={handleChangeBank}
                          onVerify={handleRunInquiry}
                        />
                      ) : null}

                      {step === 3 && verifiedInquiry ? (
                        <StepVerification
                          selectedToko={selectedToko}
                          selectedBank={selectedBank}
                          inquiry={verifiedInquiry}
                          isBusy={isBusy}
                          onBack={() => setStep(2)}
                          onSubmit={handleSubmitWithdrawal}
                        />
                      ) : null}
                    </>
                  ) : (
                    <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-12 text-center text-sm text-muted-foreground">
                      Tidak ada toko aktif yang bisa dipakai untuk withdrawal.
                    </div>
                  )}
                </CardContent>
              </Card>

              <AuditPanel
                step={step}
                selectedToko={selectedToko}
                selectedBank={selectedBank}
                verifiedInquiry={verifiedInquiry}
                minimumAmount={minimumAmount}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </main>
  )
}

function WizardProgress({ currentStep }: { currentStep: WizardStep }) {
  const steps = [
    { id: 1, title: "Pilih toko", icon: Building2Icon },
    { id: 2, title: "Pilih bank", icon: CreditCardIcon },
    { id: 3, title: "Verifikasi", icon: ShieldCheckIcon },
  ] as const

  return (
    <div className="grid gap-3 md:grid-cols-3">
      {steps.map(({ id, title, icon: Icon }) => {
        const active = currentStep === id
        const completed = currentStep > id

        return (
          <div
            key={id}
            className={[
              "rounded-[1.25rem] border px-4 py-4 transition-colors",
              active ? "border-primary/60 bg-primary/5" : "border-border/70 bg-background/40",
            ].join(" ")}
          >
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <span className="inline-flex size-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                  <Icon className="size-5" />
                </span>
                <div>
                  <p className="text-sm font-medium">{title}</p>
                  <p className="text-xs text-muted-foreground">
                    {completed ? "Selesai" : active ? "Aktif" : "Menunggu"}
                  </p>
                </div>
              </div>
              <Badge variant={completed || active ? "default" : "secondary"}>
                {id}
              </Badge>
            </div>
          </div>
        )
      })}
    </div>
  )
}

function StepChooseToko({
  tokos,
  selectedTokoId,
  selectedToko,
  onSelectToko,
  onNext,
}: {
  tokos: WithdrawalTokoOption[]
  selectedTokoId: number | null
  selectedToko: WithdrawalTokoOption | null
  onSelectToko: (value: string) => void
  onNext: () => void
}) {
  return (
    <div className="grid gap-4">
      <div className="space-y-2">
        <Label>Toko</Label>
        <Select
          value={selectedTokoId != null ? String(selectedTokoId) : ""}
          onValueChange={onSelectToko}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Pilih toko" />
          </SelectTrigger>
          <SelectContent>
            {tokos.map((toko) => (
              <SelectItem key={toko.id} value={String(toko.id)}>
                {toko.name} ({toko.ownerUsername})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
        <InfoLine label="Owner" value={selectedToko?.ownerUsername ?? "-"} />
        <InfoLine label="Current settle" value={currencyFormatter.format(selectedToko?.settleBalance ?? 0)} />
      </div>

      <Button className="w-full rounded-xl sm:w-auto" onClick={onNext} disabled={selectedToko == null}>
        Lanjut ke rekening tujuan
        <ArrowRightIcon className="size-4" />
      </Button>
    </div>
  )
}

function StepChooseBank({
  selectedToko,
  selectedBank,
  banks,
  selectedBankId,
  amountInput,
  minimumAmount,
  estimatedPlatformFee,
  estimatedTotalDeduction,
  estimatedRemainingSettle,
  isBusy,
  onBack,
  onAmountChange,
  onSelectBank,
  onVerify,
}: {
  selectedToko: WithdrawalTokoOption | null
  selectedBank: WithdrawalBankOption | null
  banks: WithdrawalBankOption[]
  selectedBankId: number | null
  amountInput: string
  minimumAmount: number
  estimatedPlatformFee: number
  estimatedTotalDeduction: number
  estimatedRemainingSettle: number
  isBusy: boolean
  onBack: () => void
  onAmountChange: (value: string) => void
  onSelectBank: (value: string) => void
  onVerify: () => Promise<void>
}) {
  return (
    <div className="grid gap-4">
      <div className="space-y-2">
        <Label>Rekening tujuan</Label>
        <Select
          value={selectedBankId != null ? String(selectedBankId) : ""}
          onValueChange={onSelectBank}
          disabled={banks.length === 0}
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Pilih rekening tujuan" />
          </SelectTrigger>
          <SelectContent>
            {banks.map((bank) => (
              <SelectItem key={bank.id} value={String(bank.id)}>
                {bank.bankName} - {bank.accountNumber} ({bank.accountName})
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>Nominal withdrawal</Label>
        <Input
          inputMode="numeric"
          value={amountInput}
          onChange={(event) => onAmountChange(event.target.value)}
          placeholder={String(minimumAmount)}
        />
        <p className="text-xs text-muted-foreground">
          Minimum withdrawal {currencyFormatter.format(minimumAmount)}. Inquiry akan dijalankan saat klik lanjut.
        </p>
      </div>

      <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
        <InfoLine label="Toko" value={selectedToko?.name ?? "-"} />
        <InfoLine label="Rekening" value={selectedBank ? `${selectedBank.bankName} - ${selectedBank.accountNumber}` : "-"} />
        <InfoLine label="Biaya platform" value={currencyFormatter.format(estimatedPlatformFee)} />
        <InfoLine label="Estimasi total dipotong" value={currencyFormatter.format(estimatedTotalDeduction)} />
        <InfoLine
          label="Estimasi sisa settle"
          value={currencyFormatter.format(estimatedRemainingSettle)}
          danger={estimatedRemainingSettle < 0}
        />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={onBack}>
          <ArrowLeftIcon className="size-4" />
          Kembali
        </Button>
        <Button
          className="w-full rounded-xl sm:w-auto"
          onClick={() => void onVerify()}
          disabled={selectedBank == null || isBusy}
        >
          Lanjut ke verifikasi
          <ArrowRightIcon className="size-4" />
        </Button>
      </div>
    </div>
  )
}

function StepVerification({
  selectedToko,
  selectedBank,
  inquiry,
  isBusy,
  onBack,
  onSubmit,
}: {
  selectedToko: WithdrawalTokoOption | null
  selectedBank: WithdrawalBankOption | null
  inquiry: WithdrawalInquiry
  isBusy: boolean
  onBack: () => void
  onSubmit: () => Promise<void>
}) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
        <InfoLine label="Toko" value={selectedToko?.name ?? "-"} />
        <InfoLine label="Bank" value={inquiry.bankName} />
        <InfoLine label="Nomor rekening" value={inquiry.accountNumber} mono />
        <InfoLine label="Pemilik rekening" value={inquiry.accountName} />
        <InfoLine label="Nominal transfer" value={currencyFormatter.format(inquiry.amount)} />
        <InfoLine label="Biaya admin bank" value={currencyFormatter.format(inquiry.bankFee)} />
        <InfoLine label="Biaya platform" value={currencyFormatter.format(inquiry.platformFee)} />
        <InfoLine label="Saldo settle saat ini" value={currencyFormatter.format(inquiry.currentSettleBalance)} />
        <InfoLine label="Total final dipotong" value={currencyFormatter.format(inquiry.finalTotalDeduction)} />
        <InfoLine
          label="Sisa settle setelah transfer"
          value={currencyFormatter.format(inquiry.finalRemainingSettle)}
          danger={inquiry.finalRemainingSettle < 0}
        />
        <InfoLine label="Partner ref no" value={inquiry.partnerRefNo} mono />
        <InfoLine label="Selected bank" value={selectedBank ? `${selectedBank.bankName} - ${selectedBank.accountNumber}` : "-"} />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button variant="outline" className="w-full rounded-xl sm:w-auto" onClick={onBack}>
          <ArrowLeftIcon className="size-4" />
          Kembali
        </Button>
        <Button
          className="w-full rounded-xl sm:w-auto"
          onClick={() => void onSubmit()}
          disabled={isBusy || inquiry.finalRemainingSettle < 0}
        >
          <ShieldCheckIcon className="size-4" />
          Proses transfer
        </Button>
      </div>
    </div>
  )
}

function SuccessPanel({
  result,
  onReset,
}: {
  result: WithdrawalSubmitResponse["data"]
  onReset: () => void
}) {
  return (
    <div className="grid gap-4 rounded-[1.5rem] border border-emerald-500/25 bg-emerald-500/5 px-5 py-5">
      <div className="space-y-1">
        <p className="text-sm font-medium uppercase tracking-[0.22em] text-emerald-700 dark:text-emerald-300">
          Transfer berhasil
        </p>
        <h2 className="text-2xl font-semibold tracking-tight">
          Dana {currencyFormatter.format(result.transaction.amount)} telah dikirim untuk proses disbursement.
        </h2>
        <p className="text-sm text-muted-foreground">
          Status lokal saat ini masih <span className="font-medium">{result.transaction.status}</span> sampai callback disbursement masuk.
        </p>
      </div>

      <div className="grid gap-3 rounded-[1.25rem] border border-emerald-500/20 bg-background/70 px-4 py-4">
        <InfoLine label="Toko" value={result.selectedToko?.name ?? "-"} />
        <InfoLine label="Bank" value={result.inquiry.bankName} />
        <InfoLine label="Nomor rekening" value={result.inquiry.accountNumber} mono />
        <InfoLine label="Partner ref no" value={result.transaction.code} mono />
        <InfoLine label="Total dipotong" value={currencyFormatter.format(result.inquiry.finalTotalDeduction)} />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button className="w-full rounded-xl sm:w-auto" onClick={onReset}>
          Withdrawal lagi
        </Button>
      </div>
    </div>
  )
}

function AuditPanel({
  step,
  selectedToko,
  selectedBank,
  verifiedInquiry,
  minimumAmount,
}: {
  step: WizardStep
  selectedToko: WithdrawalTokoOption | null
  selectedBank: WithdrawalBankOption | null
  verifiedInquiry: WithdrawalInquiry | null
  minimumAmount: number
}) {
  return (
    <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
      <CardHeader>
        <CardTitle className="text-base">Operational notes</CardTitle>
        <CardDescription>
          Ringkasan parity flow withdrawal legacy agar operator tahu apa yang sedang terjadi.
        </CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4">
        <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
          <InfoLine label="Current step" value={`Step ${step}`} />
          <InfoLine label="Selected toko" value={selectedToko?.name ?? "-"} />
          <InfoLine label="Selected bank" value={selectedBank ? `${selectedBank.bankName} - ${selectedBank.accountNumber}` : "-"} />
          <InfoLine label="Minimum amount" value={currencyFormatter.format(minimumAmount)} />
        </div>

        <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4 text-sm text-muted-foreground">
          <p>1. Step bank tidak memanggil provider realtime sampai operator klik lanjut.</p>
          <p>2. Inquiry menghitung final bank fee dan menyimpan state verifikasi di backend.</p>
          <p>3. Submit akan memanggil provider lalu mengunci row balance settle sebelum potongan final dibuat.</p>
          <p>4. Status transaksi tetap pending sampai callback disbursement diterima worker.</p>
        </div>

        {verifiedInquiry ? (
          <div className="grid gap-3 rounded-[1.25rem] border border-border/70 px-4 py-4">
            <InfoLine label="Inquiry ref" value={verifiedInquiry.partnerRefNo} mono />
            <InfoLine label="Bank fee" value={currencyFormatter.format(verifiedInquiry.bankFee)} />
            <InfoLine label="Platform fee" value={currencyFormatter.format(verifiedInquiry.platformFee)} />
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function InfoLine({
  label,
  value,
  mono = false,
  danger = false,
}: {
  label: string
  value: string
  mono?: boolean
  danger?: boolean
}) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span
        className={[
          mono ? "break-all font-mono text-xs sm:text-right" : "break-words font-medium sm:text-right",
          danger ? "text-destructive" : "",
        ].join(" ")}
      >
        {value}
      </span>
    </div>
  )
}
