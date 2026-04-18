import { z } from "zod"

export const loginFormSchema = z.object({
  login: z
    .string()
    .trim()
    .min(1, "Username atau email wajib diisi.")
    .max(100, "Username atau email terlalu panjang."),
  password: z.string().min(1, "Password wajib diisi."),
  remember: z.boolean(),
})

export type LoginFormValues = z.infer<typeof loginFormSchema>

export const loginMfaFormSchema = z.object({
  code: z
    .string()
    .trim()
    .min(6, "Kode MFA atau recovery code wajib diisi.")
    .max(32, "Kode MFA terlalu panjang."),
})

export type LoginMfaFormValues = z.infer<typeof loginMfaFormSchema>

export const signupFormSchema = z
  .object({
    username: z
      .string()
      .trim()
      .min(5, "Username minimal 5 karakter.")
      .max(20, "Username maksimal 20 karakter.")
      .regex(/^[a-zA-Z0-9]+$/, "Username harus alfanumerik."),
    name: z
      .string()
      .trim()
      .min(5, "Nama minimal 5 karakter.")
      .max(100, "Nama maksimal 100 karakter."),
    email: z.email("Email tidak valid."),
    password: z
      .string()
      .min(8, "Password minimal 8 karakter.")
      .max(100, "Password terlalu panjang."),
    password_confirmation: z.string().min(1, "Konfirmasi password wajib diisi."),
  })
  .refine((values) => values.password === values.password_confirmation, {
    path: ["password_confirmation"],
    message: "Password confirmation does not match.",
  })

export type SignupFormValues = z.infer<typeof signupFormSchema>
