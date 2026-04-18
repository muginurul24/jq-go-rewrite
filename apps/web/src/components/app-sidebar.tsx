import * as React from "react"

import { Link, useRouterState } from "@tanstack/react-router"
import { LayoutDashboardIcon } from "lucide-react"

import { backofficeNavSections } from "@/app/backoffice-nav"
import { NavUser } from "@/components/nav-user"
import { SearchForm } from "@/components/search-form"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useLogoutMutation } from "@/features/auth/queries"

type AppSidebarProps = React.ComponentProps<typeof Sidebar> & {
  user: {
    name: string
    email: string
    role: string
  }
}

function isRouteActive(pathname: string, route: string) {
  if (route === "/backoffice") {
    return pathname === route
  }

  return pathname === route || pathname.startsWith(`${route}/`)
}

export function AppSidebar({ user, ...props }: AppSidebarProps) {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const logoutMutation = useLogoutMutation()
  const sections = React.useMemo(() => {
    return backofficeNavSections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => {
          if (item.to === "/backoffice/users") {
            return user.role === "dev" || user.role === "superadmin"
          }

          return true
        }),
      }))
      .filter((section) => section.items.length > 0)
  }, [user.role])

  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" className="pointer-events-none">
              <div className="flex aspect-square size-9 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground">
                <LayoutDashboardIcon className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="font-semibold">JustQiu Control</span>
                <span className="text-xs text-sidebar-foreground/70">
                  React + Go rewrite
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <SearchForm />
      </SidebarHeader>
      <SidebarContent>
        {sections.map((section) => (
          <SidebarGroup key={section.title}>
            <SidebarGroupLabel>{section.title}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {section.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      asChild
                      isActive={isRouteActive(pathname, item.to)}
                    >
                      <Link to={item.to}>
                        <item.icon />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={{
            name: user.name,
            email: user.email,
            avatar: "",
            role: user.role,
          }}
          isLoggingOut={logoutMutation.isPending}
          onLogout={async () => {
            await logoutMutation.mutateAsync()
          }}
        />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
