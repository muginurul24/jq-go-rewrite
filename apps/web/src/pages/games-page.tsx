import { useDeferredValue, useMemo, useState } from "react"

import { Gamepad2Icon, SearchIcon } from "lucide-react"

import { useGamesQuery, useProvidersQuery } from "@/features/catalog/queries"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"

export function GamesPage() {
  const providersQuery = useProvidersQuery()
  const [providerCode, setProviderCode] = useState("")
  const [search, setSearch] = useState("")
  const [localized, setLocalized] = useState(false)
  const deferredSearch = useDeferredValue(search)
  const activeProviderCode =
    providerCode || providersQuery.data?.providers[0]?.code || ""

  const gamesQuery = useGamesQuery(activeProviderCode, localized)
  const filteredGames = useMemo(() => {
    const items = gamesQuery.data?.games ?? []
    const term = deferredSearch.trim().toLowerCase()
    if (!term) {
      return items
    }

    return items.filter((game) => {
      return (
        String(game.game_name).toLowerCase().includes(term) ||
        String(game.game_code).toLowerCase().includes(term)
      )
    })
  }, [deferredSearch, gamesQuery.data?.games])

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <Card className="rounded-[1.75rem] border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Games</CardTitle>
          <CardDescription>
            Katalog game per provider, dengan opsi localized names dari endpoint
            v2.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <Select value={activeProviderCode} onValueChange={setProviderCode}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Pilih provider" />
            </SelectTrigger>
            <SelectContent>
              {providersQuery.data?.providers.map((provider) => (
                <SelectItem key={provider.code} value={provider.code}>
                  {provider.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="relative">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Cari nama atau kode game"
              className="pl-9"
            />
          </div>

          <Select
            value={localized ? "localized" : "standard"}
            onValueChange={(value) => setLocalized(value === "localized")}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="standard">Standard names</SelectItem>
              <SelectItem value="localized">Localized names</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {gamesQuery.isLoading
          ? Array.from({ length: 9 }).map((_, index) => (
              <Skeleton key={index} className="h-40 rounded-[1.5rem]" />
            ))
          : filteredGames.map((game) => (
              <Card
                key={`${activeProviderCode}-${game.game_code}`}
                className="overflow-hidden rounded-[1.5rem] border-border/70 bg-card/90"
              >
                {game.banner ? (
                  <img
                    src={game.banner}
                    alt={String(game.game_name)}
                    className="h-36 w-full object-cover"
                    loading="lazy"
                  />
                ) : (
                  <div className="flex h-36 items-center justify-center bg-muted/60">
                    <Gamepad2Icon className="size-8 text-muted-foreground" />
                  </div>
                )}
                <CardHeader className="space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1">
                      <CardTitle className="line-clamp-2 text-lg">
                        {String(game.game_name)}
                      </CardTitle>
                      <CardDescription>{game.game_code}</CardDescription>
                    </div>
                    {game.status != null ? (
                      <Badge variant="outline" className="rounded-full px-2.5">
                        {String(game.status)}
                      </Badge>
                    ) : null}
                  </div>
                </CardHeader>
              </Card>
            ))}
      </div>

      {!gamesQuery.isLoading && filteredGames.length === 0 ? (
        <Card className="rounded-[1.5rem] border-dashed border-border/70 bg-card/90">
          <CardContent className="py-10 text-center">
            <p className="font-medium">Game tidak ditemukan</p>
            <p className="mt-2 text-sm text-muted-foreground">
              Ganti provider, matikan localized mode, atau ubah kata kunci
              pencarian.
            </p>
          </CardContent>
        </Card>
      ) : null}
    </main>
  )
}
