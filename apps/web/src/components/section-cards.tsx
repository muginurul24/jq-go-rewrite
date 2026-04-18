import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardAction,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ArrowRightLeftIcon,
  BadgeDollarSignIcon,
  ShieldCheckIcon,
  TrendingUpIcon,
  WalletCardsIcon,
} from "lucide-react"

const currencyFormatter = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
})

const cards = [
  {
    title: "Pending Balance",
    value: 125_850_000,
    badge: "+8.2%",
    detail: "Masuk dari QRIS yang belum settle penuh.",
    icon: ArrowRightLeftIcon,
    tone: "from-emerald-500/18",
  },
  {
    title: "Settle Balance",
    value: 88_400_000,
    badge: "16:00 WIB",
    detail: "Akan dipakai untuk withdrawal dan pencairan.",
    icon: ShieldCheckIcon,
    tone: "from-sky-500/18",
  },
  {
    title: "NexusGGR Balance",
    value: 64_250_000,
    badge: "Live",
    detail: "Pool operasional untuk deposit dan withdrawal player.",
    icon: WalletCardsIcon,
    tone: "from-cyan-500/18",
  },
  {
    title: "Platform Income",
    value: 12_460_000,
    badge: "+3.4%",
    detail: "Fee transaksi dan disbursement sesuai formula legacy.",
    icon: BadgeDollarSignIcon,
    tone: "from-amber-500/18",
  },
] as const

export function SectionCards() {
  return (
    <div className="grid grid-cols-1 gap-4 px-4 lg:px-6 @xl/main:grid-cols-2 @5xl/main:grid-cols-4">
      {cards.map((card) => (
        <Card
          key={card.title}
          className={`@container/card border-border/60 bg-gradient-to-b ${card.tone} to-card shadow-sm backdrop-blur`}
        >
          <CardHeader>
            <div className="mb-5 flex size-11 items-center justify-center rounded-2xl bg-background/70 ring-1 ring-border/50">
              <card.icon className="size-5 text-primary" />
            </div>
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm text-muted-foreground">{card.title}</p>
                <CardTitle className="mt-2 text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
                  {currencyFormatter.format(card.value)}
                </CardTitle>
              </div>
              <CardAction>
                <Badge variant="outline">
                  <TrendingUpIcon className="size-3.5" />
                  {card.badge}
                </Badge>
              </CardAction>
            </div>
          </CardHeader>
          <CardFooter className="flex-col items-start gap-1.5 text-sm">
            <div className="line-clamp-1 flex gap-2 font-medium">
              Parity-sensitive metric
              <TrendingUpIcon className="size-4" />
            </div>
            <div className="text-muted-foreground">{card.detail}</div>
          </CardFooter>
        </Card>
      ))}
    </div>
  )
}
