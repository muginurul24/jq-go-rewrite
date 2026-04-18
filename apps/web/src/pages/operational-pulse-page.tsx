import { format } from "date-fns"
import { motion } from "framer-motion"
import { ActivityIcon, AlertTriangleIcon, BarChart3Icon, CheckCircle2Icon } from "lucide-react"
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import { Skeleton } from "@/components/ui/skeleton"
import { useDashboardOperationalPulseQuery } from "@/features/dashboard/queries"

const chartConfig = {
  deposit: {
    label: "Deposit",
    color: "var(--chart-1)",
  },
  withdrawal: {
    label: "Withdrawal",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig

export function OperationalPulsePage() {
  const pulseQuery = useDashboardOperationalPulseQuery()
  const pulse = pulseQuery.data?.data

  const statCards = [
    {
      title: "Pending Transactions",
      value: String(pulse?.stats.pendingTransactions ?? 0),
      description: "Jumlah transaksi lokal yang masih pending saat ini.",
      icon: ActivityIcon,
    },
    {
      title: "Failed 7D",
      value: String(pulse?.stats.failedTransactions7d ?? 0),
      description: "Transaksi gagal selama 7 hari terakhir pada scope actor.",
      icon: AlertTriangleIcon,
    },
    {
      title: "QRIS Success 7D",
      value: String(pulse?.stats.successfulQris7d ?? 0),
      description: "Jumlah transaksi QRIS sukses 7 hari terakhir.",
      icon: CheckCircle2Icon,
    },
    {
      title: "NexusGGR Success 7D",
      value: String(pulse?.stats.successfulNexusggr7d ?? 0),
      description: "Jumlah transaksi NexusGGR sukses 7 hari terakhir.",
      icon: BarChart3Icon,
    },
  ]

  return (
    <main className="grid gap-6 px-4 py-4 lg:px-6">
      <motion.section
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, ease: "easeOut" }}
      >
        <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
          <CardHeader className="gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Operational pulse</CardTitle>
              <CardDescription>
                QRIS dan NexusGGR transaction chart 7 hari yang sekarang diambil
                dari transaksi sukses nyata, sama arah dengan widget legacy.
              </CardDescription>
            </div>
            <div className="rounded-2xl border border-border/60 bg-background/70 px-4 py-3 text-sm text-muted-foreground">
              Last refresh{" "}
              <span className="font-semibold text-foreground">
                {pulse?.generatedAt
                  ? format(new Date(pulse.generatedAt), "dd MMM yyyy HH:mm:ss")
                  : "Loading..."}
              </span>
            </div>
          </CardHeader>
        </Card>
      </motion.section>

      <motion.section
        className="grid gap-4 lg:grid-cols-2 xl:grid-cols-4"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, delay: 0.08, ease: "easeOut" }}
      >
        {statCards.map((card) => (
          <Card key={card.title} className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
            <CardHeader className="space-y-4">
              <div className="flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <card.icon className="size-5" />
              </div>
              <div>
                <CardDescription>{card.title}</CardDescription>
                {pulseQuery.isLoading ? (
                  <Skeleton className="mt-2 h-9 w-20" />
                ) : (
                  <CardTitle className="mt-2 text-2xl">{card.value}</CardTitle>
                )}
              </div>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              {card.description}
            </CardContent>
          </Card>
        ))}
      </motion.section>

      <motion.section
        className="grid gap-4 xl:grid-cols-2"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45, delay: 0.16, ease: "easeOut" }}
      >
        <TransactionPulseChart
          title="QRIS Transactions"
          description="Deposit dan withdrawal QRIS sukses per hari."
          data={pulse?.qris ?? []}
          isLoading={pulseQuery.isLoading}
        />
        <TransactionPulseChart
          title="NexusGGR Transactions"
          description="Deposit dan withdrawal NexusGGR sukses per hari."
          data={pulse?.nexusggr ?? []}
          isLoading={pulseQuery.isLoading}
        />
      </motion.section>
    </main>
  )
}

function TransactionPulseChart({
  title,
  description,
  data,
  isLoading,
}: {
  title: string
  description: string
  data: Array<{
    date: string
    deposit: number
    withdrawal: number
  }>
  isLoading: boolean
}) {
  return (
    <Card className="border-border/60 bg-card/85 shadow-sm backdrop-blur">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[320px] w-full rounded-2xl" />
        ) : (
          <ChartContainer config={chartConfig} className="h-[320px] w-full">
            <BarChart data={data}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tickFormatter={(value) =>
                  format(new Date(`${value}T00:00:00`), "dd MMM")
                }
              />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(value) =>
                      format(new Date(`${value}T00:00:00`), "dd MMM yyyy")
                    }
                  />
                }
              />
              <Bar
                dataKey="deposit"
                fill="var(--color-deposit)"
                radius={[8, 8, 0, 0]}
              />
              <Bar
                dataKey="withdrawal"
                fill="var(--color-withdrawal)"
                radius={[8, 8, 0, 0]}
              />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
