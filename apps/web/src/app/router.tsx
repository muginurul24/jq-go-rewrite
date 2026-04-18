/* eslint-disable react-refresh/only-export-components */
import type { ReactNode } from "react"

import type { QueryClient } from "@tanstack/react-query"
import {
  Link,
  Outlet,
  RouterProvider,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  redirect,
} from "@tanstack/react-router"
import { HomeIcon, LogInIcon } from "lucide-react"

import { BackofficeLayout } from "@/components/backoffice-layout"
import { Button } from "@/components/ui/button"
import { authBootstrapQueryOptions, useAuthBootstrap } from "@/features/auth/queries"
import { setBackofficeCSRFToken } from "@/lib/backoffice-api"
import { DashboardPage } from "@/pages/dashboard-page"
import { GamesPage } from "@/pages/games-page"
import { LoginPage } from "@/pages/login-page"
import { OperationalPulsePage } from "@/pages/operational-pulse-page"
import { ApiDocsPage } from "@/pages/api-docs-page"
import { BanksPage } from "@/pages/banks-page"
import { CallManagementPage } from "@/pages/call-management-page"
import { NexusggrTopupPage } from "@/pages/nexusggr-topup-page"
import { PlayersPage } from "@/pages/players-page"
import { ProfilePage } from "@/pages/profile-page"
import { ProvidersPage } from "@/pages/providers-page"
import { RegisterPage } from "@/pages/register-page"
import { TokosPage } from "@/pages/tokos-page"
import { TransactionsPage } from "@/pages/transactions-page"
import { UsersPage } from "@/pages/users-page"
import { WithdrawalPage } from "@/pages/withdrawal-page"

type RouterContext = {
  queryClient: QueryClient
}

type BackofficePageFrameProps = {
  children: ReactNode
}

function RootComponent() {
  return <Outlet />
}

function NotFoundPage() {
  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-6 py-10">
      <div className="w-full max-w-lg space-y-4 rounded-[2rem] border border-border/70 bg-card p-8 text-center shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.28em] text-muted-foreground">
          404
        </p>
        <h1 className="text-3xl font-semibold tracking-tight">
          Halaman tidak ditemukan
        </h1>
        <p className="text-sm leading-6 text-muted-foreground">
          Route yang Anda buka tidak ada di shell rewrite saat ini. Kembali ke
          dashboard atau login untuk melanjutkan.
        </p>
        <div className="flex flex-col gap-3 sm:flex-row sm:justify-center">
          <Button asChild className="rounded-xl">
            <Link to="/backoffice">
              <HomeIcon className="size-4" />
              Dashboard
            </Link>
          </Button>
          <Button asChild variant="outline" className="rounded-xl">
            <Link to="/login">
              <LogInIcon className="size-4" />
              Login
            </Link>
          </Button>
        </div>
      </div>
    </main>
  )
}

async function requireAuth({
  context,
}: {
  context: RouterContext
}) {
  const authState = await context.queryClient.ensureQueryData(
    authBootstrapQueryOptions(),
  )

  setBackofficeCSRFToken(authState.csrfToken ?? "")

  if (!authState.user) {
    throw redirect({ to: "/login" })
  }

  return authState
}

async function requireGuest({
  context,
}: {
  context: RouterContext
}) {
  const authState = await context.queryClient.ensureQueryData(
    authBootstrapQueryOptions(),
  )

  setBackofficeCSRFToken(authState.csrfToken ?? "")

  if (authState.user) {
    throw redirect({ to: "/backoffice" })
  }

  return authState
}

function BackofficePageFrame({ children }: BackofficePageFrameProps) {
  const authBootstrap = useAuthBootstrap()

  if (!authBootstrap.data?.user) {
    return null
  }

  return (
    <BackofficeLayout user={authBootstrap.data.user}>{children}</BackofficeLayout>
  )
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
  notFoundComponent: NotFoundPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: () => {
    throw redirect({ to: "/backoffice" })
  },
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  beforeLoad: requireGuest,
  component: LoginPage,
})

const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/register",
  beforeLoad: requireGuest,
  component: RegisterPage,
})

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <DashboardPage />
    </BackofficePageFrame>
  ),
})

const usersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/users",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <UsersPage />
    </BackofficePageFrame>
  ),
})

const operationalPulseRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/operational-pulse",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <OperationalPulsePage />
    </BackofficePageFrame>
  ),
})

const tokosRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/tokos",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <TokosPage />
    </BackofficePageFrame>
  ),
})

const banksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/banks",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <BanksPage />
    </BackofficePageFrame>
  ),
})

const playersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/players",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <PlayersPage />
    </BackofficePageFrame>
  ),
})

const transactionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/transactions",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <TransactionsPage />
    </BackofficePageFrame>
  ),
})

const providersRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/providers",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <ProvidersPage />
    </BackofficePageFrame>
  ),
})

const gamesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/games",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <GamesPage />
    </BackofficePageFrame>
  ),
})

const callManagementRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/call-management",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <CallManagementPage />
    </BackofficePageFrame>
  ),
})

const nexusggrTopupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/nexusggr-topup",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <NexusggrTopupPage />
    </BackofficePageFrame>
  ),
})

const withdrawalRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/withdrawal",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <WithdrawalPage />
    </BackofficePageFrame>
  ),
})

const apiDocsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/api-docs",
  beforeLoad: requireAuth,
  component: () => (
    <BackofficePageFrame>
      <ApiDocsPage />
    </BackofficePageFrame>
  ),
})

function ProfileRoutePage() {
  const authBootstrap = useAuthBootstrap()

  if (!authBootstrap.data?.user) {
    return null
  }

  return (
    <BackofficePageFrame>
      <ProfilePage user={authBootstrap.data.user} />
    </BackofficePageFrame>
  )
}

const profileRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/backoffice/profile",
  beforeLoad: requireAuth,
  component: ProfileRoutePage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  registerRoute,
  dashboardRoute,
  operationalPulseRoute,
  usersRoute,
  tokosRoute,
  banksRoute,
  playersRoute,
  transactionsRoute,
  providersRoute,
  gamesRoute,
  callManagementRoute,
  nexusggrTopupRoute,
  withdrawalRoute,
  apiDocsRoute,
  profileRoute,
])

export const router = createRouter({
  routeTree,
  context: {
    queryClient: undefined as unknown as QueryClient,
  },
  defaultPreload: "intent",
  defaultPreloadStaleTime: 30_000,
})

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

export function AppRouterProvider({ queryClient }: RouterContext) {
  return <RouterProvider router={router} context={{ queryClient }} />
}
