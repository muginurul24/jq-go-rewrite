import type { ReactNode } from "react"

import { Link } from "@tanstack/react-router"
import {
  ArrowRightIcon,
  LockKeyholeIcon,
  MoonStarIcon,
  ShieldCheckIcon,
} from "lucide-react"

import { MatrixBackdrop } from "@/components/matrix-backdrop"
import { ThemeToggle } from "@/components/theme-toggle"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"

type AuthShellProps = {
  eyebrow: string
  title: string
  description: string
  highlights: string[]
  children: ReactNode
}

export function AuthShell({
  eyebrow,
  title,
  description,
  highlights,
  children,
}: AuthShellProps) {
  return (
    <main className="relative min-h-svh overflow-hidden bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.14),transparent_24%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.12),transparent_28%),linear-gradient(180deg,rgba(8,47,73,0.04),transparent_28%)]">
      <div className="absolute inset-0 opacity-65">
        <MatrixBackdrop />
      </div>

      <div className="relative z-10 grid min-h-svh xl:grid-cols-[minmax(0,1.05fr)_minmax(420px,0.95fr)]">
        <section className="flex min-w-0 flex-col justify-between px-6 py-6 lg:px-10 lg:py-10">
          <div className="flex items-center justify-between gap-4">
            <Link
              to="/login"
              className="inline-flex items-center gap-3 rounded-full border border-border/70 bg-background/70 px-3 py-2 text-sm font-medium shadow-sm backdrop-blur"
            >
              <span className="flex size-9 items-center justify-center rounded-full bg-primary/12 text-primary">
                <ShieldCheckIcon className="size-4" />
              </span>
              <span className="grid text-left leading-tight">
                <span>JustQiu Control</span>
                <span className="text-xs font-normal text-muted-foreground">
                  React + Go parity rewrite
                </span>
              </span>
            </Link>

            <div className="flex items-center gap-2">
              <Badge variant="outline" className="hidden rounded-full px-3 md:inline-flex">
                Session + CSRF
              </Badge>
              <ThemeToggle />
            </div>
          </div>

          <div className="grid gap-8 py-10 lg:max-w-2xl lg:py-16">
            <div className="space-y-5">
              <div className="flex flex-wrap items-center gap-2">
                <Badge className="rounded-full bg-primary/10 px-3 text-primary hover:bg-primary/10">
                  {eyebrow}
                </Badge>
                <Badge variant="outline" className="rounded-full px-3">
                  Legacy parity-first
                </Badge>
              </div>
              <div className="space-y-3">
                <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-balance lg:text-6xl">
                  {title}
                </h1>
                <p className="max-w-xl text-sm leading-7 text-muted-foreground lg:text-base">
                  {description}
                </p>
              </div>
            </div>

            <div className="grid gap-3 md:grid-cols-2">
              {highlights.map((highlight, index) => (
                <Card
                  key={highlight}
                  className="rounded-[1.5rem] border-border/70 bg-background/70 backdrop-blur"
                >
                  <CardContent className="flex items-start gap-4 p-5">
                    <span className="inline-flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      {index === 0 ? (
                        <LockKeyholeIcon className="size-4" />
                      ) : (
                        <MoonStarIcon className="size-4" />
                      )}
                    </span>
                    <p className="text-sm leading-6 text-muted-foreground">
                      {highlight}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>

            <Card className="max-w-xl rounded-[1.75rem] border-border/70 bg-card/90 shadow-sm backdrop-blur">
              <CardContent className="flex flex-col gap-4 p-6">
                <div className="space-y-2">
                  <p className="text-sm font-semibold">Operational rewrite target</p>
                  <p className="text-sm leading-6 text-muted-foreground">
                    Shell backoffice baru tetap memakai session browser,
                    redis-backed auth, dan kontrak bisnis legacy yang sama,
                    tetapi dengan runtime React + Go yang lebih cepat dan lebih
                    terukur.
                  </p>
                </div>
                <Button asChild variant="outline" className="w-full rounded-xl">
                  <Link to="/backoffice/api-docs">
                    Lihat ruang dokumentasi parity
                    <ArrowRightIcon className="size-4" />
                  </Link>
                </Button>
              </CardContent>
            </Card>
          </div>
        </section>

        <section className="flex items-center justify-center px-6 py-8 lg:px-10 lg:py-10">
          <div className="w-full max-w-md">{children}</div>
        </section>
      </div>
    </main>
  )
}
