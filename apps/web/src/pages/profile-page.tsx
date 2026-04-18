import { useState } from "react"

import { QRCodeSVG } from "qrcode.react"
import { ShieldCheckIcon, UserRoundCogIcon } from "lucide-react"
import { toast } from "sonner"

import { ThemeToggle } from "@/components/theme-toggle"
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
import { Separator } from "@/components/ui/separator"
import type { AuthUser } from "@/features/auth/api"
import {
  useBeginMfaSetupMutation,
  useConfirmMfaSetupMutation,
  useDisableMfaMutation,
  useMfaStatusQuery,
} from "@/features/mfa/queries"
import { isBackofficeRequestError } from "@/lib/backoffice-api"

type ProfilePageProps = {
  user: AuthUser
}

export function ProfilePage({ user }: ProfilePageProps) {
  const [confirmCode, setConfirmCode] = useState("")
  const [disableCode, setDisableCode] = useState("")
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])

  const mfaStatusQuery = useMfaStatusQuery()
  const beginMfaSetupMutation = useBeginMfaSetupMutation()
  const confirmMfaSetupMutation = useConfirmMfaSetupMutation()
  const disableMfaMutation = useDisableMfaMutation()

  const mfaStatus = mfaStatusQuery.data
  const mfaEnabled = mfaStatus?.enabled ?? user.mfaEnabled ?? false
  const pendingSetup = mfaStatus?.pendingSetup ?? false
  const setupSecret = mfaStatus?.secret ?? ""
  const setupUrl = mfaStatus?.otpauthUrl ?? ""

  async function handleBeginMfaSetup() {
    try {
      setRecoveryCodes([])
      setConfirmCode("")
      await beginMfaSetupMutation.mutateAsync()
      toast.success("QR authenticator siap di-scan.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal memulai setup MFA.",
      )
    }
  }

  async function handleConfirmMfa() {
    try {
      const response = await confirmMfaSetupMutation.mutateAsync(confirmCode)
      setRecoveryCodes(response.recoveryCodes ?? [])
      setConfirmCode("")
      toast.success("MFA berhasil diaktifkan.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal mengaktifkan MFA.",
      )
    }
  }

  async function handleDisableMfa() {
    try {
      await disableMfaMutation.mutateAsync(disableCode)
      setDisableCode("")
      setRecoveryCodes([])
      toast.success("MFA berhasil dimatikan.")
    } catch (error) {
      toast.error(
        isBackofficeRequestError(error)
          ? error.payload.message
          : "Gagal mematikan MFA.",
      )
    }
  }

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader className="space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              <Badge className="rounded-full bg-primary/10 px-3 text-primary hover:bg-primary/10">
                Session-authenticated operator
              </Badge>
              <Badge variant="outline" className="rounded-full px-3">
                Role {user.role}
              </Badge>
              <Badge variant={mfaEnabled ? "default" : "outline"} className="rounded-full px-3">
                MFA {mfaEnabled ? "Enabled" : "Disabled"}
              </Badge>
            </div>
            <div className="space-y-2">
              <CardTitle className="text-2xl">{user.name}</CardTitle>
              <CardDescription className="max-w-2xl text-sm leading-6">
                Halaman profile ini menjadi titik ringkas untuk identitas operator,
                prinsip keamanan session, preferensi visual, dan setup MFA TOTP.
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            <IdentityCard label="Username" value={user.username} />
            <IdentityCard label="Email" value={user.email} />
            <IdentityCard label="Access role" value={user.role} />
            <IdentityCard label="Session state" value={user.isActive === false ? "Inactive" : "Active"} />
          </CardContent>
        </Card>

        <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <UserRoundCogIcon className="size-5 text-primary" />
              Preferences
            </CardTitle>
            <CardDescription>
              Pengaturan visual tetap terpisah dari kontrol keamanan.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-4 rounded-[1.25rem] border border-border/70 bg-background/60 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <p className="font-medium">Theme mode</p>
                <p className="text-sm text-muted-foreground">
                  Light, dark, dan system mengikuti standar shell baru.
                </p>
              </div>
              <ThemeToggle />
            </div>

            <Separator />

            <div className="space-y-3 rounded-[1.25rem] border border-emerald-500/20 bg-emerald-500/8 p-4">
              <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300">
                <ShieldCheckIcon className="size-4" />
                <p className="font-medium">Security posture</p>
              </div>
              <ul className="space-y-2 text-sm leading-6 text-muted-foreground">
                <li>Backoffice tetap memakai session browser, bukan JWT.</li>
                <li>CSRF token diputar melalui bootstrap dan login/register response.</li>
                <li>MFA TOTP sekarang bisa diaktifkan langsung dari profile.</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </section>

      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheckIcon className="size-5 text-primary" />
              Multi-factor authentication
            </CardTitle>
            <CardDescription>
              Authenticator app memakai TOTP dan recovery code sekali pakai, tetap di atas session auth legacy.
            </CardDescription>
          </div>
          <Badge variant={mfaEnabled ? "default" : "outline"} className="rounded-full px-3">
            {mfaEnabled ? "Protected" : pendingSetup ? "Pending setup" : "Optional"}
          </Badge>
        </CardHeader>
        <CardContent className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]">
          <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="text-base">
                {mfaEnabled ? "Disable MFA" : pendingSetup ? "Confirm MFA setup" : "Enable MFA"}
              </CardTitle>
              <CardDescription>
                {mfaEnabled
                  ? "Masukkan TOTP atau recovery code untuk mematikan MFA."
                  : pendingSetup
                    ? "Scan QR lalu masukkan code 6 digit pertama untuk mengaktifkan MFA."
                    : "Mulai setup MFA untuk menambah layer keamanan pada login operator."}
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4">
              {!mfaEnabled && !pendingSetup ? (
                <Button
                  data-testid="mfa-start"
                  className="w-full rounded-xl sm:w-auto"
                  onClick={() => void handleBeginMfaSetup()}
                  disabled={beginMfaSetupMutation.isPending}
                >
                  Start MFA setup
                </Button>
              ) : null}

              {!mfaEnabled && pendingSetup ? (
                <>
                  <div className="space-y-2">
                    <Label>Authenticator code</Label>
                    <Input
                      data-testid="mfa-confirm-code"
                      value={confirmCode}
                      onChange={(event) => setConfirmCode(event.target.value)}
                      placeholder="123456"
                      autoComplete="one-time-code"
                    />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      data-testid="mfa-confirm"
                      className="w-full rounded-xl sm:w-auto"
                      onClick={() => void handleConfirmMfa()}
                      disabled={confirmMfaSetupMutation.isPending || confirmCode.trim() === ""}
                    >
                      Confirm MFA
                    </Button>
                    <Button
                      variant="outline"
                      className="w-full rounded-xl sm:w-auto"
                      onClick={() => void mfaStatusQuery.refetch()}
                    >
                      Refresh
                    </Button>
                  </div>
                </>
              ) : null}

              {mfaEnabled ? (
                <>
                  <div className="space-y-2">
                    <Label>Disable with code</Label>
                    <Input
                      data-testid="mfa-disable-code"
                      value={disableCode}
                      onChange={(event) => setDisableCode(event.target.value)}
                      placeholder="123456 atau recovery code"
                    />
                  </div>
                  <Button
                    data-testid="mfa-disable"
                    variant="destructive"
                    className="w-full rounded-xl sm:w-auto"
                    onClick={() => void handleDisableMfa()}
                    disabled={disableMfaMutation.isPending || disableCode.trim() === ""}
                  >
                    Disable MFA
                  </Button>
                </>
              ) : null}
            </CardContent>
          </Card>

          <div className="grid gap-4">
            {pendingSetup && setupUrl ? (
              <Card className="rounded-[1.5rem] border-border/70 bg-background/50">
                <CardHeader>
                  <CardTitle className="text-base">Scan QR code</CardTitle>
                  <CardDescription>
                    QR ini aman untuk authenticator app dan akan tetap tersedia selama setup masih pending.
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                  <div className="overflow-x-auto rounded-[1.25rem] border border-border/70 bg-white p-5">
                    <QRCodeSVG value={setupUrl} size={220} includeMargin />
                  </div>
                  <div className="rounded-[1.25rem] border border-border/70 bg-background p-4">
                    <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">
                      Manual secret
                    </p>
                    <p data-testid="mfa-manual-secret" className="mt-2 break-all font-mono text-xs">
                      {setupSecret}
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {recoveryCodes.length > 0 ? (
              <Card className="rounded-[1.5rem] border-amber-500/25 bg-amber-500/5">
                <CardHeader>
                  <CardTitle className="text-base">Recovery codes</CardTitle>
                  <CardDescription>
                    Simpan sekarang. Daftar ini hanya ditampilkan sekali setelah MFA aktif.
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-2 sm:grid-cols-2">
                  {recoveryCodes.map((code) => (
                    <div
                      key={code}
                      data-testid="mfa-recovery-code"
                      className="rounded-xl border border-amber-500/20 bg-background px-3 py-3 font-mono text-sm"
                    >
                      {code}
                    </div>
                  ))}
                </CardContent>
              </Card>
            ) : null}
          </div>
        </CardContent>
      </Card>
    </main>
  )
}

function IdentityCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1.25rem] border border-border/70 bg-background/60 p-4">
      <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">
        {label}
      </p>
      <p className="mt-2 break-words font-semibold capitalize">{value}</p>
    </div>
  )
}
