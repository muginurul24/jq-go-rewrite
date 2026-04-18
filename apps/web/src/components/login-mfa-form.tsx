import { Link } from "@tanstack/react-router"
import { LoaderCircleIcon, ShieldCheckIcon } from "lucide-react"
import type { UseFormReturn } from "react-hook-form"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import type { LoginMfaFormValues } from "@/features/auth/forms"
import { cn } from "@/lib/utils"

type LoginMfaFormProps = React.ComponentProps<"div"> & {
  form: UseFormReturn<LoginMfaFormValues>
  onSubmit: ReturnType<UseFormReturn<LoginMfaFormValues>["handleSubmit"]>
  isPending?: boolean
  formError?: string | null
}

export function LoginMfaForm({
  className,
  form,
  onSubmit,
  isPending = false,
  formError,
  ...props
}: LoginMfaFormProps) {
  const {
    register,
    formState: { errors },
  } = form

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-xl shadow-primary/5 backdrop-blur">
        <CardHeader className="space-y-2">
          <CardTitle className="flex items-center gap-2">
            <ShieldCheckIcon className="size-5 text-primary" />
            Verify MFA
          </CardTitle>
          <CardDescription>
            Masukkan 6 digit code dari authenticator app atau salah satu recovery code.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-5" onSubmit={onSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="mfa-code">MFA code</FieldLabel>
                <Input
                  id="mfa-code"
                  autoComplete="one-time-code"
                  placeholder="123456 atau ABCD-1234"
                  aria-invalid={errors.code ? "true" : "false"}
                  {...register("code")}
                />
                <FieldDescription>
                  Recovery code akan dipakai satu kali lalu dihapus dari daftar.
                </FieldDescription>
                <FieldError errors={[errors.code]} />
              </Field>

              {formError ? (
                <div className="rounded-xl border border-destructive/25 bg-destructive/8 px-3 py-2 text-sm text-destructive">
                  {formError}
                </div>
              ) : null}

              <Field>
                <Button
                  type="submit"
                  disabled={isPending}
                  className="w-full rounded-xl"
                >
                  {isPending ? (
                    <>
                      <LoaderCircleIcon className="size-4 animate-spin" />
                      Verifying
                    </>
                  ) : (
                    "Verify MFA"
                  )}
                </Button>
                <FieldDescription className="pt-1 text-center">
                  Salah akun?{" "}
                  <Link to="/login" search={{}} className="font-medium text-primary">
                    Kembali ke login
                  </Link>
                </FieldDescription>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
