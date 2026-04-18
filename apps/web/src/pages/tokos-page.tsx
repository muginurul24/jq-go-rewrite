import { useDeferredValue, useEffect, useMemo, useState } from "react"

import { zodResolver } from "@hookform/resolvers/zod"
import { format } from "date-fns"
import {
  CopyIcon,
  KeyRoundIcon,
  PlusIcon,
  RefreshCcwIcon,
  SearchIcon,
  ShieldCheckIcon,
  StoreIcon,
} from "lucide-react"
import { useForm, useWatch } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { useAuthBootstrap } from "@/features/auth/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import {
  useCreateTokoMutation,
  useRegenerateTokoTokenMutation,
  useTokoDetailQuery,
  useTokosQuery,
  useUpdateTokoMutation,
} from "@/features/tokos/queries"
import type { TokoRecord } from "@/features/tokos/api"
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

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

const tokoFormSchema = z.object({
  userId: z.string().optional(),
  name: z.string().trim().min(1, "Nama toko wajib diisi."),
  callbackUrl: z.string().trim().max(2048, "Callback URL terlalu panjang.").optional(),
  status: z.enum(["active", "inactive"]),
})

type TokoFormValues = z.infer<typeof tokoFormSchema>

type TokoFilters = {
  search: string
  status: string
  ownerId: string
  page: number
  perPage: number
}

const defaultFilters: TokoFilters = {
  search: "",
  status: "",
  ownerId: "",
  page: 1,
  perPage: 25,
}

const defaultFormValues: TokoFormValues = {
  userId: "",
  name: "",
  callbackUrl: "",
  status: "active",
}

export function TokosPage() {
  const authBootstrap = useAuthBootstrap()
  const [filters, setFilters] = useState<TokoFilters>(defaultFilters)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedTokoId, setSelectedTokoId] = useState<number | null>(null)
  const [mode, setMode] = useState<"create" | "edit">("create")
  const deferredSearch = useDeferredValue(filters.search)
  const queryParams = useMemo(
    () => ({
      ...filters,
      search: deferredSearch,
    }),
    [deferredSearch, filters],
  )

  const form = useForm<TokoFormValues>({
    resolver: zodResolver(tokoFormSchema),
    defaultValues: defaultFormValues,
  })
  const watchedUserId = useWatch({
    control: form.control,
    name: "userId",
  })
  const watchedStatus = useWatch({
    control: form.control,
    name: "status",
  })

  const isGlobalRole =
    authBootstrap.data?.user?.role === "dev" ||
    authBootstrap.data?.user?.role === "superadmin"

  const tokosQuery = useTokosQuery(queryParams)
  const tokoDetailQuery = useTokoDetailQuery(
    mode === "edit" ? selectedTokoId : null,
  )
  const createMutation = useCreateTokoMutation()
  const updateMutation = useUpdateTokoMutation()
  const regenerateMutation = useRegenerateTokoTokenMutation()

  useEffect(() => {
    if (mode === "create") {
      form.reset(defaultFormValues)
      return
    }

    const detail = tokoDetailQuery.data?.data
    if (!detail) {
      return
    }

    form.reset({
      userId: String(detail.userId),
      name: detail.name,
      callbackUrl: detail.callbackUrl ?? "",
      status: detail.isActive ? "active" : "inactive",
    })
  }, [form, mode, tokoDetailQuery.data?.data])

  function updateFilter<Key extends keyof TokoFilters>(
    key: Key,
    value: TokoFilters[Key],
  ) {
    setFilters((current) => ({
      ...current,
      [key]: value,
      page: key === "page" || key === "perPage" ? current.page : 1,
    }))
  }

  function openCreateDialog() {
    setMode("create")
    setSelectedTokoId(null)
    form.reset(defaultFormValues)
    setDialogOpen(true)
  }

  function openEditDialog(tokoId: number) {
    setMode("edit")
    setSelectedTokoId(tokoId)
    setDialogOpen(true)
  }

  function closeDialog(nextOpen: boolean) {
    setDialogOpen(nextOpen)
    if (!nextOpen) {
      setSelectedTokoId(null)
      setMode("create")
      form.reset(defaultFormValues)
    }
  }

  async function copyToken(token?: string | null) {
    if (!token) {
      return
    }

    await navigator.clipboard.writeText(token)
    toast.success("Token toko disalin.")
  }

  const handleSubmit = form.handleSubmit(async (values) => {
    const payload = {
      userId:
        isGlobalRole && values.userId ? Number(values.userId) : undefined,
      name: values.name.trim(),
      callbackUrl: values.callbackUrl?.trim() || null,
      isActive: values.status === "active",
    }

    try {
      if (mode === "create") {
        const response = await createMutation.mutateAsync(payload)
        toast.success("Toko berhasil dibuat.")
        setMode("edit")
        setSelectedTokoId(response.data.id)
        await copyToken(response.data.token)
      } else if (selectedTokoId != null) {
        await updateMutation.mutateAsync({
          tokoId: selectedTokoId,
          payload,
        })
        toast.success("Toko berhasil diperbarui.")
      }
    } catch (error) {
      if (isBackofficeRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          const normalizedField =
            field === "callbackUrl" ? "callbackUrl" : field
          if (normalizedField in values) {
            form.setError(normalizedField as keyof TokoFormValues, { message })
          }
        }
      }

      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal menyimpan toko.",
      )
    }
  })

  async function handleRegenerateToken() {
    if (selectedTokoId == null) {
      return
    }

    try {
      const response = await regenerateMutation.mutateAsync(selectedTokoId)
      toast.success("Token toko berhasil diganti.")
      await copyToken(response.data.token)
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal mengganti token toko.",
      )
    }
  }

  const owners = tokosQuery.data?.filters.owners ?? []
  const records = tokosQuery.data?.data ?? []
  const summary = tokosQuery.data?.summary

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          title="Total toko"
          value={String(summary?.totalTokos ?? 0)}
          description="Konfigurasi toko aktif maupun nonaktif yang terlihat oleh actor saat ini."
          icon={StoreIcon}
        />
        <SummaryCard
          title="Toko aktif"
          value={String(summary?.activeTokos ?? 0)}
          description="Toko yang saat ini bisa melayani flow API dan operasional callback."
          icon={ShieldCheckIcon}
        />
        <SummaryCard
          title="Pending balance"
          value={currencyFormatter.format(summary?.totalPending ?? 0)}
          description="Agregat pending balance seluruh toko yang lolos owner scoping."
          icon={RefreshCcwIcon}
        />
        <SummaryCard
          title="NexusGGR balance"
          value={currencyFormatter.format(summary?.totalNexusggr ?? 0)}
          description="Saldo agent lokal yang dipakai untuk jalur deposit dan withdraw player."
          icon={KeyRoundIcon}
        />
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Tokos control</CardTitle>
              <CardDescription>
                Manage callback URL, owner, status aktif, preview token, dan balance
                bootstrap toko seperti workflow Filament legacy.
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                className="w-full rounded-xl sm:w-auto"
                onClick={() => tokosQuery.refetch()}
              >
                <RefreshCcwIcon className="size-4" />
                Refresh
              </Button>
              <Button className="w-full rounded-xl sm:w-auto" onClick={openCreateDialog}>
                <PlusIcon className="size-4" />
                Toko baru
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
              placeholder="Cari toko, owner, callback, atau token preview"
              className="pl-9"
            />
          </div>

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
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
          </Select>

          {isGlobalRole ? (
            <Select
              value={filters.ownerId || "all"}
              onValueChange={(value) =>
                updateFilter("ownerId", value === "all" ? "" : value)
              }
            >
              <SelectTrigger className="w-full">
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
          ) : (
            <Card className="rounded-2xl border-dashed border-border/70 bg-background/40">
              <CardContent className="flex h-full items-center px-4 py-3 text-sm text-muted-foreground">
                Owner scoping otomatis mengikuti akun saat ini.
              </CardContent>
            </Card>
          )}

          <Select
            value={String(filters.perPage)}
            onValueChange={(value) => updateFilter("perPage", Number(value))}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Rows" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10 rows</SelectItem>
              <SelectItem value="25">25 rows</SelectItem>
              <SelectItem value="50">50 rows</SelectItem>
              <SelectItem value="100">100 rows</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Daftar toko</CardTitle>
          <CardDescription>
            Grid owner-scoped dengan preview token, callback host, dan tiga bucket balance
            yang tetap dipisah: pending, settle, dan NexusGGR.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
            <Table className={isGlobalRole ? "min-w-[74rem]" : "min-w-[64rem]"}>
              <TableHeader>
                <TableRow>
                  <TableHead>Toko</TableHead>
                  {isGlobalRole ? <TableHead>Owner</TableHead> : null}
                  <TableHead>Callback</TableHead>
                  <TableHead>Token</TableHead>
                  <TableHead className="text-right">Pending</TableHead>
                  <TableHead className="text-right">Settle</TableHead>
                  <TableHead className="text-right">NexusGGR</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokosQuery.isLoading
                  ? Array.from({ length: filters.perPage }).map((_, index) => (
                      <TableRow key={index}>
                        <TableCell colSpan={isGlobalRole ? 10 : 9}>
                          <Skeleton className="h-10 w-full rounded-xl" />
                        </TableCell>
                      </TableRow>
                    ))
                  : records.map((record) => (
                      <TableRow key={record.id}>
                        <TableCell>
                          <div className="space-y-1">
                            <p className="font-medium">{record.name}</p>
                            <p className="text-xs text-muted-foreground">
                              ID #{record.id}
                            </p>
                          </div>
                        </TableCell>
                        {isGlobalRole ? (
                          <TableCell>
                            <div className="space-y-1">
                              <p className="font-medium">{record.ownerUsername}</p>
                              <p className="text-xs text-muted-foreground">
                                {record.ownerName}
                              </p>
                            </div>
                          </TableCell>
                        ) : null}
                        <TableCell className="max-w-[220px] truncate text-sm text-muted-foreground">
                          {record.callbackHost || "-"}
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {record.tokenPreview ?? "-"}
                        </TableCell>
                        <TableCell className="text-right">
                          {currencyFormatter.format(record.balances.pending)}
                        </TableCell>
                        <TableCell className="text-right">
                          {currencyFormatter.format(record.balances.settle)}
                        </TableCell>
                        <TableCell className="text-right">
                          {currencyFormatter.format(record.balances.nexusggr)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={record.isActive ? "default" : "outline"}
                            className="rounded-full"
                          >
                            {record.isActive ? "Active" : "Inactive"}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {format(new Date(record.updatedAt), "dd MMM yyyy HH:mm")}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="outline"
                            className="rounded-xl"
                            onClick={() => openEditDialog(record.id)}
                          >
                            Manage
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
              </TableBody>
            </Table>
          </div>

          {!tokosQuery.isLoading && records.length === 0 ? (
            <Card className="rounded-[1.25rem] border-dashed border-border/70 bg-background/40">
              <CardContent className="py-10 text-center">
                <p className="font-medium">Belum ada toko yang cocok</p>
                <p className="mt-2 text-sm text-muted-foreground">
                  Ubah filter atau buat toko baru untuk mulai menghubungkan callback dan token API.
                </p>
              </CardContent>
            </Card>
          ) : null}

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">
              Page {tokosQuery.data?.meta.page ?? filters.page} dari{" "}
              {tokosQuery.data?.meta.totalPages ?? 1} • {tokosQuery.data?.meta.total ?? 0} toko
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={(tokosQuery.data?.meta.page ?? filters.page) <= 1}
                onClick={() =>
                  updateFilter(
                    "page",
                    Math.max(1, (tokosQuery.data?.meta.page ?? filters.page) - 1),
                  )
                }
              >
                Previous
              </Button>
              <Button
                variant="outline"
                className="rounded-xl"
                disabled={
                  (tokosQuery.data?.meta.page ?? filters.page) >=
                  (tokosQuery.data?.meta.totalPages ?? 1)
                }
                onClick={() =>
                  updateFilter(
                    "page",
                    (tokosQuery.data?.meta.page ?? filters.page) + 1,
                  )
                }
              >
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={closeDialog}>
        <DialogContent
          data-testid="tokos-dialog"
          className="max-h-[92svh] overflow-y-auto sm:max-w-3xl"
        >
          <DialogHeader>
            <DialogTitle>
              {mode === "create" ? "Create toko" : "Manage toko"}
            </DialogTitle>
            <DialogDescription>
              {mode === "create"
                ? "Buat toko baru, bootstrap balance lokal, dan generate token API baru."
                : "Edit owner, callback URL, status aktif, lalu regenerate token jika dibutuhkan."}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
            <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
              <CardHeader>
                <CardTitle className="text-base">Konfigurasi toko</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {isGlobalRole ? (
                  <div className="space-y-2">
                    <Label htmlFor="toko-form-owner">Owner</Label>
                    <Select
                      value={watchedUserId || "none"}
                      onValueChange={(value) =>
                        form.setValue("userId", value === "none" ? "" : value)
                      }
                    >
                      <SelectTrigger
                        id="toko-form-owner"
                        data-testid="toko-form-owner"
                        className="w-full"
                      >
                        <SelectValue placeholder="Pilih owner" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">Pilih owner</SelectItem>
                        {owners.map((owner) => (
                          <SelectItem key={owner.id} value={String(owner.id)}>
                            {owner.username} • {owner.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {form.formState.errors.userId ? (
                      <p className="text-sm text-destructive">
                        {form.formState.errors.userId.message}
                      </p>
                    ) : null}
                  </div>
                ) : null}

                <div className="space-y-2">
                  <Label htmlFor="toko-form-name">Nama toko</Label>
                  <Input
                    id="toko-form-name"
                    data-testid="toko-form-name"
                    {...form.register("name")}
                    placeholder="Nama toko"
                  />
                  {form.formState.errors.name ? (
                    <p className="text-sm text-destructive">
                      {form.formState.errors.name.message}
                    </p>
                  ) : null}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="toko-form-callback-url">Callback URL</Label>
                  <Input
                    id="toko-form-callback-url"
                    data-testid="toko-form-callback-url"
                    {...form.register("callbackUrl")}
                    placeholder="store.example.com/callback"
                  />
                  {form.formState.errors.callbackUrl ? (
                    <p className="text-sm text-destructive">
                      {form.formState.errors.callbackUrl.message}
                    </p>
                  ) : null}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="toko-form-status">Status</Label>
                  <Select
                    value={watchedStatus}
                    onValueChange={(value) =>
                      form.setValue("status", value as TokoFormValues["status"])
                    }
                  >
                    <SelectTrigger
                      id="toko-form-status"
                      data-testid="toko-form-status"
                      className="w-full"
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="active">Active</SelectItem>
                      <SelectItem value="inactive">Inactive</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button
                    data-testid="toko-form-submit"
                    className="w-full rounded-xl sm:w-auto"
                    onClick={handleSubmit}
                    disabled={createMutation.isPending || updateMutation.isPending}
                  >
                    {mode === "create" ? "Create toko" : "Save changes"}
                  </Button>
                  <Button
                    variant="outline"
                    className="w-full rounded-xl sm:w-auto"
                    onClick={() => closeDialog(false)}
                  >
                    Cancel
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
              <CardHeader>
                <CardTitle className="text-base">Token dan balance</CardTitle>
                <CardDescription>
                  Disclosure token hanya tersedia di detail view dan setelah create/regenerate.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {mode === "edit" && tokoDetailQuery.isLoading ? (
                  <Skeleton className="h-40 rounded-[1.25rem]" />
                ) : (
                  <TokenPanel
                    record={tokoDetailQuery.data?.data ?? null}
                    isLoading={regenerateMutation.isPending}
                    onCopy={copyToken}
                    onRegenerate={handleRegenerateToken}
                  />
                )}
              </CardContent>
            </Card>
          </div>
        </DialogContent>
      </Dialog>
    </main>
  )
}

function SummaryCard({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: typeof StoreIcon
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

function TokenPanel({
  record,
  isLoading,
  onCopy,
  onRegenerate,
}: {
  record: TokoRecord | null
  isLoading: boolean
  onCopy: (token?: string | null) => Promise<void>
  onRegenerate: () => Promise<void>
}) {
  if (!record) {
    return (
      <div className="rounded-[1.25rem] border border-dashed border-border/70 px-4 py-5 text-sm text-muted-foreground">
        Simpan toko terlebih dulu untuk mendapatkan token API dan preview balance lokal.
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="rounded-[1.25rem] border border-border/70 bg-card/80 p-4">
        <Label>Active token</Label>
        <p className="mt-2 break-all font-mono text-xs text-muted-foreground">
          {record.token ?? "-"}
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          <Button
            variant="outline"
            className="w-full rounded-xl sm:w-auto"
            onClick={() => void onCopy(record.token)}
          >
            <CopyIcon className="size-4" />
            Copy token
          </Button>
          <Button className="w-full rounded-xl sm:w-auto" onClick={onRegenerate} disabled={isLoading}>
            <KeyRoundIcon className="size-4" />
            Regenerate token
          </Button>
        </div>
      </div>

      <div className="grid gap-3">
        <BalanceLine label="Pending" value={record.balances.pending} />
        <BalanceLine label="Settle" value={record.balances.settle} />
        <BalanceLine label="NexusGGR" value={record.balances.nexusggr} />
      </div>
    </div>
  )
}

function BalanceLine({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col gap-1 rounded-[1.1rem] border border-border/70 px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="font-medium sm:text-right">{currencyFormatter.format(value)}</span>
    </div>
  )
}
