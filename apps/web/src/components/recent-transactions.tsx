import * as React from "react"

import { SearchIcon, SlidersHorizontalIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

type TransactionStatus = "pending" | "success" | "failed" | "expired"

type TransactionRow = {
  code: string
  toko: string
  player: string
  category: "qris" | "nexusggr"
  type: "deposit" | "withdrawal"
  amount: number
  status: TransactionStatus
  createdAt: string
}

const transactions: TransactionRow[] = [
  {
    code: "QR-20260417-001",
    toko: "JustQiu Pusat",
    player: "adi88",
    category: "qris",
    type: "deposit",
    amount: 350_000,
    status: "pending",
    createdAt: "2026-04-17 09:12 WIB",
  },
  {
    code: "NX-20260417-002",
    toko: "JustQiu Timur",
    player: "raka303",
    category: "nexusggr",
    type: "withdrawal",
    amount: 120_000,
    status: "success",
    createdAt: "2026-04-17 08:41 WIB",
  },
  {
    code: "QR-20260416-113",
    toko: "JustQiu Barat",
    player: "putri11",
    category: "qris",
    type: "withdrawal",
    amount: 875_000,
    status: "failed",
    createdAt: "2026-04-16 19:33 WIB",
  },
  {
    code: "NX-20260416-071",
    toko: "JustQiu Selatan",
    player: "ekoalpha",
    category: "nexusggr",
    type: "deposit",
    amount: 540_000,
    status: "success",
    createdAt: "2026-04-16 14:07 WIB",
  },
  {
    code: "QR-20260416-049",
    toko: "JustQiu Pusat",
    player: "nova777",
    category: "qris",
    type: "deposit",
    amount: 100_000,
    status: "expired",
    createdAt: "2026-04-16 11:12 WIB",
  },
]

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

const statusVariant: Record<TransactionStatus, "secondary" | "default" | "destructive" | "outline"> =
  {
    pending: "secondary",
    success: "default",
    failed: "destructive",
    expired: "outline",
  }

export function RecentTransactions() {
  const [query, setQuery] = React.useState("")
  const [status, setStatus] = React.useState<string>("all")

  const rows = transactions.filter((transaction) => {
    const matchesQuery =
      query.length === 0 ||
      transaction.code.toLowerCase().includes(query.toLowerCase()) ||
      transaction.player.toLowerCase().includes(query.toLowerCase()) ||
      transaction.toko.toLowerCase().includes(query.toLowerCase())

    const matchesStatus =
      status === "all" ? true : transaction.status === status

    return matchesQuery && matchesStatus
  })

  return (
    <Card className="border-border/60 bg-card/80 shadow-sm backdrop-blur">
      <CardHeader className="gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <CardTitle>Recent transactions</CardTitle>
          <CardDescription>
            Foundation untuk tabel transaksi parity: search, filter, audit, dan
            status visibility.
          </CardDescription>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="relative">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Cari kode, toko, player..."
              className="w-full min-w-60 pl-9"
            />
          </div>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="w-full min-w-40">
              <SelectValue placeholder="Semua status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua status</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="success">Success</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" className="gap-2">
            <SlidersHorizontalIcon className="size-4" />
            Filter
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-2xl border border-border/60">
          <Table className="min-w-[58rem]">
            <TableHeader>
              <TableRow className="bg-muted/40">
                <TableHead>Code</TableHead>
                <TableHead>Toko</TableHead>
                <TableHead>Player</TableHead>
                <TableHead>Flow</TableHead>
                <TableHead className="text-right">Amount</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((transaction) => (
                <TableRow key={transaction.code}>
                  <TableCell className="font-medium">{transaction.code}</TableCell>
                  <TableCell>{transaction.toko}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {transaction.player}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <Badge variant="outline" className="w-fit capitalize">
                        {transaction.category}
                      </Badge>
                      <span className="text-xs text-muted-foreground capitalize">
                        {transaction.type}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-semibold tabular-nums">
                    {currencyFormatter.format(transaction.amount)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[transaction.status]} className="capitalize">
                      {transaction.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {transaction.createdAt}
                  </TableCell>
                </TableRow>
              ))}
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={7}
                    className="h-28 text-center text-sm text-muted-foreground"
                  >
                    Tidak ada transaksi yang cocok dengan filter saat ini.
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
