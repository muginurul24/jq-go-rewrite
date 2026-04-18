import { zodResolver } from "@hookform/resolvers/zod"
import { useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { toast } from "sonner"

import { AuthShell } from "@/components/auth-shell"
import { LoginForm } from "@/components/login-form"
import { LoginMfaForm } from "@/components/login-mfa-form"
import { isAuthRequestError } from "@/features/auth/api"
import {
  loginFormSchema,
  loginMfaFormSchema,
  type LoginFormValues,
  type LoginMfaFormValues,
} from "@/features/auth/forms"
import { useAuthBootstrap, useLoginMutation, useVerifyLoginMfaMutation } from "@/features/auth/queries"

export function LoginPage() {
  const navigate = useNavigate()
  const authBootstrap = useAuthBootstrap()
  const loginMutation = useLoginMutation()
  const verifyLoginMfaMutation = useVerifyLoginMfaMutation()
  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginFormSchema),
    defaultValues: {
      login: "",
      password: "",
      remember: true,
    },
  })
  const mfaForm = useForm<LoginMfaFormValues>({
    resolver: zodResolver(loginMfaFormSchema),
    defaultValues: {
      code: "",
    },
  })

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      const response = await loginMutation.mutateAsync(values)
      if (response.requiresMfa) {
        toast.success("Password valid. Lanjutkan dengan MFA.")
        return
      }

      toast.success("Session backoffice aktif.")
      await navigate({ to: "/backoffice" })
    } catch (error) {
      if (isAuthRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          if (field in values) {
            form.setError(field as keyof LoginFormValues, { message })
          }
        }
      }

      toast.error(
        isAuthRequestError(error)
          ? error.payload.message
          : "Login gagal. Coba lagi.",
      )
    }
  })

  const handleMfaSubmit = mfaForm.handleSubmit(async (values) => {
    try {
      await verifyLoginMfaMutation.mutateAsync(values)
      toast.success("MFA verified. Session backoffice aktif.")
      await navigate({ to: "/backoffice" })
    } catch (error) {
      if (isAuthRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          if (field in values) {
            mfaForm.setError(field as keyof LoginMfaFormValues, { message })
          }
        }
      }

      toast.error(
        isAuthRequestError(error)
          ? error.payload.message
          : "Verifikasi MFA gagal. Coba lagi.",
      )
    }
  })

  const showMfaChallenge = authBootstrap.data?.mfaPending === true

  return (
    <AuthShell
      eyebrow="Backoffice login"
      title={
        showMfaChallenge
          ? "Verifikasi MFA untuk menyelesaikan login operator."
          : "Masuk ke control room dengan session auth yang setara legacy."
      }
      description={
        showMfaChallenge
          ? "Session login ditahan sampai code MFA valid. Flow ini menjaga browser session tetap aman tanpa mengganti model auth legacy."
          : "Login menerima username atau email, mempertahankan intent Laravel lama, tetapi sekarang duduk di atas shell React yang lebih responsif dan runtime Go yang lebih ramping."
      }
      highlights={[
        "Session browser, CSRF token, dan redis-backed auth tetap dipertahankan untuk parity serta keamanan operasional.",
        showMfaChallenge
          ? "Authenticator app dan recovery code bisa dipakai untuk menyelesaikan challenge tanpa memindahkan auth backoffice ke JWT."
          : "Dark, light, dan system theme tersedia dari awal agar operator bisa bekerja nyaman di berbagai kondisi layar.",
      ]}
    >
      {showMfaChallenge ? (
        <LoginMfaForm
          form={mfaForm}
          isPending={verifyLoginMfaMutation.isPending}
          formError={
            isAuthRequestError(verifyLoginMfaMutation.error)
              ? verifyLoginMfaMutation.error.payload.message
              : null
          }
          onSubmit={handleMfaSubmit}
        />
      ) : (
        <LoginForm
          form={form}
          isPending={loginMutation.isPending}
          formError={
            isAuthRequestError(loginMutation.error)
              ? loginMutation.error.payload.message
              : null
          }
          onSubmit={handleSubmit}
        />
      )}
    </AuthShell>
  )
}
