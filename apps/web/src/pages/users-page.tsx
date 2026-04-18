import { useDeferredValue, useEffect, useMemo, useState } from "react"

import { zodResolver } from "@hookform/resolvers/zod"
import { format } from "date-fns"
import {
  PlusIcon,
  RefreshCcwIcon,
  SearchIcon,
  ShieldCheckIcon,
  UsersRoundIcon,
} from "lucide-react"
import { useForm, useWatch } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { useAuthBootstrap } from "@/features/auth/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"
import {
  useCreateUserMutation,
  useUserDetailQuery,
  useUsersQuery,
  useUpdateUserMutation,
} from "@/features/users/queries"
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

const userFormSchema = z.object({
  username: z.string().trim().min(1, "Username wajib diisi."),
  name: z.string().trim().min(1, "Nama wajib diisi."),
  email: z.string().email("Email tidak valid."),
  role: z.string().trim().min(1, "Role wajib dipilih."),
  status: z.enum(["active", "inactive"]),
  password: z.string().optional(),
})

type UserFormValues = z.infer<typeof userFormSchema>

type UserFilters = {
  search: string
  role: string
  status: string
  page: number
  perPage: number
}

const defaultFilters: UserFilters = {
  search: "",
  role: "",
  status: "",
  page: 1,
  perPage: 25,
}

const defaultFormValues: UserFormValues = {
  username: "",
  name: "",
  email: "",
  role: "user",
  status: "active",
  password: "",
}

export function UsersPage() {
  const authBootstrap = useAuthBootstrap()
  const actorRole = authBootstrap.data?.user?.role
  const canManageUsers = actorRole === "dev" || actorRole === "superadmin"

  const [filters, setFilters] = useState<UserFilters>(defaultFilters)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [mode, setMode] = useState<"create" | "edit">("create")
  const deferredSearch = useDeferredValue(filters.search)
  const queryParams = useMemo(
    () => ({
      ...filters,
      search: deferredSearch,
    }),
    [deferredSearch, filters],
  )

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: defaultFormValues,
  })
  const watchedRole = useWatch({
    control: form.control,
    name: "role",
  })
  const watchedStatus = useWatch({
    control: form.control,
    name: "status",
  })

  const usersQuery = useUsersQuery(queryParams, canManageUsers)
  const detailQuery = useUserDetailQuery(
    mode === "edit" ? selectedUserId : null,
    canManageUsers,
  )
  const createMutation = useCreateUserMutation()
  const updateMutation = useUpdateUserMutation()

  useEffect(() => {
    if (mode === "create") {
      form.reset(defaultFormValues)
      return
    }

    const detail = detailQuery.data?.data
    if (!detail) {
      return
    }

    form.reset({
      username: detail.username,
      name: detail.name,
      email: detail.email,
      role: detail.role,
      status: detail.isActive ? "active" : "inactive",
      password: "",
    })
  }, [detailQuery.data?.data, form, mode])

  function updateFilter<Key extends keyof UserFilters>(
    key: Key,
    value: UserFilters[Key],
  ) {
    setFilters((current) => ({
      ...current,
      [key]: value,
      page: key === "page" || key === "perPage" ? current.page : 1,
    }))
  }

  function openCreateDialog() {
    setMode("create")
    setSelectedUserId(null)
    form.reset(defaultFormValues)
    setDialogOpen(true)
  }

  function openEditDialog(userId: number) {
    setMode("edit")
    setSelectedUserId(userId)
    setDialogOpen(true)
  }

  function closeDialog(nextOpen: boolean) {
    setDialogOpen(nextOpen)
    if (!nextOpen) {
      setMode("create")
      setSelectedUserId(null)
      form.reset(defaultFormValues)
    }
  }

  const roleOptions =
    actorRole === "dev"
      ? [
          { value: "dev", label: "Dev" },
          { value: "superadmin", label: "Superadmin" },
          { value: "admin", label: "Admin" },
          { value: "user", label: "User" },
        ]
      : [
          { value: "admin", label: "Admin" },
          { value: "user", label: "User" },
        ]

  const handleSubmit = form.handleSubmit(async (values) => {
    const payload = {
      username: values.username.trim(),
      name: values.name.trim(),
      email: values.email.trim(),
      role: values.role,
      isActive: values.status === "active",
      ...(values.password?.trim() ? { password: values.password.trim() } : {}),
    }

    try {
      if (mode === "create") {
        await createMutation.mutateAsync(payload)
        toast.success("User berhasil dibuat.")
      } else if (selectedUserId != null) {
        await updateMutation.mutateAsync({
          userId: selectedUserId,
          payload,
        })
        toast.success("User berhasil diperbarui.")
      }

      closeDialog(false)
    } catch (error) {
      if (isBackofficeRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          if (field in values) {
            form.setError(field as keyof UserFormValues, { message })
          }
        }
      }

      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal menyimpan user.",
      )
    }
  })

  if (!canManageUsers) {
    return (
      <main className="grid gap-6 px-4 py-4 lg:px-6">
        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader>
            <CardTitle>Users</CardTitle>
            <CardDescription>
              Halaman ini hanya tersedia untuk `dev` dan `superadmin`, mengikuti policy legacy.
            </CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Akses server-side sudah dibatasi. Shell React menyembunyikan menu ini untuk role lain,
            tetapi direct access tetap akan berakhir di state terbatas seperti ini.
          </CardContent>
        </Card>
      </main>
    )
  }

  const records = usersQuery.data?.data ?? []
  const summary = usersQuery.data?.summary

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MiniUserStat
          title="Visible users"
          value={String(summary?.totalUsers ?? 0)}
          description="List mengikuti query legacy: default hanya role admin dan user."
          icon={UsersRoundIcon}
        />
        <MiniUserStat
          title="Active users"
          value={String(summary?.activeUsers ?? 0)}
          description="Akun operator yang masih boleh login dan mengakses backoffice."
          icon={ShieldCheckIcon}
        />
        <MiniUserStat
          title="Admin accounts"
          value={String(summary?.adminUsers ?? 0)}
          description="Role admin yang biasa dipakai untuk owner-level operasional."
          icon={ShieldCheckIcon}
        />
        <MiniUserStat
          title="End users"
          value={String(summary?.endUsers ?? 0)}
          description="Role user yang tetap aktif sebagai owner scoped operator."
          icon={UsersRoundIcon}
        />
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>User management</CardTitle>
              <CardDescription>
                Kelola operator, role, status aktif, dan password reset ringan tanpa keluar dari shell rewrite.
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                className="w-full rounded-xl sm:w-auto"
                onClick={() => usersQuery.refetch()}
              >
                <RefreshCcwIcon className="size-4" />
                Refresh
              </Button>
              <Button className="w-full rounded-xl sm:w-auto" onClick={openCreateDialog}>
                <PlusIcon className="size-4" />
                User baru
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-4">
          <div className="relative md:col-span-2">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={filters.search}
              onChange={(event) => updateFilter("search", event.target.value)}
              placeholder="Cari username, nama, email, atau role"
              className="pl-9"
            />
          </div>
          <Select
            value={filters.role || "all"}
            onValueChange={(value) =>
              updateFilter("role", value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Semua role" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua role</SelectItem>
              <SelectItem value="admin">Admin</SelectItem>
              <SelectItem value="user">User</SelectItem>
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
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Daftar operator</CardTitle>
          <CardDescription>
            Sama seperti legacy Filament, grid utama fokus ke role admin/user. Privileged role tetap dijaga lebih ketat.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="overflow-x-auto rounded-[1.25rem] border border-border/70">
            <Table className="min-w-[56rem]">
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usersQuery.isLoading
                  ? Array.from({ length: filters.perPage }).map((_, index) => (
                      <TableRow key={index}>
                        <TableCell colSpan={7}>
                          <Skeleton className="h-10 w-full rounded-xl" />
                        </TableCell>
                      </TableRow>
                    ))
                  : records.map((record) => (
                      <TableRow key={record.id}>
                        <TableCell className="font-medium">{record.username}</TableCell>
                        <TableCell>{record.name}</TableCell>
                        <TableCell>{record.email}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className="rounded-full capitalize">
                            {record.role}
                          </Badge>
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

          {!usersQuery.isLoading && records.length === 0 ? (
            <Card className="rounded-[1.25rem] border-dashed border-border/70 bg-background/40">
              <CardContent className="py-10 text-center">
                <p className="font-medium">Tidak ada user yang cocok</p>
                <p className="mt-2 text-sm text-muted-foreground">
                  Ubah filter atau buat operator baru dari dialog create.
                </p>
              </CardContent>
            </Card>
          ) : null}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={closeDialog}>
        <DialogContent
          data-testid="users-dialog"
          className="max-h-[92svh] overflow-y-auto sm:max-w-2xl"
        >
          <DialogHeader>
            <DialogTitle>{mode === "create" ? "Create user" : "Manage user"}</DialogTitle>
            <DialogDescription>
              Role yang tersedia mengikuti actor saat ini, sama seperti opsi form legacy Filament.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="user-form-username">Username</Label>
              <Input
                id="user-form-username"
                data-testid="user-form-username"
                {...form.register("username")}
                placeholder="username"
              />
              {form.formState.errors.username ? (
                <p className="text-sm text-destructive">
                  {form.formState.errors.username.message}
                </p>
              ) : null}
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-form-name">Name</Label>
              <Input
                id="user-form-name"
                data-testid="user-form-name"
                {...form.register("name")}
                placeholder="Nama lengkap"
              />
              {form.formState.errors.name ? (
                <p className="text-sm text-destructive">
                  {form.formState.errors.name.message}
                </p>
              ) : null}
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-form-email">Email</Label>
              <Input
                id="user-form-email"
                data-testid="user-form-email"
                {...form.register("email")}
                placeholder="nama@email.com"
              />
              {form.formState.errors.email ? (
                <p className="text-sm text-destructive">
                  {form.formState.errors.email.message}
                </p>
              ) : null}
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-form-role">Role</Label>
              <Select
                value={watchedRole}
                onValueChange={(value) => form.setValue("role", value)}
              >
                <SelectTrigger id="user-form-role" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {form.formState.errors.role ? (
                <p className="text-sm text-destructive">
                  {form.formState.errors.role.message}
                </p>
              ) : null}
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-form-status">Status</Label>
              <Select
                value={watchedStatus}
                onValueChange={(value) =>
                  form.setValue("status", value as UserFormValues["status"])
                }
              >
                <SelectTrigger id="user-form-status" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="inactive">Inactive</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="user-form-password">
                Password {mode === "edit" ? "(opsional)" : ""}
              </Label>
              <Input
                id="user-form-password"
                data-testid="user-form-password"
                {...form.register("password")}
                type="password"
                placeholder={mode === "edit" ? "Kosongkan jika tidak diubah" : "Minimal 8 karakter"}
              />
              {form.formState.errors.password ? (
                <p className="text-sm text-destructive">
                  {form.formState.errors.password.message}
                </p>
              ) : null}
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button
              data-testid="user-form-submit"
              className="w-full rounded-xl sm:w-auto"
              onClick={handleSubmit}
              disabled={createMutation.isPending || updateMutation.isPending || detailQuery.isLoading}
            >
              {mode === "create" ? "Create user" : "Save changes"}
            </Button>
            <Button
              variant="outline"
              className="w-full rounded-xl sm:w-auto"
              onClick={() => closeDialog(false)}
            >
              Cancel
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </main>
  )
}

function MiniUserStat({
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
