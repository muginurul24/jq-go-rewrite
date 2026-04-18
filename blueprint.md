# JustQiuV2 React-Go Rewrite Blueprint

Last updated: 2026-04-17  
Scope: full rewrite of `../justqiuv2` into React + Vite frontend and Go backend, without changing external behavior.

## 1. Goal

Rewrite the entire legacy Laravel + Filament project into:

- Frontend: React + Vite + TypeScript + shadcn/ui
- Backend: Go + PostgreSQL + Redis
- Deployment shape: one HTTP app for SPA + API, one worker, one scheduler

This is a rewrite, not a redesign of business rules. The new system may improve code quality, UI, UX, observability, and performance, but must keep the same:

- endpoint paths
- HTTP methods
- request fields
- response fields
- status code behavior
- finance formulas
- auth semantics
- role scoping
- webhook flow
- callback flow to `callback_url`
- schedule timing

## 2. Non-Negotiable Constraints

1. API contract parity is the highest priority. Internal implementation may change; external behavior may not.
2. Browser auth stays session-based with CSRF protection.
3. Toko-to-API auth stays opaque bearer token based, not JWT.
4. Redis remains the primary store for cache, session, and queue.
5. PostgreSQL remains the system of record.
6. Timezone must remain `Asia/Jakarta`.
7. Finance computations must be copied exactly, including legacy constants and rounding behavior.
8. Any new UX feature must be additive and must not break legacy flows.
9. Dark, light, and system theme modes are mandatory.
10. All admin list pages must support search, filter, pagination, sort, and detailed audit views.

## 3. Legacy Audit Summary

Legacy app base:

- Laravel 13
- Filament 5
- Livewire 4
- Sanctum token auth for Toko API
- Redis for cache, session, queue
- PostgreSQL as default DB
- App timezone `Asia/Jakarta`

Legacy domain model:

| Entity | Core fields | Notes |
| --- | --- | --- |
| `users` | `username`, `name`, `email`, `password`, `role`, `is_active` | Roles: `dev`, `superadmin`, `admin`, `user` |
| `tokos` | `user_id`, `name`, `callback_url`, `token`, `is_active` | New toko auto-creates one balance row |
| `balances` | `toko_id`, `settle`, `pending`, `nexusggr` | Three separate money buckets |
| `banks` | `user_id`, `bank_code`, `bank_name`, `account_number`, `account_name` | Withdrawal destination accounts |
| `players` | `toko_id`, `username`, `ext_username` | Local username mapped to upstream username |
| `transactions` | `toko_id`, `player`, `external_player`, `category`, `type`, `status`, `amount`, `code`, `note` | Main audit ledger |
| `incomes` | `ggr`, `fee_transaction`, `fee_withdrawal`, `amount` | Platform formula source |
| `notifications` | standard notifiable payload | Used by backoffice |
| `personal_access_tokens` | Sanctum-style opaque token hashes | Needed for Toko API auth |

Legacy admin feature map:

- Dashboard stats and charts
- User management
- Toko management + token generation
- Bank management + account inquiry
- Player list + money info action
- Transaction list + filters + export + audit drawer
- Provider list
- Games browser by provider
- Call Management
- NexusGGR topup via QRIS
- Withdrawal wizard
- API documentation page
- Login, register, profile, notifications, 2FA in production

## 4. Compatibility Contract Freeze

### 4.1 Toko API auth

- Auth header remains `Authorization: Bearer <opaque-token>`.
- Token format must stay compatible with existing Sanctum-issued tokens if legacy DB is migrated.
- Do not replace this with JWT.
- Go backend must verify hashed tokens against `personal_access_tokens` and scope them to `Toko`.

### 4.2 Browser auth

- Backoffice remains cookie-session based.
- Sessions stored in Redis.
- CSRF required for session-backed write requests.
- Same-site behavior remains `lax` unless deployment explicitly requires cross-site `none`.
- Login by username or email must remain supported.
- Production-grade TOTP/2FA remains mandatory in production.

### 4.3 Public and SPA routes

- `/login` and `/register` must remain valid entry points.
- `/backoffice/*` remains the main admin app path.
- `/api/*` reserved for backend API.
- SPA fallback must not swallow `/api/*`.

### 4.4 API routes to preserve

#### Webhook routes

| Route | Method | Auth | Behavior |
| --- | --- | --- | --- |
| `/api/webhook/qris` | `POST` | none, throttled `60/min` | validate callback, enqueue QRIS job, respond `{"status":true,"message":"OK"}` |
| `/api/webhook/disbursement` | `POST` | none, throttled `60/min` | validate callback, enqueue disbursement job, respond `{"status":true,"message":"OK"}` |

#### `/api/v1` routes

| Route | Method | Auth | Request parity | Response parity |
| --- | --- | --- | --- | --- |
| `/test` | `POST` | bearer Toko token | legacy route exists but no implementation found | freeze as unresolved legacy gap; do not invent behavior without confirmation |
| `/user/create` | `POST` | bearer | `username` | `success`, `username` |
| `/providers` | `GET` | bearer | none | `success`, `providers[]` |
| `/games` | `POST` | bearer | `provider_code` | `success`, `provider_code`, `games[]` |
| `/games/v2` | `POST` | bearer | `provider_code` | `success`, `provider_code`, `games[]` localized |
| `/user/deposit` | `POST` | bearer | `username`, `amount`, `agent_sign?` | `success`, `agent`, `user` |
| `/user/withdraw` | `POST` | bearer | `username`, `amount`, `agent_sign?` | `success`, `agent`, `user` |
| `/game/launch` | `POST` | bearer | `username`, `provider_code`, `game_code?`, `lang?` | `success`, `launch_url` |
| `/money/info` | `POST` | bearer | `username?`, `all_users?` | `success`, `agent`, optional `user`, optional `user_list` |
| `/game/log` | `POST` | bearer | `username`, `game_type`, `start`, `end`, `page?`, `perPage?` | `success`, `total_count`, `page`, `perPage`, `logs[]` |
| `/user/withdraw-reset` | `POST` | bearer | `username?`, `all_users?` | `success`, `agent`, optional `user`, optional `user_list` |
| `/transfer/status` | `POST` | bearer | `username`, `agent_sign` | `success`, `amount`, `type`, `agent`, `user` |
| `/call/players` | `GET` | bearer | none | `success`, `data[]` |
| `/call/list` | `POST` | bearer | `provider_code`, `game_code` | `success`, `calls[]` |
| `/call/apply` | `POST` | bearer | `provider_code`, `game_code`, `username`, `call_rtp`, `call_type` | `success`, `called_money` |
| `/call/history` | `POST` | bearer | `offset?`, `limit?` | `success`, `data[]` |
| `/call/cancel` | `POST` | bearer | `call_id` | `success`, `canceled_money` |
| `/control/rtp` | `POST` | bearer | `provider_code`, `username`, `rtp` | `success`, `changed_rtp` |
| `/control/users-rtp` | `POST` | bearer | `user_codes[]`, `rtp` | `success`, `changed_rtp` |
| `/merchant-active` | `POST` | bearer | `label`, `client?` | `success`, `store`, `balance` |
| `/generate` | `POST` | bearer | `username`, `amount`, `expire?`, `custom_ref?` | `success`, `data`, `trx_id` |
| `/check-status` | `POST` | bearer | `trx_id` | `success`, `trx_id`, `status` |
| `/balance` | `GET` | bearer | none | `success`, `pending_balance`, `settle_balance`, `nexusggr_balance` |

### 4.5 Input rules to preserve

- Local player usernames are always lowercased before validation/use.
- `user_codes` on `/control/users-rtp` are local usernames, not upstream IDs.
- `amount >= 10000` for:
  - `/user/deposit`
  - `/generate`
- `amount >= 1` for `/user/withdraw`.
- `/user/withdraw-reset` requires `username` unless `all_users=true`.
- `/call/apply` `call_type in [1,2]`.
- `/check-status` uses local `trx_id`.
- QRIS callback `merchant_id` must match configured `QRIS_GLOBAL_UUID`.
- Disbursement callback `merchant_id` must match configured `QRIS_GLOBAL_UUID`.

### 4.6 Response sanitation to preserve

The wrapper must keep hiding raw upstream fields exactly as legacy intended:

- map `ext_username` back to local `username`
- hide upstream `user_code`, `agent_code`, `msg`, `secret`, and similar raw fields
- whitelist only fields returned by the wrapper
- return local toko name as `agent.code`
- return local `username`, not upstream user code

## 5. Legacy Business Rules That Must Be Copied Exactly

### 5.1 QRIS deposit callback

When a QRIS deposit callback succeeds and transaction purpose is regular deposit:

- `plusIncome = amount * fee_transaction / 100`
- `finalPending = amount - plusIncome`
- `balances.pending += round(finalPending)`
- `incomes.amount += plusIncome`

When transaction purpose is `nexusggr_topup`:

- `finalNexusggr = amount * 100 / ggr`
- `balances.nexusggr += round(finalNexusggr)`
- `incomes.amount += amount - 1800`

Notes:

- the `1800` deduction is a hard legacy constant and must not be changed silently
- callback payload forwarded to toko callback must omit `merchant_id`

### 5.2 NexusGGR user deposit

- Validate player belongs to authenticated toko.
- Reject with `400` when `balance.nexusggr <= amount`.
- Call upstream `user_deposit`.
- On success:
  - `balances.nexusggr -= amount`
  - create `transactions` row:
    - `category = nexusggr`
    - `type = deposit`
    - `status = success`
    - `code = agent_sign`
    - `note.method = user_deposit`

### 5.3 NexusGGR user withdrawal

- Validate player belongs to authenticated toko.
- Check upstream `money_info` first.
- Reject if upstream user balance `< amount`.
- Call upstream `user_withdraw`.
- On success:
  - `balances.nexusggr += amount`
  - create `transactions` row:
    - `category = nexusggr`
    - `type = withdrawal`
    - `status = success`
    - `code = agent_sign`
    - `note.method = user_withdraw`

### 5.4 Withdraw reset

- Supports single user or all users.
- Creates one local `nexusggr` `withdrawal` success transaction per mapped player.
- `note.method = user_withdraw_reset`
- `note.scope = single_user | all_users`
- amount stored from upstream `withdraw_amount`

### 5.5 QRIS withdrawal flow

Wizard logic to preserve:

1. Select toko
2. Select destination bank and amount
3. Run inquiry only when continuing to next step
4. Show verified account + bank admin fee + platform fee + final deduction
5. Submit transfer only after successful inquiry

Formula:

- `platformFee = round(amount * fee_withdrawal / 100)`
- `estimatedDeduction = amount + platformFee`
- `finalDeduction = amount + bankFee + platformFee`

On successful transfer request:

- lock selected toko balance row
- reject if current `settle < finalDeduction`
- `balances.settle -= finalDeduction`
- create `transactions` row:
  - `category = qris`
  - `type = withdrawal`
  - `status = pending`
  - `code = partner_ref_no`
  - `note.purpose = withdrawal`
  - store `bank_id`, `bank_name`, `account_number`, `account_name`, `fee`, `platform_fee`, `inquiry_id`

### 5.6 Disbursement callback

If callback marks transaction `success`:

- set transaction status to `success`
- append `transaction_date` into note
- create income row if missing
- `incomes.amount += platform_fee`

If callback marks transaction non-success:

- set transaction status to callback status
- refund toko settle:
  - `balances.settle += transaction.amount + platform_fee + bank_fee`

Callback relay to toko callback:

- forward payload without `merchant_id`
- unique by `partner_ref_no`

### 5.7 Automatic pending-to-settle settlement

Schedule:

- weekdays
- every day at `16:00`
- timezone `Asia/Jakarta`

Rule per balance row:

- if `pending > 0`
- `amountToSettle = round(pending * 0.7)`
- `pending -= amountToSettle`
- `settle += amountToSettle`

### 5.8 Pending transaction expiry

- New pending transaction enqueues delayed expiry job
- delay: `30 minutes`
- when job runs:
  - if transaction still `pending`
  - and `created_at <= now - 30 minutes`
  - set status to `expired`

## 6. Jobs and Scheduler Parity

| Worker job | Unique key | Retries | Notes |
| --- | --- | --- | --- |
| `process_qris_callback` | `qris:<trx_id>` | 3, backoff `1s,5s,10s` | applies finance formulas, relays callback |
| `process_disbursement_callback` | `disbursement:<partner_ref_no>` | 3, backoff `1s,5s,10s` | settles or refunds, relays callback |
| `send_toko_callback` | `<eventType>:<reference>` | 4, backoff `10s,30s,60s` | request timeout `10s`, job timeout `15s` |
| `expire_pending_transaction` | transaction id | 1 | delayed by `30 minutes` |

Recommended Go implementation:

- queue: `asynq`
- scheduler: dedicated `cmd/scheduler` with `robfig/cron/v3`
- uniqueness: Redis-backed unique tasks
- transactional integrity: `SELECT ... FOR UPDATE`

## 7. Target Architecture

## 7.1 Repo layout

```text
.
|-- AGENTS.md
|-- blueprint.md
|-- Makefile
|-- docker-compose.yml
|-- .env.example
|-- pnpm-workspace.yaml
|-- apps
|   |-- web
|   |   |-- src
|   |   |   |-- app
|   |   |   |-- components
|   |   |   |-- features
|   |   |   |-- hooks
|   |   |   |-- lib
|   |   |   |-- routes
|   |   |   |-- styles
|   |   |   `-- types
|   |   `-- public
|   `-- api
|       |-- cmd
|       |   |-- server
|       |   |-- worker
|       |   `-- scheduler
|       |-- internal
|       |   |-- auth
|       |   |-- config
|       |   |-- db
|       |   |-- http
|       |   |-- jobs
|       |   |-- modules
|       |   |-- services
|       |   `-- security
|       `-- migrations
|-- contracts
|   |-- http
|   |-- fixtures
|   `-- openapi
`-- tests
    |-- e2e
    |-- integration
    `-- load
```

## 7.2 Frontend stack

- React 19
- Vite
- TypeScript strict mode
- React Router or TanStack Router
- TanStack Query
- TanStack Table
- React Hook Form + Zod
- shadcn/ui
- lucide-react
- Framer Motion
- Recharts for business charts
- lightweight canvas layer for matrix animation
- Sonner for notifications

Frontend rules:

- no direct `fetch` inside page components
- all API calls go through typed client layer
- all table state synced to URL when useful
- reduced-motion fallback required
- keyboard navigation and a11y labels required

## 7.3 Backend stack

- Go 1.24+
- `chi` or `echo`; prefer `chi` for low-overhead HTTP
- `pgx/v5` + `sqlc`
- `goose` SQL migrations
- Redis via `go-redis`
- `asynq` for queues
- `robfig/cron/v3` for scheduler
- session middleware backed by Redis
- CSRF middleware for browser routes
- bcrypt-compatible password verification for migrated users
- Sanctum-compatible opaque token verification for migrated Toko tokens

Backend rules:

- no ORM-heavy abstractions
- query layer generated and reviewed
- business logic in services, not handlers
- transaction boundaries explicit
- request DTOs separate from DB models
- structured logs only

## 8. Data Compatibility Strategy

### 8.1 Database

Core tables to keep compatible in name and meaning:

- `users`
- `tokos`
- `balances`
- `banks`
- `players`
- `transactions`
- `incomes`
- `notifications`
- `personal_access_tokens`

Recommended approach:

- keep core table names identical to legacy
- allow ancillary implementation tables to differ if behavior is unchanged
- preserve JSON shape inside `transactions.note` wherever legacy audit depends on it

### 8.2 Password migration

- Existing password hashes should be reused.
- Do not force password reset during cutover.
- Go auth layer must verify legacy hash formats that exist in production data.

### 8.3 Token migration

- Existing Sanctum bearer tokens must remain valid after DB cutover if possible.
- Re-implement token parsing and SHA-256 verification behavior compatible with legacy table data.
- Token regeneration from admin UI must preserve legacy semantics: old tokens revoked, new token stored and shown once.

## 9. UI and UX Blueprint

## 9.1 Design direction

Base direction:

- shadcn/ui as primary design system
- clean modern operational dashboard
- Inter for sans
- Fira Code for mono and audit payloads
- neutral + emerald + amber accent system
- responsive from mobile to desktop
- dark, light, system theme toggle in header/user menu

Do not ship generic filler dashboards. Replace demo content with actual operational value.

## 9.2 Mandatory shadcn bootstrap

From `apps/web`:

```bash
npx shadcn@latest init
npx shadcn@latest add dashboard-01
npx shadcn@latest add sidebar-01
npx shadcn@latest add login-01
npx shadcn@latest add signup-01
```

Then extend with standard primitives needed by the admin app:

- `button`
- `card`
- `dropdown-menu`
- `dialog`
- `sheet`
- `table`
- `pagination`
- `select`
- `form`
- `input`
- `textarea`
- `tabs`
- `badge`
- `tooltip`
- `calendar`
- `popover`
- `command`
- `chart`
- `data-table`
- `sonner`

## 9.3 Dashboard

Dashboard must become a premium operational cockpit, not a static copy.

Required dashboard behavior:

- animated matrix background with reduced-motion fallback
- cards for `pending`, `settle`, `nexusggr`, and `platform income`
- external balance cards for dev role only
- QRIS and NexusGGR charts
- live delta badges and subtle motion on refresh
- responsive card-to-chart-to-table composition

Recommended visual layers:

- canvas-based green matrix rain at low opacity
- subtle scanline/grid overlay
- motion-driven card reveal
- chart hover transitions
- mono-styled audit values for financial references

## 9.4 Navigation map

| Group | Pages |
| --- | --- |
| Master Data | Users, Tokos, Banks, Players |
| Transaksi | Transactions, NexusGGR Topup, Withdrawal |
| NexusGGR | Providers, Games, Call Management |
| Integrasi | API Documentation |

Suggested lucide icons:

- Dashboard: `LayoutDashboard`
- Users: `Users`
- Tokos: `Store`
- Banks: `Landmark`
- Players: `UserRound`
- Transactions: `ArrowUpDown`
- NexusGGR Topup: `QrCode`
- Withdrawal: `WalletCards`
- Providers: `Blocks`
- Games: `Gamepad2`
- Call Management: `PhoneCall`
- API Docs: `BookText`
- Theme toggle: `SunMoon`

## 9.5 Page parity requirements

### Dashboard

- 30-second refresh interval
- role-scoped totals
- dev-only external balance widgets
- 7-day QRIS and NexusGGR charts

### Users

- only `dev` and `superadmin`
- role editing limited by acting user role

### Tokos

- create/edit/delete
- callback URL
- active toggle
- generate/regenerate API token

### Banks

- bank inquiry action
- owner-scoped list

### Players

- owner-scoped list
- money info action

### Transactions

- search
- filters: category, type, status, toko, date range, amount range
- CSV/XLSX export
- detail drawer with monospace audit payload
- summary stats above table
- only `dev` can create/edit/delete

### Providers and Games

- provider caching
- games grouped by provider
- provider tabs
- search, pagination, sort

### Call Management

- active player list
- select player
- fetch call list
- apply call
- call history table
- local username mapping preserved

### NexusGGR Topup

- select toko
- amount input
- generate QRIS
- restore pending topup state
- poll/check payment status
- expire state handling

### Withdrawal

- 3-step wizard
- settle preview
- inquiry only when moving forward
- final deduction summary
- success state with reference

### API Documentation

- machine-verified route inventory
- example request/response payloads
- callback payload section

## 10. Security Blueprint

Mandatory controls:

- Redis-backed session store
- CSRF middleware for browser writes
- httpOnly session cookies
- secure cookies in production
- same-site `lax` by default
- token rotation for Toko API
- rate limit login, webhook, and auth-sensitive endpoints
- TOTP 2FA in production
- input validation on every endpoint
- request id on every request
- audit logging on token generation, withdrawal, callback failures

Keep same-or-better security without breaking legacy clients:

- do not require new headers for existing Toko API consumers
- do not add breaking signature requirements unless feature-flagged

## 11. Performance Blueprint

Targets:

- faster median response than legacy PHP stack
- low cold-start overhead
- p95 admin navigation under acceptable interactive thresholds
- p95 cached provider/game reads materially faster than legacy

Implementation direction:

- `sqlc` + `pgx` with prepared statements
- Redis cache for provider and game list
- Redis queue for callbacks and scheduled jobs
- SPA assets prebuilt and served from same origin
- Vite code-splitting by route
- TanStack Query caching for expensive admin screens
- table virtualization for large lists
- optimistic UI only where safe
- no N+1 query patterns

Cache semantics to preserve:

- provider list TTL: 1 day
- game list TTL: 1 day
- external dashboard balance snapshots: 5 minutes

## 12. Environment Blueprint

Reuse legacy variable names where possible:

```env
APP_NAME=
APP_ENV=
APP_DEBUG=
APP_URL=
APP_LOCALE=

DB_HOST=
DB_PORT=
DB_DATABASE=
DB_USERNAME=
DB_PASSWORD=

SESSION_DRIVER=redis
SESSION_LIFETIME=120
SESSION_DOMAIN=
SESSION_SECURE_COOKIE=
SESSION_SAME_SITE=lax

CACHE_STORE=redis
CACHE_PREFIX=

QUEUE_CONNECTION=redis

REDIS_HOST=
REDIS_PORT=
REDIS_PASSWORD=
REDIS_DB=
REDIS_CACHE_DB=

QRIS_BASE_URL=
QRIS_CLIENT=
QRIS_CLIENT_KEY=
QRIS_GLOBAL_UUID=

NEXUSGGR_BASE_URL=
NEXUSGGR_AGENT_CODE=
NEXUSGGR_AGENT_TOKEN=
```

Add Go-specific variables only when required, for example:

- `SESSION_SECRET`
- `CSRF_SECRET`
- `HTTP_PORT`
- `WORKER_CONCURRENCY`

## 13. MCP and Tooling Bootstrap

### 13.1 shadcn MCP

Codex global config should include:

```toml
[mcp_servers.shadcn]
command = "npx"
args = ["shadcn@latest", "mcp"]
```

Project expectation:

- shadcn MCP available before frontend implementation begins
- use it to browse blocks, components, and examples instead of manually copying random snippets

### 13.2 Additional MCPs recommended

If your client/tooling supports them, also enable:

- browser automation MCP for end-to-end UI verification
- PostgreSQL read-only MCP for schema inspection
- Redis read-only MCP for cache/session inspection

These are recommended for velocity, but the rewrite must not depend on a single vendor-specific MCP implementation.

## 14. Implementation Phases

### Phase 0: Freeze contract

- inventory all legacy routes
- capture golden request/response fixtures
- lock finance formulas
- lock role matrix

### Phase 1: Scaffold

- create monorepo structure
- bootstrap Vite React app
- bootstrap Go app, worker, scheduler
- configure Redis/Postgres/dev tooling
- configure shadcn and theme system

### Phase 2: Auth and core data

- users, tokos, balances, banks, players, transactions, incomes
- browser session auth
- Toko bearer token auth
- role-based authorization
- notifications

### Phase 3: API parity

- implement `/api/v1` routes one-by-one
- implement webhook routes
- add contract tests against frozen fixtures

### Phase 4: Backoffice parity

- dashboard
- master data pages
- transactions
- provider/games/call pages
- topup and withdrawal pages
- API docs page

### Phase 5: Polish

- matrix dashboard animation
- export jobs
- responsive refinement
- accessibility pass
- observability and tracing

### Phase 6: Verification and cutover

- run contract suite
- run e2e suite
- run load test
- dry-run migrated DB
- cut over with rollback plan

## 15. Testing Strategy

Minimum required test layers:

- unit tests for finance formulas
- integration tests for upstream adapters
- contract tests for every `/api/v1` route
- worker tests for callbacks and scheduler
- Playwright e2e for auth, token generation, topup, withdrawal, transaction filtering
- load tests for webhook and high-traffic admin API paths

Critical golden tests:

- QRIS callback success and topup success
- disbursement success and failure refund
- pending expiry after 30 minutes
- withdraw reset sanitized response
- transfer status sanitized response
- call players/history filtering by toko
- transaction export columns

## 16. Acceptance Criteria

Rewrite is considered done only when:

1. every legacy endpoint is present and contract-tested
2. every finance formula matches legacy outputs
3. browser auth uses Redis session + CSRF
4. Toko API uses legacy-compatible opaque tokens
5. schedule `weekday 16:00 Asia/Jakarta` is active
6. pending expiry `30 minutes` works
7. dashboard, transactions, withdrawal, topup, providers, games, call management, users, tokos, banks, players, API docs are all feature-complete
8. dark, light, system theme modes work
9. responsive layout is production-ready
10. load tests show lower latency than legacy in comparable scenarios

## 17. Known Legacy Gaps To Resolve Before Coding Around Them

- `/api/v1/test` route exists in `routes/api.php`, but no matching controller method was found in `WebhookTokoController`. Do not fabricate a replacement behavior without confirmation.
- Legacy code contains both API-facing and page-facing cache key naming variants for provider/game data. Rewrite should normalize the internal cache design, but preserve the same cached behavior and TTLs.

## 18. References

- Legacy source of truth: `../justqiuv2`
- shadcn MCP docs: `https://ui.shadcn.com/docs/mcp`
- shadcn Vite install docs: `https://ui.shadcn.com/docs/installation/vite`
- shadcn dark mode for Vite: `https://ui.shadcn.com/docs/dark-mode/vite`

## 19. Public API Contract Appendix

This appendix is the parity freeze for the public compatibility layer. It is intentionally repetitive, because this section is meant to be used during implementation and testing.

### 19.1 Authentication contract

- Public Toko API stays bearer-token based.
- Token source of truth remains `personal_access_tokens`.
- Tokenable entity remains `Toko`.
- Existing token regeneration semantics must be preserved:
  - revoke old tokens
  - issue a new token
  - keep operator-visible token handling behavior in the admin UI

### 19.2 Status enums to preserve

Transaction enums:

- `category`: `qris`, `nexusggr`
- `type`: `deposit`, `withdrawal`
- `status`: `pending`, `success`, `failed`, `expired`

Role enums:

- `dev`
- `superadmin`
- `admin`
- `user`

### 19.3 Request normalization rules

- Lowercase local `username` before validation and lookup on endpoints that use it.
- Lowercase all values in `user_codes` before validation on `/control/users-rtp`.
- Normalize QRIS status from upstream:
  - `pending` -> `pending`
  - `success` -> `success`
  - `paid` -> `success`
  - `failed` -> `failed`
  - `expired` -> `expired`

### 19.4 Endpoint-by-endpoint parity notes

#### `POST /api/v1/user/create`

- Validate username uniqueness per `toko_id`, not globally.
- Create local `players` row with:
  - local `username`
  - generated upstream `ext_username`
- Same username on another Toko remains valid.

#### `GET /api/v1/providers`

- Cache result for 1 day.
- Only expose whitelisted fields:
  - `code`
  - `name`
  - `status`

#### `POST /api/v1/games`

- Cache by `provider_code` for 1 day.
- Only expose whitelisted fields:
  - `id`
  - `game_code`
  - `game_name`
  - `banner`
  - `status`

#### `POST /api/v1/games/v2`

- Preserve localized `game_name` payload.
- Keep uncached initially, matching current legacy controller behavior.

#### `POST /api/v1/user/deposit`

- Reject with `404` if player does not belong to authenticated Toko.
- Reject with `400` when local `balance.nexusggr <= amount`.
- After upstream success:
  - decrement local `nexusggr`
  - create success transaction
  - return updated agent balance and upstream user balance

#### `POST /api/v1/user/withdraw`

- Reject with `404` if player does not belong to authenticated Toko.
- Query upstream balance before withdrawal.
- Reject with `400` if upstream balance is insufficient.
- After upstream success:
  - increment local `nexusggr`
  - create success transaction

#### `POST /api/v1/game/launch`

- `lang` defaults to `en`.
- Local username lookup maps to upstream `ext_username` before launch call.

#### `POST /api/v1/money/info`

- When `username` is present, resolve it locally first.
- When `all_users=true`, map upstream user list back to local usernames.
- Keep response shape limited to `agent`, optional `user`, optional `user_list`.

#### `POST /api/v1/game/log`

- Default values:
  - `page = 0`
  - `perPage = 100`
- Only map whitelisted fields from each log entry:
  - `type`
  - `bet_money`
  - `win_money`
  - `txn_id`
  - `txn_type`

#### `POST /api/v1/user/withdraw-reset`

- Support single-user and `all_users=true`.
- Create one local `nexusggr` withdrawal success transaction per mapped user returned by upstream.
- Preserve note payload semantics:
  - `method = user_withdraw_reset`
  - `scope = single_user | all_users`

#### `POST /api/v1/transfer/status`

- Resolve local username to `ext_username` before upstream call.
- Only expose:
  - `amount`
  - `type`
  - `agent`
  - `user`

#### `GET /api/v1/call/players`

- Return only players that belong to authenticated Toko.
- Drop records whose upstream `user_code` cannot be mapped to a local player.
- Strip raw upstream-only fields.

#### `POST /api/v1/call/list`

- Only expose:
  - `rtp`
  - `call_type`

#### `POST /api/v1/call/apply`

- Input uses local `username`.
- Upstream call uses mapped `ext_username`.
- Preserve `call_type` semantics:
  - `1 = Common Free`
  - `2 = Buy Bonus Free`

#### `POST /api/v1/call/history`

- Filter response to users belonging to authenticated Toko only.
- Map `user_code` back to local username.
- Strip upstream fields such as:
  - `user_code`
  - `agent_code`
  - `msg`

#### `POST /api/v1/call/cancel`

- Preserve minimal response:
  - `success`
  - `canceled_money`

#### `POST /api/v1/control/rtp`

- Resolve local username to `ext_username`.
- Preserve `changed_rtp` response shape.

#### `POST /api/v1/control/users-rtp`

- Input field remains `user_codes`.
- Values are local usernames.
- Upstream payload sends mapped external usernames.

#### `POST /api/v1/merchant-active`

- Preserve current legacy behavior exactly:
  - validate request
  - return local store info
  - return local balances
- Do not "fix" this to call upstream unless parity is explicitly re-approved.

#### `POST /api/v1/generate`

- Default `expire` to `300` when omitted.
- Create local `qris` deposit `pending` transaction with:
  - `purpose = generate`
  - optional `custom_ref`

#### `POST /api/v1/check-status`

- Find local QRIS transaction by authenticated Toko and `trx_id`.
- Sync local status from upstream normalized status.
- Do not apply financial callback logic here.

#### `GET /api/v1/balance`

- Return local:
  - `pending_balance`
  - `settle_balance`
  - `nexusggr_balance`

### 19.5 Webhook parity notes

#### `POST /api/webhook/qris`

- Validate `merchant_id` equals configured `QRIS_GLOBAL_UUID`.
- Remove `merchant_id` before relaying callback to Toko.
- Queue processing, do not execute heavy work inline.

#### `POST /api/webhook/disbursement`

- Validate `merchant_id` equals configured `QRIS_GLOBAL_UUID`.
- Remove `merchant_id` before relaying callback to Toko.
- Queue processing, do not execute heavy work inline.

## 20. Backoffice Screen Parity Appendix

This section converts legacy Filament pages/resources into React screens and BFF endpoints.

| Legacy screen/resource | Rewrite screen | Required parity |
| --- | --- | --- |
| `Dashboard` + widgets | `/backoffice` | role-scoped KPI cards, charts, 30s refresh semantics |
| `UserResource` | `/backoffice/users` | CRUD for allowed roles only |
| `TokoResource` | `/backoffice/tokos` | CRUD, callback URL, active toggle, token regeneration |
| `BankResource` | `/backoffice/banks` | CRUD, account inquiry action, owner scoping |
| `PlayerResource` | `/backoffice/players` | searchable list, owner scoping, money info action |
| `TransactionResource` | `/backoffice/transactions` | list, filters, export, detail drawer, dev-only mutation |
| `Provider` page | `/backoffice/providers` | cached provider list, search, status display |
| `Games` page | `/backoffice/games` | provider tabs, search, sort, pagination |
| `CallManagement` page | `/backoffice/call-management` | player selection, call list, apply action, history table |
| `NexusggrTopup` page | `/backoffice/nexusggr-topup` | QR generation, pending restore, status check |
| `Withdrawal` page | `/backoffice/withdrawal` | 3-step wizard, inquiry, final deduction, submit |
| `ApiDocumentation` page | `/backoffice/api-docs` | docs, examples, callback payload reference |

### 20.1 Dashboard parity details

- Preserve dev-only cards:
  - platform income
  - external QR pending
  - external QR settle
  - external agent balance
- Preserve universal cards:
  - pending
  - settle
  - NexusGGR
- Preserve 7-day QRIS and NexusGGR chart meaning.
- Replace Filament chart widgets with React chart cards, not different metrics.

### 20.2 Transaction screen parity details

Mandatory filters:

- category
- type
- status
- toko
- date range
- amount range

Mandatory actions:

- view detail
- export CSV/XLSX
- edit only if actor is `dev`
- delete only if actor is `dev`

Mandatory detail view sections:

- summary
- player/reference
- audit payload

### 20.3 Withdrawal screen parity details

Wizard steps must remain:

1. select Toko
2. select bank and amount
3. verify inquiry and submit

UX improvements are allowed only if these semantics remain unchanged:

- inquiry is triggered when moving forward, not via a separate primary flow
- final deduction summary is visible before submit
- success state shows reference and reset action

### 20.4 Topup screen parity details

- Toko selector
- amount input
- generate QR
- restore last pending topup for the selected Toko
- check payment status
- mark expired if pending topup passes expiry threshold

## 21. Environment Mapping Appendix

This section defines which legacy env names should be preserved, which should be adapted, and which are obsolete.

### 21.1 Preserve as-is

These names should stay unchanged unless there is a strong operational reason otherwise:

- `APP_NAME`
- `APP_ENV`
- `APP_DEBUG`
- `APP_URL`
- `APP_LOCALE`
- `DB_HOST`
- `DB_PORT`
- `DB_DATABASE`
- `DB_USERNAME`
- `DB_PASSWORD`
- `SESSION_DRIVER`
- `SESSION_LIFETIME`
- `SESSION_DOMAIN`
- `SESSION_SECURE_COOKIE`
- `SESSION_SAME_SITE`
- `QUEUE_CONNECTION`
- `CACHE_STORE`
- `CACHE_PREFIX`
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `REDIS_CACHE_DB`
- `QRIS_BASE_URL`
- `QRIS_CLIENT`
- `QRIS_CLIENT_KEY`
- `QRIS_GLOBAL_UUID`
- `NEXUSGGR_BASE_URL`
- `NEXUSGGR_AGENT_CODE`
- `NEXUSGGR_AGENT_TOKEN`

### 21.2 Replace with Go-native equivalents

Legacy PHP-specific env can be retired or mapped:

| Legacy env | Rewrite treatment |
| --- | --- |
| `DB_CONNECTION=pgsql` | implicit in Go config or preserved for compatibility |
| `OCTANE_SERVER` | obsolete in Go rewrite |
| `APP_KEY` | replace with Go-specific session/crypto secrets |
| `VITE_APP_NAME` | handled by frontend build config |

### 21.3 New env recommended for rewrite

- `HTTP_PORT`
- `SESSION_SECRET`
- `CSRF_SECRET`
- `TOKEN_DISPLAY_ENCRYPTION_KEY`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `PROMETHEUS_ENABLED`
- `QUEUE_WORKER_CONCURRENCY`

### 21.4 Configuration loading rule

Load config once at startup into typed structs. Do not scatter raw env access across handlers/services.

## 22. AGENTS.md Final Recommended Content

The project should not stop at the short draft above. This is the fuller recommended version for the actual file in the repository:

```md
# AGENTS.md

## Identity

This repository rewrites `../justqiuv2` into React + Vite + Go.
The rewrite must preserve public behavior and business meaning.

## Source of truth

Always verify against:

- ../justqiuv2/routes/api.php
- ../justqiuv2/routes/web.php
- ../justqiuv2/routes/console.php
- ../justqiuv2/app/Http/Controllers/Api/*
- ../justqiuv2/app/Http/Requests/Api/*
- ../justqiuv2/app/Jobs/*
- ../justqiuv2/app/Console/Commands/*
- ../justqiuv2/app/Models/*
- ../justqiuv2/app/Policies/*
- ../justqiuv2/app/Services/*
- ../justqiuv2/app/Filament/*
- ../justqiuv2/tests/Feature/*
- blueprint.md

## Never change these without explicit approval

1. Public API paths
2. Public API methods
3. Public request field names
4. Public response field names
5. Financial formulas
6. Job/scheduler meaning
7. Role/access semantics
8. Pending/settle/NexusGGR meaning
9. Callback field stripping rules
10. Toko bearer token auth style

## Frontend rules

- Use React + Vite + TypeScript.
- Use shadcn/ui first.
- Use lucide-react.
- Use Inter and Fira Code.
- Keep dark/light/system mode.
- Every major list screen must support search/filter/sort/pagination.
- Keep animations tasteful and cheap.
- Respect reduced motion.

## Backend rules

- Use Go.
- Use explicit handlers, services, repositories.
- Use pgx + sqlc.
- Use Redis for session, cache, queue.
- Use transactions and row locks for money movement.
- Jobs must be idempotent.
- Validate all DTOs.
- Log in structured JSON.

## Migration rules

- Freeze contract before coding.
- Build parity fixtures before endpoint implementation.
- Migrate module by module.
- Compare rewrite behavior to legacy for each step.
- Do not "improve" behavior that might be contractual.

## Security rules

- Backoffice uses session auth + CSRF.
- Public Toko API uses opaque bearer tokens.
- Cookies are hardened in production.
- MFA stays enabled in production.
- Authorization lives on the server, not only in the UI.

## Testing rules

- Unit test formulas and normalization.
- Contract test all public endpoints.
- Integration test provider adapters and auth flows.
- E2E test auth, dashboard, transactions, topup, withdrawal, token regen.
- Load test critical endpoints before production.

## Performance rules

- No N+1.
- No ORM-heavy abstractions.
- Cache provider/game reads.
- Keep request path short.
- Do not use client-side mega-state for server data.

## MCP guidance

- Configure shadcn MCP before frontend implementation.
- Use browser automation tools for UI verification.
- Use DB/Redis inspection tools during parity debugging.

## Done definition

Work is only done when:

- parity tests pass
- financial tests pass
- role tests pass
- UI behavior is verified
- logs/metrics/traces are present where relevant
```

## 23. Detailed Implementation Sequence Appendix

This appendix expands the main sequence into an engineer-friendly checklist.

### 23.1 Day 0 baseline

- create route inventory from legacy
- create endpoint fixture inventory
- create table inventory
- create role matrix
- create screen matrix
- create finance formula doc

### 23.2 Infrastructure setup

- create monorepo
- set up pnpm workspace
- set up Go module
- set up Postgres and Redis locally
- add lint/format/test commands
- configure CI basics

### 23.3 Frontend shell

- initialize Vite React app
- initialize shadcn
- install required blocks and primitives
- wire theme provider
- wire router
- build sidebar shell
- build auth layout

### 23.4 Backend shell

- bootstrap chi app
- add config loader
- add logger
- add DB and Redis clients
- add request id and recovery middleware
- add session and CSRF middleware
- add auth middleware

### 23.5 Schema and repositories

- write SQL migrations
- generate sqlc queries
- create repository packages
- add transaction helper

### 23.6 Auth and users

- session login/logout
- username-or-email login support
- current-user endpoint for backoffice bootstrap
- bearer token parsing for Toko API
- token regeneration service for Tokos

### 23.7 Public compatibility endpoints

Implement in this order:

1. `/providers`
2. `/games`
3. `/games/v2`
4. `/balance`
5. `/money/info`
6. `/user/create`
7. `/game/launch`
8. `/call/list`
9. `/call/history`
10. `/call/players`
11. `/call/apply`
12. `/call/cancel`
13. `/control/rtp`
14. `/control/users-rtp`
15. `/user/deposit`
16. `/user/withdraw`
17. `/user/withdraw-reset`
18. `/transfer/status`
19. `/merchant-active`
20. `/generate`
21. `/check-status`
22. webhook endpoints

### 23.8 Background processing

- QRIS callback worker
- disbursement callback worker
- callback relay worker
- delayed pending expiry worker
- weekday settlement scheduler

### 23.9 Backoffice pages

Implement in this order:

1. login
2. register
3. dashboard
4. tokos
5. banks
6. players
7. transactions
8. users
9. providers
10. games
11. call management
12. NexusGGR topup
13. withdrawal
14. API docs

### 23.10 Finalization

- export flows
- notifications
- metrics/tracing
- load tests
- staging parity report
- cutover checklist

## 24. Parity Sign-off Checklist

Before declaring the rewrite ready, sign off every line below:

- route inventory matches legacy
- public auth style matches legacy
- response fixtures match legacy
- all finance formulas are tested
- callback relay strips `merchant_id`
- weekday settlement runs at `16:00` Asia/Jakarta
- pending expiry is `30 minutes`
- same username allowed across Tokos
- transaction mutations restricted to `dev`
- dashboard cards are role-correct
- transaction filters/search/pagination/export work
- withdrawal wizard behavior is correct
- topup pending restore behavior is correct
- provider/game cache TTLs are correct
- dark/light/system themes work
- mobile layout is acceptable
- metrics/logging/tracing are enabled
- load tests beat legacy baseline or are within an explicitly accepted margin
