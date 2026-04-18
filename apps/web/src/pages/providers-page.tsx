import { Building2Icon, RefreshCcwIcon } from "lucide-react"

import { useProvidersQuery } from "@/features/catalog/queries"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

export function ProvidersPage() {
  const providersQuery = useProvidersQuery()
  const providers = providersQuery.data?.providers ?? []

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>Providers</CardTitle>
            <CardDescription>
              Katalog provider sanitized dari NexusGGR melalui cache backend
              rewrite.
            </CardDescription>
          </div>
          <Button
            variant="outline"
            className="rounded-xl"
            onClick={() => providersQuery.refetch()}
          >
            <RefreshCcwIcon className="size-4" />
            Refresh providers
          </Button>
        </CardHeader>
      </Card>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {providersQuery.isLoading
          ? Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-36 rounded-[1.5rem]" />
            ))
          : providers.map((provider) => (
              <Card
                key={provider.code}
                className="rounded-[1.5rem] border-border/70 bg-card/90"
              >
                <CardHeader className="space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <span className="inline-flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                      <Building2Icon className="size-5" />
                    </span>
                    <Badge variant="outline" className="rounded-full px-2.5">
                      {String(provider.status)}
                    </Badge>
                  </div>
                  <div className="space-y-1">
                    <CardTitle className="text-lg">{provider.name}</CardTitle>
                    <CardDescription>Code: {provider.code}</CardDescription>
                  </div>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">
                  Provider ini sudah melalui whitelist field legacy dan siap
                  dipakai untuk katalog game maupun launch flow.
                </CardContent>
              </Card>
            ))}
      </div>

      {!providersQuery.isLoading && providers.length === 0 ? (
        <Card className="rounded-[1.5rem] border-dashed border-border/70 bg-card/90">
          <CardContent className="py-10 text-center">
            <p className="font-medium">Belum ada provider</p>
            <p className="mt-2 text-sm text-muted-foreground">
              Coba refresh ulang atau periksa konfigurasi NexusGGR.
            </p>
          </CardContent>
        </Card>
      ) : null}
    </main>
  )
}
