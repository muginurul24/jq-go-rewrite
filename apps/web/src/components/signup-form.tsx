import { Link } from "@tanstack/react-router"
import { LoaderCircleIcon } from "lucide-react"
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
import type { SignupFormValues } from "@/features/auth/forms"
import { Input } from "@/components/ui/input"

type SignupFormProps = React.ComponentProps<typeof Card> & {
  form: UseFormReturn<SignupFormValues>
  onSubmit: ReturnType<UseFormReturn<SignupFormValues>["handleSubmit"]>
  isPending?: boolean
  formError?: string | null
}

export function SignupForm({
  form,
  onSubmit,
  isPending = false,
  formError,
  ...props
}: SignupFormProps) {
  const {
    register,
    formState: { errors },
  } = form

  return (
    <Card
      className="rounded-[1.75rem] border-border/70 bg-card/95 shadow-xl shadow-primary/5 backdrop-blur"
      {...props}
    >
      <CardHeader>
        <CardTitle>Create an account</CardTitle>
        <CardDescription>
          Daftarkan operator baru dengan field parity yang sama seperti legacy.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="space-y-5" onSubmit={onSubmit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="username">Username</FieldLabel>
              <Input
                id="username"
                autoComplete="username"
                placeholder="operator123"
                aria-invalid={errors.username ? "true" : "false"}
                {...register("username")}
              />
              <FieldError errors={[errors.username]} />
            </Field>

            <Field>
              <FieldLabel htmlFor="name">Full Name</FieldLabel>
              <Input
                id="name"
                autoComplete="name"
                placeholder="John Doe"
                aria-invalid={errors.name ? "true" : "false"}
                {...register("name")}
              />
              <FieldError errors={[errors.name]} />
            </Field>

            <Field>
              <FieldLabel htmlFor="email">Email</FieldLabel>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="operator@example.com"
                aria-invalid={errors.email ? "true" : "false"}
                {...register("email")}
              />
              <FieldDescription>
                Email dipakai untuk identitas operator dan alur login parity.
              </FieldDescription>
              <FieldError errors={[errors.email]} />
            </Field>

            <Field>
              <FieldLabel htmlFor="password">Password</FieldLabel>
              <Input
                id="password"
                type="password"
                autoComplete="new-password"
                aria-invalid={errors.password ? "true" : "false"}
                {...register("password")}
              />
              <FieldDescription>Minimal 8 karakter.</FieldDescription>
              <FieldError errors={[errors.password]} />
            </Field>

            <Field>
              <FieldLabel htmlFor="password_confirmation">
                Confirm Password
              </FieldLabel>
              <Input
                id="password_confirmation"
                type="password"
                autoComplete="new-password"
                aria-invalid={errors.password_confirmation ? "true" : "false"}
                {...register("password_confirmation")}
              />
              <FieldError errors={[errors.password_confirmation]} />
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
                    Membuat akun
                  </>
                ) : (
                  "Create Account"
                )}
              </Button>
              <FieldDescription className="px-2 text-center">
                Sudah punya akun?{" "}
                <Link to="/login" className="font-medium text-primary">
                  Masuk di sini
                </Link>
              </FieldDescription>
            </Field>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  )
}
