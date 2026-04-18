import { useDeferredValue, useEffect, useMemo, useState } from "react"

import { zodResolver } from "@hookform/resolvers/zod"
import { format } from "date-fns"
import {
  Building2Icon,
  CreditCardIcon,
  PlusIcon,
  RefreshCcwIcon,
  SearchIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { useForm, useWatch } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { useAuthBootstrap } from "@/features/auth/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import {
  useBanksQuery,
  useCreateBankMutation,
  useInquiryBankMutation,
  useUpdateBankMutation,
} from "@/features/banks/queries"
import type { BankRecord } from "@/features/banks/api"
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

const bankOptions = [
  { code: "002", name: "BRI" },
  { code: "008", name: "Mandiri" },
  { code: "009", name: "BNI" },
  { code: "014", name: "BCA" },
  { code: "501", name: "Blu BCA Digital" },
  { code: "022", name: "CIMB" },
  { code: "013", name: "Permata" },
  { code: "111", name: "DKI" },
  { code: "451", name: "BSI" },
  { code: "542", name: "JAGO" },
  { code: "490", name: "NEO" },
] as const

const bankFormSchema = z.object({
  userId: z.string().optional(),
  bankCode: z.string().trim().min(1, "Bank wajib dipilih."),
  accountNumber: z.string().trim().min(1, "Nomor rekening wajib diisi."),
  accountName: z.string().trim().min(1, "Nama rekening wajib diisi."),
})

type BankFormValues = z.infer<typeof bankFormSchema>

type BankFilters = {
  search: string
  ownerId: string
  page: number
  perPage: number
}

const defaultFilters: BankFilters = {
  search: "",
  ownerId: "",
  page: 1,
  perPage: 25,
}

const defaultFormValues: BankFormValues = {
  userId: "",
  bankCode: "",
  accountNumber: "",
  accountName: "",
}

export function BanksPage() {
  const authBootstrap = useAuthBootstrap()
  const [filters, setFilters] = useState<BankFilters>(defaultFilters)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedBank, setSelectedBank] = useState<BankRecord | null>(null)
  const [mode, setMode] = useState<"create" | "edit">("create")
  const deferredSearch = useDeferredValue(filters.search)
  const queryParams = useMemo(
    () => ({
      ...filters,
      search: deferredSearch,
    }),
    [deferredSearch, filters],
  )

  const form = useForm<BankFormValues>({
    resolver: zodResolver(bankFormSchema),
    defaultValues: defaultFormValues,
  })
  const watchedUserId = useWatch({
    control: form.control,
    name: "userId",
  })
  const watchedBankCode = useWatch({
    control: form.control,
    name: "bankCode",
  })
  const watchedAccountNumber = useWatch({
    control: form.control,
    name: "accountNumber",
  })
  const watchedAccountName = useWatch({
    control: form.control,
    name: "accountName",
  })

  const isGlobalRole =
    authBootstrap.data?.user?.role === "dev" ||
    authBootstrap.data?.user?.role === "superadmin"

  const banksQuery = useBanksQuery(queryParams)
  const createMutation = useCreateBankMutation()
  const updateMutation = useUpdateBankMutation()
  const inquiryMutation = useInquiryBankMutation()

  useEffect(() => {
    if (mode === "create" || !selectedBank) {
      form.reset(defaultFormValues)
      return
    }

    form.reset({
      userId: String(selectedBank.userId),
      bankCode: selectedBank.bankCode,
      accountNumber: selectedBank.accountNumber,
      accountName: selectedBank.accountName,
    })
  }, [form, mode, selectedBank])

  function updateFilter<Key extends keyof BankFilters>(
    key: Key,
    value: BankFilters[Key],
  ) {
    setFilters((current) => ({
      ...current,
      [key]: value,
      page: key === "page" || key === "perPage" ? current.page : 1,
    }))
  }

  function openCreateDialog() {
    setMode("create")
    setSelectedBank(null)
    form.reset(defaultFormValues)
    setDialogOpen(true)
  }

  function openEditDialog(record: BankRecord) {
    setMode("edit")
    setSelectedBank(record)
    setDialogOpen(true)
  }

  function closeDialog(nextOpen: boolean) {
    setDialogOpen(nextOpen)
    if (!nextOpen) {
      setMode("create")
      setSelectedBank(null)
      form.reset(defaultFormValues)
    }
  }

  async function handleInquiry() {
    try {
      const response = await inquiryMutation.mutateAsync({
        userId: isGlobalRole && watchedUserId ? Number(watchedUserId) : undefined,
        bankCode: watchedBankCode,
        accountNumber: watchedAccountNumber,
      })
      form.setValue("accountName", response.data.accountName, {
        shouldDirty: true,
        shouldValidate: true,
      })
      toast.success("Rekening berhasil diverifikasi.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Inquiry rekening gagal.",
      )
    }
  }

  const handleSubmit = form.handleSubmit(async (values) => {
    const payload = {
      userId: isGlobalRole && values.userId ? Number(values.userId) : undefined,
      bankCode: values.bankCode,
      accountNumber: values.accountNumber.trim(),
      accountName: values.accountName.trim(),
    }

    try {
      if (mode === "create") {
        await createMutation.mutateAsync(payload)
        toast.success("Rekening berhasil dibuat.")
      } else if (selectedBank != null) {
        await updateMutation.mutateAsync({
          bankId: selectedBank.id,
          payload,
        })
        toast.success("Rekening berhasil diperbarui.")
      }
      closeDialog(false)
    } catch (error) {
      if (isBackofficeRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          if (field in values) {
            form.setError(field as keyof BankFormValues, { message })
          }
        }
      }

      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal menyimpan rekening.",
      )
    }
  })

  const owners = banksQuery.data?.filters.owners ?? []
  const records = banksQuery.data?.data ?? []
  const meta = banksQuery.data?.meta

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 lg:grid-cols-3">
        <StatCard
          title="Total accounts"
          value={String(meta?.total ?? 0)}
          description="Seluruh rekening tujuan aktif yang tersedia untuk workflow withdrawal."
          icon={CreditCardIcon}
        />
        <StatCard
          title="Owner scope"
          value={isGlobalRole ? "Global" : "Owner"}
          description="Role dev/superadmin melihat seluruh rekening, role lain tetap owner-scoped."
          icon={Building2Icon}
        />
        <StatCard
          title="Inquiry flow"
          value="Ready"
          description="Validasi rekening tetap lewat adapter backend agar parity ke provider tetap terjaga."
          icon={ShieldCheckIcon}
        />
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>Banks</CardTitle>
            <CardDescription>
              Daftar rekening tujuan dengan inquiry rekening parity-friendly seperti legacy Filament.
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                className="w-full rounded-xl sm:w-auto"
                onClick={() => banksQuery.refetch()}
              >
              <RefreshCcwIcon className="size-4" />
              Refresh
            </Button>
            <Button className="w-full rounded-xl sm:w-auto" onClick={openCreateDialog}>
              <PlusIcon className="size-4" />
              Tambah rekening
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-3 xl:grid-cols-[1.2fr_0.8fr_0.6fr_0.4fr]">
            <div className="relative">
              <SearchIcon className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={filters.search}
                onChange={(event) => updateFilter("search", event.target.value)}
                placeholder="Cari bank, rekening, pemilik, atau owner..."
                className="pl-9"
              />
            </div>
            <Select
              value={filters.ownerId || "all"}
              onValueChange={(value) =>
                updateFilter("ownerId", value === "all" ? "" : value)
              }
              disabled={!isGlobalRole}
            >
              <SelectTrigger>
                <SelectValue placeholder="Semua owner" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua owner</SelectItem>
                {owners.map((owner) => (
                  <SelectItem key={owner.id} value={String(owner.id)}>
                    {owner.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={String(filters.perPage)}
              onValueChange={(value) => updateFilter("perPage", Number(value))}
            >
              <SelectTrigger>
                <SelectValue placeholder="Per page" />
              </SelectTrigger>
              <SelectContent>
                {[10, 25, 50, 100].map((value) => (
                  <SelectItem key={value} value={String(value)}>
                    {value} / page
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="flex items-center justify-end text-sm text-muted-foreground">
              {meta ? `Page ${meta.page} / ${Math.max(meta.totalPages, 1)}` : "Loading"}
            </div>
          </div>

          {banksQuery.isLoading ? (
            <Skeleton className="h-[28rem] rounded-[1.5rem]" />
          ) : (
            <div className="overflow-x-auto rounded-[1.5rem] border border-border/70">
              <Table className={isGlobalRole ? "min-w-[64rem]" : "min-w-[54rem]"}>
                <TableHeader>
                  <TableRow>
                    <TableHead>Bank</TableHead>
                    <TableHead>Account</TableHead>
                    <TableHead>Account name</TableHead>
                    {isGlobalRole ? <TableHead>Owner</TableHead> : null}
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {records.length ? records.map((record) => (
                    <TableRow key={record.id}>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">{record.bankName}</span>
                          <span className="text-xs text-muted-foreground">{record.bankCode}</span>
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{record.accountNumber}</TableCell>
                      <TableCell>{record.accountName}</TableCell>
                      {isGlobalRole ? (
                        <TableCell>
                          <div className="flex flex-col">
                            <span className="font-medium">{record.ownerName}</span>
                            <span className="text-xs text-muted-foreground">{record.ownerUsername}</span>
                          </div>
                        </TableCell>
                      ) : null}
                      <TableCell className="text-sm text-muted-foreground">
                        {format(new Date(record.updatedAt), "dd MMM yyyy HH:mm")}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          className="rounded-xl"
                          onClick={() => openEditDialog(record)}
                        >
                          Edit
                        </Button>
                      </TableCell>
                    </TableRow>
                  )) : (
                    <TableRow>
                      <TableCell colSpan={isGlobalRole ? 6 : 5} className="py-12 text-center text-sm text-muted-foreground">
                        Tidak ada rekening yang cocok dengan filter saat ini.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          )}

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              Total {meta?.total ?? 0} rekening ditemukan.
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={(meta?.page ?? 1) <= 1}
                onClick={() => updateFilter("page", Math.max(1, filters.page - 1))}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={meta == null || meta.page >= meta.totalPages}
                onClick={() => updateFilter("page", filters.page + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={closeDialog}>
        <DialogContent
          data-testid="banks-dialog"
          className="max-h-[92svh] overflow-y-auto sm:max-w-2xl"
        >
          <DialogHeader>
            <DialogTitle>
              {mode === "create" ? "Tambah rekening" : "Edit rekening"}
            </DialogTitle>
            <DialogDescription>
              Inquiry rekening tetap berjalan melalui backend agar account name selalu sinkron dengan provider.
            </DialogDescription>
          </DialogHeader>

          <form className="grid gap-4" onSubmit={handleSubmit}>
            {isGlobalRole ? (
              <div className="space-y-2">
                <Label htmlFor="bank-form-owner">Owner</Label>
                <Select
                  value={watchedUserId || ""}
                  onValueChange={(value) => form.setValue("userId", value, { shouldDirty: true })}
                >
                  <SelectTrigger
                    id="bank-form-owner"
                    data-testid="bank-form-owner"
                    className="w-full"
                  >
                    <SelectValue placeholder="Pilih owner" />
                  </SelectTrigger>
                  <SelectContent>
                    {owners.map((owner) => (
                      <SelectItem key={owner.id} value={String(owner.id)}>
                        {owner.username} ({owner.name})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {form.formState.errors.userId ? (
                  <p className="text-xs text-destructive">{form.formState.errors.userId.message}</p>
                ) : null}
              </div>
            ) : null}

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="bank-form-bank-code">Bank</Label>
                <Select
                  value={watchedBankCode || ""}
                  onValueChange={(value) => {
                    form.setValue("bankCode", value, { shouldDirty: true, shouldValidate: true })
                    form.setValue("accountName", "", { shouldDirty: true })
                  }}
                >
                  <SelectTrigger
                    id="bank-form-bank-code"
                    data-testid="bank-form-bank-code"
                    className="w-full"
                  >
                    <SelectValue placeholder="Pilih bank" />
                  </SelectTrigger>
                  <SelectContent>
                    {bankOptions.map((bank) => (
                      <SelectItem key={bank.code} value={bank.code}>
                        {bank.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {form.formState.errors.bankCode ? (
                  <p className="text-xs text-destructive">{form.formState.errors.bankCode.message}</p>
                ) : null}
              </div>

              <div className="space-y-2">
                <Label htmlFor="bank-form-account-number">Nomor rekening</Label>
                <Input
                  id="bank-form-account-number"
                  data-testid="bank-form-account-number"
                  value={watchedAccountNumber || ""}
                  onChange={(event) => {
                    form.setValue("accountNumber", event.target.value, {
                      shouldDirty: true,
                      shouldValidate: true,
                    })
                    form.setValue("accountName", "", { shouldDirty: true })
                  }}
                  placeholder="1234567890"
                />
                {form.formState.errors.accountNumber ? (
                  <p className="text-xs text-destructive">{form.formState.errors.accountNumber.message}</p>
                ) : null}
              </div>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                className="w-full rounded-xl sm:w-auto"
                onClick={() => void handleInquiry()}
                disabled={inquiryMutation.isPending}
              >
                <ShieldCheckIcon className="size-4" />
                Cek rekening
              </Button>
              <Badge variant="secondary" className="rounded-lg px-3 py-1">
                Inquiry memakai nominal validasi Rp 25.000 seperti legacy.
              </Badge>
            </div>

            <div className="space-y-2">
              <Label htmlFor="bank-form-account-name">Nama rekening</Label>
              <Input
                id="bank-form-account-name"
                data-testid="bank-form-account-name"
                value={watchedAccountName || ""}
                onChange={(event) =>
                  form.setValue("accountName", event.target.value, {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
                placeholder="Akan terisi setelah inquiry"
              />
              {form.formState.errors.accountName ? (
                <p className="text-xs text-destructive">{form.formState.errors.accountName.message}</p>
              ) : null}
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button type="button" variant="outline" className="w-full rounded-xl sm:w-auto" onClick={() => closeDialog(false)}>
                Batal
              </Button>
              <Button
                type="submit"
                data-testid="bank-form-submit"
                className="w-full rounded-xl sm:w-auto"
                disabled={createMutation.isPending || updateMutation.isPending}
              >
                {mode === "create" ? "Simpan rekening" : "Perbarui rekening"}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </main>
  )
}

function StatCard({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: typeof CreditCardIcon
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
