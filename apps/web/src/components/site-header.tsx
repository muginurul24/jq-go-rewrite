import { ShieldCheckIcon } from "lucide-react"

import { NotificationCenter } from "@/components/notification-center"
import { ThemeToggle } from "@/components/theme-toggle"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"

type SiteHeaderProps = {
  title: string
  subtitle: string
  phaseLabel?: string
}

export function SiteHeader({
  title,
  subtitle,
  phaseLabel = "Ready",
}: SiteHeaderProps) {
  return (
    <header className="flex min-h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:min-h-(--header-height)">
      <div className="flex w-full flex-wrap items-start gap-3 px-4 py-3 lg:flex-nowrap lg:items-center lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <div className="grid min-w-0 flex-1 gap-0.5">
          <h1 className="truncate text-base font-medium">{title}</h1>
          <p className="line-clamp-2 text-xs text-muted-foreground sm:line-clamp-1">
            {subtitle}
          </p>
        </div>
        <div className="ml-auto flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto lg:flex-nowrap">
          <Badge variant="outline" className="hidden md:inline-flex">
            {phaseLabel}
          </Badge>
          <Button
            variant="outline"
            size="sm"
            className="hidden md:inline-flex"
            disabled
          >
            <ShieldCheckIcon className="size-4" />
            Session protected
          </Button>
          <NotificationCenter />
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}
