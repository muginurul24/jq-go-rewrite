import { Link } from "@tanstack/react-router"
import { LoaderCircleIcon } from "lucide-react"
import type { UseFormReturn } from "react-hook-form"
import { Controller } from "react-hook-form"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import type { LoginFormValues } from "@/features/auth/forms"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

type LoginFormProps = React.ComponentProps<"div"> & {
  form: UseFormReturn<LoginFormValues>
  onSubmit: ReturnType<UseFormReturn<LoginFormValues>["handleSubmit"]>
  isPending?: boolean
  formError?: string | null
}

export function LoginForm({
  className,
  form,
  onSubmit,
  isPending = false,
  formError,
  ...props
}: LoginFormProps) {
  const {
    register,
    control,
    formState: { errors },
  } = form

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-xl shadow-primary/5 backdrop-blur">
        <CardHeader className="space-y-2">
          <CardTitle>Login to your account</CardTitle>
          <CardDescription>
            Gunakan username atau email operator untuk masuk ke backoffice.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-5" onSubmit={onSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="login">Username atau Email</FieldLabel>
                <Input
                  id="login"
                  autoComplete="username"
                  placeholder="dev@justqiu.local"
                  aria-invalid={errors.login ? "true" : "false"}
                  {...register("login")}
                />
                <FieldError errors={[errors.login]} />
              </Field>

              <Field>
                <div className="flex items-center">
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                </div>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  aria-invalid={errors.password ? "true" : "false"}
                  {...register("password")}
                />
                <FieldError errors={[errors.password]} />
              </Field>

              <Controller
                name="remember"
                control={control}
                render={({ field }) => (
                  <Field orientation="horizontal">
                    <Checkbox
                      id="remember"
                      checked={field.value}
                      onCheckedChange={(checked) => field.onChange(Boolean(checked))}
                    />
                    <FieldContent>
                      <FieldLabel htmlFor="remember">Remember this session</FieldLabel>
                      <FieldDescription>
                        Gunakan hanya pada perangkat operator yang aman.
                      </FieldDescription>
                    </FieldContent>
                  </Field>
                )}
              />

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
                      Menyiapkan session
                    </>
                  ) : (
                    "Login"
                  )}
                </Button>
                <FieldDescription className="pt-1 text-center">
                  Belum punya akun?{" "}
                  <Link to="/register" className="font-medium text-primary">
                    Buat operator baru
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
