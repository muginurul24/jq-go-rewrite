import { zodResolver } from "@hookform/resolvers/zod"
import { useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { toast } from "sonner"

import { AuthShell } from "@/components/auth-shell"
import { SignupForm } from "@/components/signup-form"
import { isAuthRequestError } from "@/features/auth/api"
import { signupFormSchema, type SignupFormValues } from "@/features/auth/forms"
import { useRegisterMutation } from "@/features/auth/queries"

export function RegisterPage() {
  const navigate = useNavigate()
  const registerMutation = useRegisterMutation()
  const form = useForm<SignupFormValues>({
    resolver: zodResolver(signupFormSchema),
    defaultValues: {
      username: "",
      name: "",
      email: "",
      password: "",
      password_confirmation: "",
    },
  })

  const handleSubmit = form.handleSubmit(async (values) => {
    try {
      await registerMutation.mutateAsync(values)
      toast.success("Akun backoffice berhasil dibuat.")
      await navigate({ to: "/backoffice" })
    } catch (error) {
      if (isAuthRequestError(error) && error.payload.errors) {
        for (const [field, message] of Object.entries(error.payload.errors)) {
          if (field in values) {
            form.setError(field as keyof SignupFormValues, { message })
          }
        }
      }

      toast.error(
        isAuthRequestError(error)
          ? error.payload.message
          : "Registrasi gagal. Coba lagi.",
      )
    }
  })

  return (
    <AuthShell
      eyebrow="Operator onboarding"
      title="Daftarkan operator baru tanpa memutus pola akses legacy."
      description="Register tetap menjaga field parity lama: username, name, email, password, dan confirmation. Setelah berhasil, session langsung aktif supaya alur onboarding tidak terputus."
      highlights={[
        "Validasi form ditulis dengan Zod dan React Hook Form agar rule frontend tetap selaras dengan backend Go.",
        "Akses setelah register langsung diarahkan ke backoffice, sama seperti ekspektasi panel admin modern yang cepat dan praktis.",
      ]}
    >
      <SignupForm
        form={form}
        isPending={registerMutation.isPending}
        formError={
          isAuthRequestError(registerMutation.error)
            ? registerMutation.error.payload.message
            : null
        }
        onSubmit={handleSubmit}
      />
    </AuthShell>
  )
}
