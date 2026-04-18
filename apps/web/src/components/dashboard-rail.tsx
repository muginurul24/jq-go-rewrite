import {
  ActivityIcon,
  ArrowUpRightIcon,
  BotMessageSquareIcon,
  Clock3Icon,
  DatabaseZapIcon,
  ShieldCheckIcon,
  ZapIcon,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"

const milestones = [
  {
    title: "Session + CSRF parity",
    detail: "Backoffice tetap cookie-session, bukan JWT.",
  },
  {
    title: "Redis-first runtime",
    detail: "Cache, session, queue, dan rate-limit tetap searah legacy.",
  },
  {
    title: "Finance guardrails",
    detail: "Pending, settle, dan nexusggr tidak boleh bercampur.",
  },
]

const systemSignals = [
  { label: "HTTP Runtime", value: "Ready", icon: ActivityIcon },
  { label: "Redis Session", value: "Planned", icon: DatabaseZapIcon },
  { label: "Scheduler", value: "Weekday 16:00", icon: Clock3Icon },
  { label: "Audit Trail", value: "Structured", icon: ShieldCheckIcon },
]

export function DashboardRail() {
  return (
    <div className="grid gap-4">
      <Card className="border-border/60 bg-card/80 shadow-sm backdrop-blur">
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Quick actions</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-2">
          <Button className="justify-between rounded-xl">
            Open withdrawal wizard
            <ArrowUpRightIcon className="size-4" />
          </Button>
          <Button variant="outline" className="justify-between rounded-xl">
            Generate QRIS topup
            <ZapIcon className="size-4" />
          </Button>
          <Button variant="outline" className="justify-between rounded-xl">
            Review callback logs
            <BotMessageSquareIcon className="size-4" />
          </Button>
        </CardContent>
      </Card>

      <Card className="border-border/60 bg-card/80 shadow-sm backdrop-blur">
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Phase 1 milestones</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {milestones.map((milestone, index) => (
            <div key={milestone.title}>
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-medium">{milestone.title}</p>
                <span className="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-[0.18em] text-emerald-600 dark:text-emerald-300">
                  Active
                </span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {milestone.detail}
              </p>
              {index < milestones.length - 1 ? (
                <Separator className="mt-4" />
              ) : null}
            </div>
          ))}
        </CardContent>
      </Card>

      <Card className="border-border/60 bg-card/80 shadow-sm backdrop-blur">
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Runtime signals</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3">
          {systemSignals.map((signal) => (
            <div
              key={signal.label}
              className="flex items-center gap-3 rounded-2xl border border-border/60 bg-background/70 px-3 py-3"
            >
              <div className="flex size-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <signal.icon className="size-4" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium">{signal.label}</p>
                <p className="text-xs text-muted-foreground">{signal.value}</p>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
