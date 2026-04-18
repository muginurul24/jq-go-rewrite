import type { CSSProperties, ReactNode } from "react"

import { useRouterState } from "@tanstack/react-router"

import { backofficeRouteMeta } from "@/app/backoffice-nav"
import { AppSidebar } from "@/components/app-sidebar"
import { SiteHeader } from "@/components/site-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import type { AuthUser } from "@/features/auth/api"

const sidebarStyle = {
  "--sidebar-width": "19rem",
  "--header-height": "4rem",
} as CSSProperties

const defaultRouteMeta = backofficeRouteMeta["/backoffice"]!

type BackofficeLayoutProps = {
  user: AuthUser
  children: ReactNode
}

export function BackofficeLayout({
  user,
  children,
}: BackofficeLayoutProps) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  const meta = backofficeRouteMeta[pathname] ?? defaultRouteMeta

  return (
    <SidebarProvider style={sidebarStyle}>
      <AppSidebar user={user} />
      <SidebarInset className="bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.06),transparent_30%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.08),transparent_28%)]">
        <SiteHeader
          title={meta.title}
          subtitle={meta.description}
          phaseLabel="Phase 2"
        />
        {children}
      </SidebarInset>
    </SidebarProvider>
  )
}
