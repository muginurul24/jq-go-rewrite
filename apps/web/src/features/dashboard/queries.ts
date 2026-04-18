import { useQuery } from "@tanstack/react-query"

import {
  getDashboardOperationalPulse,
  getDashboardOverview,
} from "@/features/dashboard/api"

export function useDashboardOverviewQuery() {
  return useQuery({
    queryKey: ["backoffice", "dashboard", "overview"],
    queryFn: getDashboardOverview,
    refetchInterval: 30_000,
  })
}

export function useDashboardOperationalPulseQuery() {
  return useQuery({
    queryKey: ["backoffice", "dashboard", "operational-pulse"],
    queryFn: getDashboardOperationalPulse,
    refetchInterval: 30_000,
  })
}
