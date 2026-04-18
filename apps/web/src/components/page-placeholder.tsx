import { Link } from "@tanstack/react-router"
import { ArrowRightIcon, SparklesIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

type PagePlaceholderProps = {
  title: string
  description: string
  nextSteps: string[]
}

export function PagePlaceholder({
  title,
  description,
  nextSteps,
}: PagePlaceholderProps) {
  return (
    <div className="grid gap-6 px-4 py-4 lg:px-6">
      <Card className="overflow-hidden rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <Badge className="rounded-full bg-primary/10 px-3 text-primary hover:bg-primary/10">
              Backoffice Route Ready
            </Badge>
            <Badge variant="outline" className="rounded-full px-3">
              Shadcn-first shell
            </Badge>
          </div>
          <div className="space-y-2">
            <CardTitle className="text-2xl">{title}</CardTitle>
            <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
              {description}
            </p>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
          <Card className="rounded-[1.25rem] border-dashed border-border/70 bg-background/50">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <SparklesIcon className="size-4 text-primary" />
                Production Checklist
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              {nextSteps.map((step, index) => (
                <div
                  key={step}
                  className="flex items-start gap-3 rounded-xl border border-border/60 px-3 py-3"
                >
                  <span className="mt-0.5 inline-flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                    {index + 1}
                  </span>
                  <p>{step}</p>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card className="rounded-[1.25rem] border-border/70 bg-linear-to-b from-primary/8 to-transparent">
            <CardHeader>
              <CardTitle className="text-base">Route Navigation</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm leading-6 text-muted-foreground">
                Shell, auth guard, theme, session cookie, dan sidebar route
                sudah aktif. Bagian berikutnya tinggal mengikat data resource dan
                aksi operasional ke endpoint internal/backoffice yang sesuai.
              </p>
              <Button asChild className="w-full rounded-xl">
                <Link to="/backoffice">
                  Kembali ke Dashboard
                  <ArrowRightIcon className="size-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        </CardContent>
      </Card>
    </div>
  )
}
