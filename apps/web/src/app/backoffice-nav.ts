import type { LucideIcon } from "lucide-react"
import {
  ArrowLeftRightIcon,
  BadgeDollarSignIcon,
  BookOpenTextIcon,
  Building2Icon,
  ChartNoAxesCombinedIcon,
  CreditCardIcon,
  Gamepad2Icon,
  LayoutDashboardIcon,
  PhoneCallIcon,
  QrCodeIcon,
  UsersRoundIcon,
  WalletCardsIcon,
} from "lucide-react"

export type BackofficeNavSection = {
  title: string
  items: Array<{
    title: string
    to: string
    icon: LucideIcon
  }>
}

export const backofficeNavSections: BackofficeNavSection[] = [
  {
    title: "Overview",
    items: [
      {
        title: "Dashboard",
        to: "/backoffice",
        icon: LayoutDashboardIcon,
      },
      {
        title: "Operational Pulse",
        to: "/backoffice/operational-pulse",
        icon: ChartNoAxesCombinedIcon,
      },
    ],
  },
  {
    title: "Master Data",
    items: [
      {
        title: "Users",
        to: "/backoffice/users",
        icon: UsersRoundIcon,
      },
      {
        title: "Tokos",
        to: "/backoffice/tokos",
        icon: Building2Icon,
      },
      {
        title: "Banks",
        to: "/backoffice/banks",
        icon: CreditCardIcon,
      },
      {
        title: "Players",
        to: "/backoffice/players",
        icon: WalletCardsIcon,
      },
    ],
  },
  {
    title: "Transaksi",
    items: [
      {
        title: "Transactions",
        to: "/backoffice/transactions",
        icon: ArrowLeftRightIcon,
      },
      {
        title: "Withdrawal",
        to: "/backoffice/withdrawal",
        icon: BadgeDollarSignIcon,
      },
      {
        title: "Topup QRIS",
        to: "/backoffice/nexusggr-topup",
        icon: QrCodeIcon,
      },
    ],
  },
  {
    title: "Integrasi",
    items: [
      {
        title: "Providers",
        to: "/backoffice/providers",
        icon: Gamepad2Icon,
      },
      {
        title: "Games",
        to: "/backoffice/games",
        icon: Gamepad2Icon,
      },
      {
        title: "Call Management",
        to: "/backoffice/call-management",
        icon: PhoneCallIcon,
      },
      {
        title: "API Docs",
        to: "/backoffice/api-docs",
        icon: BookOpenTextIcon,
      },
    ],
  },
]

export const backofficeRouteMeta: Record<
  string,
  { title: string; description: string }
> = {
  "/backoffice": {
    title: "Dashboard",
    description: "Operational control room untuk parity monitoring dan KPI utama.",
  },
  "/backoffice/operational-pulse": {
    title: "Operational Pulse",
    description: "Chart QRIS dan NexusGGR 7 hari dengan signal operasional real.",
  },
  "/backoffice/users": {
    title: "Users",
    description: "Manajemen akun operator, role, dan status akses.",
  },
  "/backoffice/tokos": {
    title: "Tokos",
    description: "Konfigurasi toko, callback URL, token, dan ownership.",
  },
  "/backoffice/banks": {
    title: "Banks",
    description: "Daftar rekening tujuan dengan workflow inquiry dan validasi.",
  },
  "/backoffice/players": {
    title: "Players",
    description: "Player mapping lokal-ke-upstream dengan audit yang jelas.",
  },
  "/backoffice/transactions": {
    title: "Transactions",
    description: "Pusat audit deposit, withdrawal, status, filter, dan export.",
  },
  "/backoffice/providers": {
    title: "Providers",
    description: "Katalog provider dari NexusGGR dengan cache parity-friendly.",
  },
  "/backoffice/games": {
    title: "Games",
    description: "Katalog game lintas provider dengan localized game names.",
  },
  "/backoffice/call-management": {
    title: "Call Management",
    description: "Kontrol call player, history, RTP, dan aksi operasional real-time.",
  },
  "/backoffice/nexusggr-topup": {
    title: "NexusGGR Topup",
    description: "Topup QRIS ke saldo NexusGGR dengan status polling dan expiry.",
  },
  "/backoffice/withdrawal": {
    title: "Withdrawal",
    description: "Wizard transfer dana dengan inquiry, fee, dan submit final.",
  },
  "/backoffice/api-docs": {
    title: "API Docs",
    description: "Referensi endpoint publik, payload, callback, dan contoh respons.",
  },
  "/backoffice/profile": {
    title: "Profile",
    description: "Preferensi operator, security, dan kontrol sesi.",
  },
}
