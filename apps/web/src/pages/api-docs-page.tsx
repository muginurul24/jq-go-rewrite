import { useMemo } from "react"

import {
  BookOpenTextIcon,
  CheckCheckIcon,
  CopyIcon,
  ShieldIcon,
  WebhookIcon,
} from "lucide-react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

type ParamDoc = {
  name: string
  type: string
  required: boolean
  description: string
}

type EndpointDoc = {
  method: "GET" | "POST"
  path: string
  name: string
  description: string
  params: ParamDoc[]
}

type ErrorCaseDoc = {
  status: number
  when: string
  response: Record<string, unknown>
}

type EndpointGroup = {
  group: string
  description: string
  endpoints: EndpointDoc[]
}

type CallbackDoc = {
  key: string
  title: string
  description: string
  method: "POST"
  target: string
  trigger: string
  responseExpectation: string
  notes: string[]
  fields: ParamDoc[]
}

const callbackDocs: CallbackDoc[] = [
  {
    key: "qris",
    title: "Callback Deposit QRIS / VA",
    description:
      "Dikirim ke `callback_url` toko setelah upstream QRIS/VA mengirim notifikasi pembayaran, project selesai memproses transaksi lokal, dan saldo toko diperbarui.",
    method: "POST",
    target: "<CALLBACK_URL_TOKO>",
    trigger:
      "Sesudah notifikasi pembayaran dari upstream diterima di `/api/webhook/qris` dan transaksi deposit lokal berhasil diproses.",
    responseExpectation:
      "Server toko sebaiknya membalas HTTP 2xx. Jika gagal atau timeout, project akan menganggap pengiriman gagal dan job callback akan retry.",
    notes: [
      "Payload yang diteruskan ke toko adalah payload callback upstream yang sudah lolos validasi, tanpa dibungkus object tambahan.",
      "Field acuan utama untuk mencocokkan transaksi deposit adalah `trx_id`.",
      "Field `status` dapat bernilai `pending`, `success`, `failed`, atau `expired`, sesuai data dari upstream.",
      "Header yang dikirim project adalah `Content-Type: application/json` dan `Accept: application/json`.",
    ],
    fields: [
      {
        name: "amount",
        type: "integer",
        required: true,
        description: "Nominal pembayaran dari upstream, dalam rupiah.",
      },
      {
        name: "terminal_id",
        type: "string",
        required: true,
        description: "Identifier user / terminal dari transaksi QRIS.",
      },
      {
        name: "trx_id",
        type: "string",
        required: true,
        description: "ID transaksi lokal yang sebelumnya diterima saat hit endpoint `/generate`.",
      },
      {
        name: "rrn",
        type: "string",
        required: false,
        description: "Reference Retrieval Number dari jaringan pembayaran, jika tersedia.",
      },
      {
        name: "custom_ref",
        type: "string",
        required: false,
        description: "Referensi kustom yang sebelumnya dikirim saat generate QRIS, jika ada.",
      },
      {
        name: "vendor",
        type: "string",
        required: false,
        description: "Nama vendor / channel upstream, jika tersedia.",
      },
      {
        name: "status",
        type: "string",
        required: true,
        description: "Status pembayaran dari upstream.",
      },
      {
        name: "created_at",
        type: "string",
        required: false,
        description: "Waktu transaksi dibuat di upstream.",
      },
      {
        name: "finish_at",
        type: "string",
        required: false,
        description: "Waktu transaksi selesai di upstream.",
      },
    ],
  },
]

const apiGroups: EndpointGroup[] = [
  {
    group: "User Management",
    description: "Mengelola user di platform game.",
    endpoints: [
      {
        method: "POST",
        path: "/user/create",
        name: "Create User",
        description:
          "Membuat player baru untuk toko yang sedang autentikasi. Wrapper akan membuat `ext_username` internal ke NexusGGR, tetapi response tetap memakai `username` lokal.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player untuk toko ini",
          },
        ],
      },
      {
        method: "POST",
        path: "/user/deposit",
        name: "User Deposit",
        description: "Deposit saldo ke akun user.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player tujuan deposit",
          },
          {
            name: "amount",
            type: "numeric",
            required: true,
            description: "Jumlah deposit (min: 10000)",
          },
          {
            name: "agent_sign",
            type: "string",
            required: false,
            description: "Tanda tangan agent untuk tracking",
          },
        ],
      },
      {
        method: "POST",
        path: "/user/withdraw",
        name: "User Withdraw",
        description: "Withdraw saldo dari akun user.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player yang ingin withdraw",
          },
          {
            name: "amount",
            type: "numeric",
            required: true,
            description: "Jumlah withdraw (min: 50000)",
          },
          {
            name: "agent_sign",
            type: "string",
            required: false,
            description: "Tanda tangan agent untuk tracking",
          },
        ],
      },
      {
        method: "POST",
        path: "/user/withdraw-reset",
        name: "User Withdraw Reset",
        description: "Reset withdraw untuk satu atau semua user.",
        params: [
          {
            name: "username",
            type: "string",
            required: false,
            description: "Username lokal player. Wajib diisi jika `all_users` tidak bernilai `true`.",
          },
          {
            name: "all_users",
            type: "boolean",
            required: false,
            description: "Reset semua user (true/false)",
          },
        ],
      },
      {
        method: "POST",
        path: "/transfer/status",
        name: "Transfer Status",
        description: "Cek status transfer berdasarkan agent_sign.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player",
          },
          {
            name: "agent_sign",
            type: "string",
            required: true,
            description: "Tanda tangan agent",
          },
        ],
      },
      {
        method: "POST",
        path: "/money/info",
        name: "Money Info",
        description: "Lihat saldo agent dan/atau user.",
        params: [
          {
            name: "username",
            type: "string",
            required: false,
            description: "Username lokal player (opsional)",
          },
          {
            name: "all_users",
            type: "boolean",
            required: false,
            description: "Tampilkan semua user (true/false)",
          },
        ],
      },
    ],
  },
  {
    group: "Games",
    description: "Provider dan game list.",
    endpoints: [
      {
        method: "GET",
        path: "/providers",
        name: "Provider List",
        description:
          "Daftar semua provider game yang tersedia. Response hanya mengembalikan field provider yang di-whitelist oleh wrapper.",
        params: [],
      },
      {
        method: "POST",
        path: "/games",
        name: "Game List",
        description:
          "Daftar game berdasarkan provider. Response disanitasi dan tidak meneruskan payload upstream mentah.",
        params: [
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
        ],
      },
      {
        method: "POST",
        path: "/games/v2",
        name: "Game List V2",
        description:
          "Daftar game (v2) dengan nama yang sudah dilokalkan. Response disanitasi dan hanya memuat field game yang di-whitelist.",
        params: [
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
        ],
      },
      {
        method: "POST",
        path: "/game/launch",
        name: "Game Launch",
        description: "Launch game dan dapatkan URL game.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player yang akan bermain",
          },
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
          {
            name: "game_code",
            type: "string",
            required: false,
            description: "Kode game spesifik",
          },
          {
            name: "lang",
            type: "string",
            required: false,
            description: "Bahasa (default: en)",
          },
        ],
      },
      {
        method: "POST",
        path: "/game/log",
        name: "Game Log",
        description: "Riwayat transaksi game (turnover).",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player",
          },
          {
            name: "game_type",
            type: "string",
            required: true,
            description: "Tipe game (slot, live, MN, SB)",
          },
          {
            name: "start",
            type: "string",
            required: true,
            description: "Waktu mulai (Y-m-d H:i:s)",
          },
          {
            name: "end",
            type: "string",
            required: true,
            description: "Waktu selesai (Y-m-d H:i:s)",
          },
          {
            name: "page",
            type: "integer",
            required: false,
            description: "Halaman (default: 0)",
          },
          {
            name: "perPage",
            type: "integer",
            required: false,
            description: "Jumlah per halaman (default: 100)",
          },
        ],
      },
    ],
  },
  {
    group: "Call Management",
    description: "Kelola call (RTP) untuk player.",
    endpoints: [
      {
        method: "GET",
        path: "/call/players",
        name: "Call Players",
        description:
          "Daftar player toko ini yang sedang aktif bermain. Data sudah difilter dan dipetakan kembali ke `username` lokal.",
        params: [],
      },
      {
        method: "POST",
        path: "/call/list",
        name: "Call List",
        description:
          "Daftar opsi call yang tersedia untuk kombinasi provider dan game tertentu. Response hanya mengembalikan field yang di-whitelist oleh wrapper.",
        params: [
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
          {
            name: "game_code",
            type: "string",
            required: true,
            description: "Kode game",
          },
        ],
      },
      {
        method: "POST",
        path: "/call/apply",
        name: "Call Apply",
        description: "Terapkan call ke user.",
        params: [
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
          {
            name: "game_code",
            type: "string",
            required: true,
            description: "Kode game",
          },
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player",
          },
          {
            name: "call_rtp",
            type: "integer",
            required: true,
            description: "Nilai RTP yang dipilih dari endpoint `/call/list`",
          },
          {
            name: "call_type",
            type: "integer",
            required: true,
            description: "1 = Common Free, 2 = Buy Bonus Free",
          },
        ],
      },
      {
        method: "POST",
        path: "/call/history",
        name: "Call History",
        description:
          "Riwayat call yang pernah dilakukan oleh player toko ini. Response difilter ke player milik toko yang sedang autentikasi.",
        params: [
          {
            name: "offset",
            type: "integer",
            required: false,
            description: "Offset paginasi (default: 0)",
          },
          {
            name: "limit",
            type: "integer",
            required: false,
            description: "Limit per halaman (default: 50)",
          },
        ],
      },
      {
        method: "POST",
        path: "/call/cancel",
        name: "Call Cancel",
        description: "Batalkan call yang masih pending.",
        params: [
          {
            name: "call_id",
            type: "integer",
            required: true,
            description: "ID call dari endpoint `/call/history`",
          },
        ],
      },
      {
        method: "POST",
        path: "/control/rtp",
        name: "Control RTP",
        description: "Set RTP control untuk satu user pada provider.",
        params: [
          {
            name: "provider_code",
            type: "string",
            required: true,
            description: "Kode provider",
          },
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username lokal player",
          },
          {
            name: "rtp",
            type: "numeric",
            required: true,
            description: "Nilai RTP target",
          },
        ],
      },
      {
        method: "POST",
        path: "/control/users-rtp",
        name: "Control Users RTP",
        description:
          "Set RTP control untuk banyak player sekaligus. Input tetap memakai field `user_codes`, tetapi nilainya adalah array `username` lokal milik toko ini.",
        params: [
          {
            name: "user_codes",
            type: "array",
            required: true,
            description: "Array username lokal player (min: 1)",
          },
          {
            name: "rtp",
            type: "numeric",
            required: true,
            description: "Nilai RTP target",
          },
        ],
      },
    ],
  },
  {
    group: "QRIS & Pembayaran",
    description: "Informasi merchant, generate QRIS, cek status, dan cek saldo.",
    endpoints: [
      {
        method: "POST",
        path: "/merchant-active",
        name: "Merchant Active",
        description:
          "Validasi merchant QRIS lalu kembalikan info store dan saldo lokal toko yang sedang autentikasi.",
        params: [
          {
            name: "label",
            type: "string",
            required: true,
            description: "Username user lokal yang dipakai sebagai label merchant",
          },
          {
            name: "client",
            type: "string",
            required: false,
            description: "Nama toko lokal sebagai client",
          },
        ],
      },
      {
        method: "POST",
        path: "/generate",
        name: "Generate QRIS",
        description:
          "Generate QRIS untuk deposit pada toko yang sedang autentikasi dan simpan transaksi pending lokal.",
        params: [
          {
            name: "username",
            type: "string",
            required: true,
            description: "Username pembayar",
          },
          {
            name: "amount",
            type: "integer",
            required: true,
            description: "Jumlah pembayaran (min: 10000)",
          },
          {
            name: "expire",
            type: "integer",
            required: false,
            description: "Waktu kadaluarsa (detik)",
          },
          {
            name: "custom_ref",
            type: "string",
            required: false,
            description: "Referensi kustom",
          },
        ],
      },
      {
        method: "POST",
        path: "/check-status",
        name: "Check Status",
        description:
          "Cek status transaksi QRIS berdasarkan trx_id lokal milik toko yang sedang autentikasi dan sinkronkan status transaksi.",
        params: [
          {
            name: "trx_id",
            type: "string",
            required: true,
            description: "ID transaksi",
          },
        ],
      },
      {
        method: "GET",
        path: "/balance",
        name: "Balance",
        description: "Lihat saldo lokal QRIS toko yang sedang autentikasi.",
        params: [],
      },
    ],
  },
]

const endpointResponses: Record<string, unknown> = {
  "/user/create": {
    success: true,
    username: "demo-player",
  },
  "/user/deposit": {
    success: true,
    agent: {
      code: "TOKO-DEMO",
      balance: 490000,
    },
    user: {
      username: "demo-player",
      balance: 19000,
    },
  },
  "/user/withdraw": {
    success: true,
    agent: {
      code: "TOKO-DEMO",
      balance: 540000,
    },
    user: {
      username: "demo-player",
      balance: 40000,
    },
  },
  "/user/withdraw-reset": {
    success: true,
    agent: {
      code: "TOKO-DEMO",
      balance: 500000,
    },
    user: {
      username: "demo-player",
      withdraw_amount: 25000,
      balance: 50000,
    },
  },
  "/transfer/status": {
    success: true,
    amount: 50000,
    type: "user_deposit",
    agent: {
      code: "TOKO-DEMO",
      balance: 300000,
    },
    user: {
      username: "demo-player",
      balance: 125000,
    },
  },
  "/money/info": {
    success: true,
    agent: {
      code: "TOKO-DEMO",
      balance: 500000,
    },
    user: {
      username: "demo-player",
      balance: 9000,
    },
  },
  "/providers": {
    success: true,
    providers: [
      {
        code: "PGSOFT",
        name: "PG Soft",
        status: 1,
      },
    ],
  },
  "/games": {
    success: true,
    provider_code: "PGSOFT",
    games: [
      {
        id: 1,
        game_code: "mahjong",
        game_name: "Mahjong Ways",
        banner: "https://cdn.example.com/mahjong.png",
        status: 1,
      },
    ],
  },
  "/games/v2": {
    success: true,
    provider_code: "PGSOFT",
    games: [
      {
        id: 1,
        game_code: "mahjong",
        game_name: {
          en: "Mahjong Ways",
          id: "Mahjong Ways",
        },
      },
    ],
  },
  "/game/launch": {
    success: true,
    launch_url: "https://example-game-launch.test/session/abc123",
  },
  "/game/log": {
    success: true,
    total_count: 1,
    page: 0,
    perPage: 100,
    logs: [
      {
        type: "slot",
        bet_money: 1000,
        win_money: 1500,
        txn_id: "TXN-1",
        txn_type: "credit",
      },
    ],
  },
  "/call/players": {
    success: true,
    data: [
      {
        username: "demo-player",
        provider_code: "PGSOFT",
        game_code: "mahjong",
        bet: 1000,
        balance: 9000,
        total_debit: 5000,
        total_credit: 3000,
        target_rtp: 80,
        real_rtp: 60,
      },
    ],
  },
  "/call/list": {
    success: true,
    calls: [
      {
        rtp: 92,
        call_type: "Free",
      },
    ],
  },
  "/call/apply": {
    success: true,
    called_money: 150000,
  },
  "/call/history": {
    success: true,
    data: [
      {
        id: 10,
        username: "demo-player",
        provider_code: "PGSOFT",
        game_code: "mahjong",
        bet: 1000,
        user_prev: 10000,
        user_after: 11000,
        agent_prev: 500000,
        agent_after: 499000,
        expect: 1200,
        missed: 200,
        real: 1000,
        rtp: 90,
        type: "common",
        status: 2,
        created_at: "2026-04-04T10:00:00Z",
        updated_at: "2026-04-04T10:05:00Z",
      },
    ],
  },
  "/call/cancel": {
    success: true,
    canceled_money: 42000,
  },
  "/control/rtp": {
    success: true,
    changed_rtp: 91.5,
  },
  "/control/users-rtp": {
    success: true,
    changed_rtp: 91.5,
  },
  "/merchant-active": {
    success: true,
    store: {
      name: "Demo Store",
      callback_url: "https://example-store.test/callback",
      token: "tok_demo_123456",
    },
    balance: {
      nexusggr: 1500000,
      pending: 250000,
      settle: 1250000,
    },
  },
  "/generate": {
    success: true,
    data: "000201010212...",
    trx_id: "TRX-001",
  },
  "/check-status": {
    success: true,
    trx_id: "TRX-001",
    status: "success",
  },
  "/balance": {
    success: true,
    pending_balance: 250000,
    settle_balance: 1250000,
    nexusggr_balance: 1500000,
  },
}

const endpointRequests: Record<string, Record<string, unknown> | null> = {
  "/user/create": {
    username: "demo-player",
  },
  "/user/deposit": {
    username: "demo-player",
    amount: 10000,
    agent_sign: "SIGN-DEMO-001",
  },
  "/user/withdraw": {
    username: "demo-player",
    amount: 50000,
    agent_sign: "SIGN-DEMO-002",
  },
  "/user/withdraw-reset": {
    username: "demo-player",
  },
  "/transfer/status": {
    username: "demo-player",
    agent_sign: "SIGN-DEMO-001",
  },
  "/money/info": {
    username: "demo-player",
  },
  "/providers": null,
  "/games": {
    provider_code: "PGSOFT",
  },
  "/games/v2": {
    provider_code: "PGSOFT",
  },
  "/game/launch": {
    username: "demo-player",
    provider_code: "PGSOFT",
    game_code: "mahjong",
    lang: "en",
  },
  "/game/log": {
    username: "demo-player",
    game_type: "slot",
    start: "2026-04-06 00:00:00",
    end: "2026-04-06 23:59:59",
    page: 0,
    perPage: 100,
  },
  "/call/players": null,
  "/call/list": {
    provider_code: "PGSOFT",
    game_code: "mahjong",
  },
  "/call/apply": {
    provider_code: "PGSOFT",
    game_code: "mahjong",
    username: "demo-player",
    call_rtp: 92,
    call_type: 1,
  },
  "/call/history": {
    offset: 0,
    limit: 50,
  },
  "/call/cancel": {
    call_id: 10,
  },
  "/control/rtp": {
    provider_code: "PGSOFT",
    username: "demo-player",
    rtp: 91.5,
  },
  "/control/users-rtp": {
    user_codes: ["demo-player", "demo-player-2"],
    rtp: 91.5,
  },
  "/merchant-active": {
    label: "demo-player",
    client: "TOKO-DEMO",
  },
  "/generate": {
    username: "demo-player",
    amount: 50000,
    expire: 300,
    custom_ref: "INV-20260406-001",
  },
  "/check-status": {
    trx_id: "TRX-001",
  },
  "/balance": null,
}

const invalidJSONError = {
  status: 400,
  when: "Payload JSON tidak valid atau body tidak bisa diparse.",
  response: { message: "Invalid JSON payload" },
} satisfies ErrorCaseDoc

const forbiddenError = {
  status: 403,
  when: "Bearer token toko tidak valid atau context toko tidak terbentuk.",
  response: { message: "Forbidden" },
} satisfies ErrorCaseDoc

function validationError(errors: Record<string, string>, when: string): ErrorCaseDoc {
  return {
    status: 422,
    when,
    response: {
      message: "Validation failed",
      errors,
    },
  }
}

function notFoundError(message: string, when: string): ErrorCaseDoc {
  return {
    status: 404,
    when,
    response: {
      success: false,
      message,
    },
  }
}

function upstreamFailureError(message: string, when: string): ErrorCaseDoc {
  return {
    status: 500,
    when,
    response: {
      success: false,
      message,
    },
  }
}

function internalMessageError(message: string, when: string): ErrorCaseDoc {
  return {
    status: 500,
    when,
    response: { message },
  }
}

const endpointErrorCases: Record<string, ErrorCaseDoc[]> = {
  "/user/create": [
    invalidJSONError,
    validationError({ username: "Username is required." }, "Field `username` kosong atau invalid."),
    validationError({ username: "Username has already been taken." }, "Username lokal sudah dipakai pada toko yang sama."),
    forbiddenError,
    upstreamFailureError("Failed to create user on upstream platform", "Upstream NexusGGR menolak pembuatan user."),
  ],
  "/user/deposit": [
    invalidJSONError,
    validationError({ amount: "Amount must be at least 10000." }, "Nominal deposit di bawah batas minimum."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    {
      status: 400,
      when: "Saldo lokal `nexusggr` toko tidak cukup untuk melakukan deposit.",
      response: {
        success: false,
        message: "Insufficient balance",
      },
    },
    upstreamFailureError("Failed to deposit user on upstream platform", "Upstream deposit gagal diproses."),
  ],
  "/user/withdraw": [
    invalidJSONError,
    validationError({ amount: "Amount must be at least 1." }, "Nominal withdraw tidak valid."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    {
      status: 400,
      when: "Saldo user pada upstream tidak cukup untuk withdraw.",
      response: {
        success: false,
        message: "User has insufficient balance on upstream platform",
      },
    },
    upstreamFailureError("Failed to get user balance from upstream platform", "Lookup saldo user upstream gagal sebelum withdraw."),
    upstreamFailureError("Failed to withdraw user on upstream platform", "Upstream withdraw gagal diproses."),
  ],
  "/user/withdraw-reset": [
    invalidJSONError,
    validationError({ username: "Username is required." }, "Mode single-user dipakai tetapi `username` kosong."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to reset withdraw on upstream platform", "Upstream menolak withdraw reset."),
  ],
  "/transfer/status": [
    invalidJSONError,
    validationError({ agent_sign: "Agent sign is required." }, "Field `agent_sign` kosong."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to get transfer status from upstream platform", "Lookup transfer status ke upstream gagal."),
  ],
  "/money/info": [
    invalidJSONError,
    validationError({ username: "Username must not exceed 50 characters." }, "Field `username` terlalu panjang."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to get balance information from upstream platform", "Lookup saldo upstream gagal."),
  ],
  "/providers": [
    upstreamFailureError("Failed to get provider list from upstream platform", "NexusGGR provider list gagal diambil."),
  ],
  "/games": [
    invalidJSONError,
    validationError({ provider_code: "Provider code is required." }, "Field `provider_code` kosong."),
    upstreamFailureError("Failed to get game list from upstream platform", "NexusGGR game list gagal diambil."),
  ],
  "/games/v2": [
    invalidJSONError,
    validationError({ provider_code: "Provider code is required." }, "Field `provider_code` kosong."),
    upstreamFailureError("Failed to get localized game list from upstream platform", "NexusGGR game list v2 gagal diambil."),
  ],
  "/game/launch": [
    invalidJSONError,
    validationError({ provider_code: "Provider code is required." }, "Field wajib untuk launch game tidak lengkap."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to launch game on upstream platform", "Launch game ditolak upstream."),
  ],
  "/game/log": [
    invalidJSONError,
    validationError({ game_type: "Game type is required." }, "Parameter log game tidak lengkap."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to get game logs from upstream platform", "Riwayat game gagal diambil dari upstream."),
  ],
  "/call/players": [
    forbiddenError,
    upstreamFailureError("Failed to get active players from upstream platform", "Daftar player aktif gagal diambil dari upstream."),
  ],
  "/call/list": [
    invalidJSONError,
    validationError({ provider_code: "Provider code is required." }, "Field `provider_code` atau `game_code` kosong."),
    upstreamFailureError("Failed to get call list from upstream platform", "Opsi call tidak berhasil diambil dari upstream."),
  ],
  "/call/apply": [
    invalidJSONError,
    validationError({ call_type: "Call type is required." }, "Payload apply call tidak lengkap."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to apply call on upstream platform", "Apply call ditolak oleh upstream."),
  ],
  "/call/history": [
    invalidJSONError,
    validationError({ limit: "Limit must be at least 1." }, "Parameter paginasi history tidak valid."),
    forbiddenError,
    upstreamFailureError("Failed to get call history from upstream platform", "Riwayat call gagal diambil dari upstream."),
  ],
  "/call/cancel": [
    invalidJSONError,
    validationError({ call_id: "Call id is required." }, "Field `call_id` kosong."),
    upstreamFailureError("Failed to cancel call on upstream platform", "Cancel call ditolak upstream."),
  ],
  "/control/rtp": [
    invalidJSONError,
    validationError({ rtp: "RTP is required." }, "Field control RTP tidak lengkap."),
    forbiddenError,
    notFoundError("Player not found", "Username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to change RTP on upstream platform", "Perubahan RTP ditolak upstream."),
  ],
  "/control/users-rtp": [
    invalidJSONError,
    validationError({ user_codes: "At least one user code is required." }, "Daftar `user_codes` kosong."),
    forbiddenError,
    notFoundError("Player not found", "Salah satu username lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to change RTP on upstream platform", "Perubahan RTP multi-user ditolak upstream."),
  ],
  "/merchant-active": [
    invalidJSONError,
    validationError({ label: "Label is required." }, "Field `label` kosong."),
    forbiddenError,
    internalMessageError("Failed to load balance", "Lookup/create saldo lokal toko gagal."),
  ],
  "/generate": [
    invalidJSONError,
    validationError({ amount: "Amount must be at least 10000." }, "Parameter generate QRIS tidak valid."),
    forbiddenError,
    upstreamFailureError("Failed to generate QRIS from upstream provider", "QRIS gagal dibuat di upstream."),
  ],
  "/check-status": [
    invalidJSONError,
    validationError({ trx_id: "Trx ID is required." }, "Field `trx_id` kosong."),
    forbiddenError,
    notFoundError("Transaction not found", "Transaksi QRIS lokal tidak ditemukan pada toko yang sedang autentikasi."),
    upstreamFailureError("Failed to get QRIS transaction status from upstream provider", "Status transaksi gagal diambil dari upstream."),
  ],
  "/balance": [
    forbiddenError,
    internalMessageError("Failed to load balance", "Lookup/create saldo lokal toko gagal."),
  ],
}

const callbackRequestExamples: Record<string, Record<string, unknown>> = {
  qris: {
    amount: 100000,
    terminal_id: "demo-player",
    trx_id: "TRX-001",
    rrn: "123456789012",
    custom_ref: "INV-20260406-001",
    vendor: "QRISVIP",
    status: "success",
    created_at: "2026-04-06 10:00:00",
    finish_at: "2026-04-06 10:00:12",
  },
}

const callbackResponseExample = {
  status: true,
  message: "Callback diterima",
}

export function ApiDocsPage() {
  const baseUrl = useMemo(() => {
    const configuredBaseURL = import.meta.env.VITE_PUBLIC_API_BASE_URL?.trim()
    if (configuredBaseURL) {
      return configuredBaseURL.replace(/\/+$/, "")
    }

    if (typeof window === "undefined") {
      return "/api/v1"
    }

    return `${window.location.origin}/api/v1`
  }, [])

  async function copySnippet(code: string) {
    await navigator.clipboard.writeText(code)
    toast.success("Snippet copied.")
  }

  return (
    <main className="grid gap-4 px-4 py-4 lg:px-6">
      <section className="grid gap-3 lg:grid-cols-3">
        <DocStat
          title="Public API"
          value={`${apiGroups.reduce((total, group) => total + group.endpoints.length, 0)} endpoints`}
          description="Semua endpoint publik rewrite yang mengikuti kontrak legacy."
          icon={BookOpenTextIcon}
        />
        <DocStat
          title="Callback"
          value={`${callbackDocs.length} flow`}
          description="Callback ke `callback_url` toko sesudah proses lokal selesai."
          icon={WebhookIcon}
        />
        <DocStat
          title="Security"
          value="Bearer Token"
          description="Semua endpoint publik memakai bearer token opaque ala legacy Sanctum."
          icon={ShieldIcon}
        />
      </section>

      <Card className="rounded-[1.35rem] border-border/70 bg-card/90 shadow-sm">
        <CardHeader className="gap-3">
          <CardTitle className="text-xl tracking-tight">Autentikasi</CardTitle>
          <CardDescription className="leading-6">
            Semua request API harus menyertakan Bearer Token di header Authorization.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3">
          <CodeSnippet
            eyebrow="Header"
            title="Authorization"
            description="Base auth contract mengikuti legacy."
            code={`Authorization: Bearer <YOUR_SANCTUM_TOKEN>`}
            language="http"
            onCopy={copySnippet}
          />
          <div className="rounded-[1rem] border border-border/70 bg-background/60 px-3.5 py-3">
            <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
              Base URL
            </p>
            <p className="mt-1 font-mono text-sm">{baseUrl}</p>
          </div>
        </CardContent>
      </Card>

      <section className="grid gap-4">
        <SectionHeader
          title="Callback ke callback_url Toko"
          description="Setelah upstream QRIS/VA mengirim webhook ke project ini, project akan meneruskan notifikasi yang relevan ke `callback_url` milik toko. Struktur bagian ini mengikuti dokumentasi legacy."
        />

        {callbackDocs.map((callback) => (
          <details
            key={callback.key}
            className="group rounded-[1.35rem] border border-border/70 bg-card/90 shadow-sm"
            open
          >
            <summary className="cursor-pointer list-none px-4 py-3.5">
              <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge className="rounded-full">{callback.method}</Badge>
                  <code className="font-mono text-sm">{callback.target}</code>
                  <span className="font-medium">{callback.title}</span>
                </div>
                <span className="text-xs text-muted-foreground">
                  {callback.fields.length} fields
                </span>
              </div>
            </summary>

            <div className="grid gap-4 border-t border-border/70 px-4 py-4">
              <p className="text-sm leading-6 text-muted-foreground">{callback.description}</p>

              <div className="grid gap-3 lg:grid-cols-2">
                <MetaCard title="Dikirim Kapan" value={callback.trigger} />
                <MetaCard
                  title="Respons yang Diharapkan"
                  value={callback.responseExpectation}
                />
              </div>

              <DocChecklist items={callback.notes} />
              <ParamTable title="Field yang Akan Diterima Server Toko" params={callback.fields} />

              <div className="grid gap-4 xl:grid-cols-2">
                <CodeSnippet
                  eyebrow="Rust request"
                  title="Contoh Request yang Diterima callback_url"
                  description="Request example menggunakan Rust `reqwest`, namun shape payload mengikuti legacy."
                  code={buildRustRequestSnippet(callback.target, callback.method, callbackRequestExamples[callback.key] ?? null, false)}
                  language="rust"
                  onCopy={copySnippet}
                />
                <CodeSnippet
                  eyebrow="JSON response"
                  title="Contoh Response dari Server Toko"
                  description="Contoh acknowledgement response yang sama seperti legacy docs."
                  code={jsonCode(callbackResponseExample)}
                  language="json"
                  onCopy={copySnippet}
                />
              </div>
            </div>
          </details>
        ))}
      </section>

      {apiGroups.map((group) => (
        <section key={group.group} className="grid gap-4">
          <SectionHeader title={group.group} description={group.description} />

          {group.endpoints.map((endpoint) => {
            const errorCases = endpointErrorCases[endpoint.path] ?? []

            return (
              <details
                key={endpoint.path}
                className="group rounded-[1.35rem] border border-border/70 bg-card/90 shadow-sm"
              >
                <summary className="cursor-pointer list-none px-4 py-3.5">
                  <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge
                        variant={endpoint.method === "GET" ? "outline" : "default"}
                        className="rounded-full"
                      >
                        {endpoint.method}
                      </Badge>
                      <code className="font-mono text-sm">{endpoint.path}</code>
                      <span className="font-medium">{endpoint.name}</span>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {endpoint.params.length > 0 ? `${endpoint.params.length} params` : "No params"}
                    </span>
                  </div>
                </summary>

                <div className="grid gap-4 border-t border-border/70 px-4 py-4">
                  <p className="text-sm leading-6 text-muted-foreground">{endpoint.description}</p>

                  {endpoint.params.length > 0 ? (
                    <ParamTable title="Parameter" params={endpoint.params} />
                  ) : (
                    <Card className="rounded-[1rem] border-dashed border-border/70 bg-background/40 shadow-none">
                      <CardContent className="px-3.5 py-3 text-sm text-muted-foreground">
                        Endpoint ini tidak memerlukan parameter.
                      </CardContent>
                    </Card>
                  )}

                  {errorCases.length ? (
                    <ErrorMatrix cases={errorCases} />
                  ) : null}

                  <div className="grid gap-4 xl:grid-cols-2">
                    <CodeSnippet
                      eyebrow="Rust request"
                      title="Contoh Request"
                      description="Request example menggunakan Rust `reqwest` dan payload contoh mengikuti field legacy."
                      code={buildRustRequestSnippet(baseUrl, endpoint.method, endpointRequests[endpoint.path] ?? null, true, endpoint.path)}
                      language="rust"
                      onCopy={copySnippet}
                    />
                    <CodeSnippet
                      eyebrow="JSON response"
                      title="Contoh Response"
                      description="Response example sama seperti baseline dokumentasi legacy."
                      code={jsonCode(endpointResponses[endpoint.path] ?? { success: true })}
                      language="json"
                      onCopy={copySnippet}
                    />
                  </div>
                </div>
              </details>
          )})}
        </section>
      ))}
    </main>
  )
}

function DocStat({
  title,
  value,
  description,
  icon: Icon,
}: {
  title: string
  value: string
  description: string
  icon: typeof BookOpenTextIcon
}) {
  return (
    <Card className="rounded-[1.35rem] border-border/70 bg-card/90 shadow-sm">
      <CardHeader className="space-y-2.5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <CardDescription>{title}</CardDescription>
            <CardTitle className="text-xl tracking-tight">{value}</CardTitle>
          </div>
          <span className="inline-flex size-10 items-center justify-center rounded-[1rem] bg-primary/10 text-primary">
            <Icon className="size-5" />
          </span>
        </div>
        <p className="text-sm leading-6 text-muted-foreground">{description}</p>
      </CardHeader>
    </Card>
  )
}

function SectionHeader({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <Card className="rounded-[1.2rem] border-border/70 bg-card/90 shadow-sm">
      <CardHeader className="gap-2">
        <CardTitle className="text-lg tracking-tight">{title}</CardTitle>
        <CardDescription className="leading-6">{description}</CardDescription>
      </CardHeader>
    </Card>
  )
}

function MetaCard({
  title,
  value,
}: {
  title: string
  value: string
}) {
  return (
    <Card className="rounded-[1rem] border-border/70 bg-background/60 shadow-none">
      <CardContent className="grid gap-1.5 px-3.5 py-3">
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
          {title}
        </p>
        <p className="text-sm leading-6 text-foreground">{value}</p>
      </CardContent>
    </Card>
  )
}

function ParamTable({
  title,
  params,
}: {
  title: string
  params: ParamDoc[]
}) {
  return (
    <Card className="rounded-[1.1rem] border-border/70 bg-background/50 shadow-none">
      <CardHeader className="pb-2">
        <CardTitle className="text-base tracking-tight">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-[1rem] border border-border/70">
          <Table className="min-w-[52rem]">
            <TableHeader>
              <TableRow>
                <TableHead>Nama</TableHead>
                <TableHead>Tipe</TableHead>
                <TableHead>Wajib</TableHead>
                <TableHead>Keterangan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {params.map((param) => (
                <TableRow key={`${param.name}-${param.type}`}>
                  <TableCell className="font-mono text-xs">{param.name}</TableCell>
                  <TableCell>{param.type}</TableCell>
                  <TableCell>
                    <Badge
                      variant={param.required ? "default" : "outline"}
                      className="rounded-full"
                    >
                      {param.required ? "Ya" : "Tidak"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {param.description}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function ErrorMatrix({ cases }: { cases: ErrorCaseDoc[] }) {
  return (
    <Card className="rounded-[1.1rem] border-border/70 bg-background/50 shadow-none">
      <CardHeader className="gap-1.5 pb-2">
        <CardTitle className="text-base tracking-tight">Error matrix</CardTitle>
        <CardDescription className="leading-6">
          Status code dan body utama yang benar-benar dipakai handler publik rewrite.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto rounded-[1rem] border border-border/70">
          <Table className="min-w-[64rem]">
            <TableHeader>
              <TableRow>
                <TableHead className="w-24">Status</TableHead>
                <TableHead className="min-w-[16rem]">Kapan terjadi</TableHead>
                <TableHead className="min-w-[20rem]">Contoh response</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cases.map((errorCase, index) => (
                <TableRow key={`${errorCase.status}-${index}`}>
                  <TableCell className="align-top">
                    <Badge variant="outline" className="rounded-full font-mono">
                      {errorCase.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="align-top text-sm leading-6 text-muted-foreground">
                    {errorCase.when}
                  </TableCell>
                  <TableCell className="align-top">
                    <pre className="overflow-x-auto rounded-xl border border-border/70 bg-background p-3 font-mono text-xs leading-6 whitespace-pre-wrap">
                      {jsonCode(errorCase.response)}
                    </pre>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function CodeSnippet({
  eyebrow,
  title,
  description,
  code,
  language,
  onCopy,
}: {
  eyebrow?: string
  title: string
  description: string
  code: string
  language: "rust" | "json" | "http"
  onCopy: (value: string) => Promise<void>
}) {
  return (
    <Card className="rounded-[1.1rem] border-border/70 bg-background/50 shadow-none">
      <CardHeader className="gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          {eyebrow ? (
            <p className="mb-1 text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
              {eyebrow}
            </p>
          ) : null}
          <CardTitle className="text-base tracking-tight">{title}</CardTitle>
          <CardDescription className="leading-6">{description}</CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="rounded-full font-mono text-[11px]">
            {language}
          </Badge>
          <Button variant="outline" className="rounded-xl" onClick={() => void onCopy(code)}>
            <CopyIcon className="size-4" />
            Copy
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <pre className="overflow-x-auto rounded-[1rem] border border-border/70 bg-background/70 p-3.5 font-mono text-[11px] leading-5">
          <code>{code}</code>
        </pre>
      </CardContent>
    </Card>
  )
}

function DocChecklist({ items }: { items: string[] }) {
  return (
    <div className="grid gap-2">
      {items.map((item) => (
        <div
          key={item}
          className="flex items-start gap-2.5 rounded-[1rem] border border-border/70 bg-background/50 px-3 py-2.5"
        >
          <CheckCheckIcon className="mt-0.5 size-4 text-primary" />
          <p className="text-sm leading-6 text-muted-foreground">{item}</p>
        </div>
      ))}
    </div>
  )
}

function jsonCode(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function buildRustRequestSnippet(
  baseUrl: string,
  method: "GET" | "POST",
  payload: Record<string, unknown> | null,
  includeAuth: boolean,
  path = "",
) {
  const headerImports = includeAuth
    ? "use reqwest::header::{ACCEPT, AUTHORIZATION, CONTENT_TYPE};"
    : "use reqwest::header::{ACCEPT, CONTENT_TYPE};"
  const needsJSON = method === "POST" && payload != null
  const authLine = includeAuth
    ? '\n    let token = "12|plain-text-token";\n'
    : "\n"
  const methodLine = method === "GET" ? ".get" : ".post"
  const contentTypeLine = needsJSON ? '\n        .header(CONTENT_TYPE, "application/json")' : ""
  const authorizationLine = includeAuth
    ? '\n        .header(AUTHORIZATION, format!("Bearer {token}"))'
    : ""
  const jsonImport = needsJSON ? "\nuse serde_json::json;" : ""
  const jsonBody = needsJSON
    ? `\n        .json(&json!(${rustObject(payload)}))`
    : ""

  return `${headerImports}${jsonImport}

#[tokio::main]
async fn main() -> anyhow::Result<()> {${authLine}    let client = reqwest::Client::new();

    let response = client
        ${methodLine}("${baseUrl}${path}")
        .header(ACCEPT, "application/json")${contentTypeLine}${authorizationLine}${jsonBody}
        .send()
        .await?;

    println!("{}", response.text().await?);
    Ok(())
}`
}

function rustObject(value: unknown, indent = 0): string {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return "[]"
    }

    const nextIndent = " ".repeat(indent + 4)
    const currentIndent = " ".repeat(indent)
    return `[\n${value.map((item) => `${nextIndent}${rustObject(item, indent + 4)}`).join(",\n")}\n${currentIndent}]`
  }

  if (value && typeof value === "object") {
    const entries = Object.entries(value)
    if (entries.length === 0) {
      return "{}"
    }

    const nextIndent = " ".repeat(indent + 4)
    const currentIndent = " ".repeat(indent)
    return `{\n${entries
      .map(([key, item]) => `${nextIndent}"${key}": ${rustObject(item, indent + 4)}`)
      .join(",\n")}\n${currentIndent}}`
  }

  if (typeof value === "string") {
    return `"${value}"`
  }

  return String(value)
}
