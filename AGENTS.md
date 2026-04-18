# AGENTS.md

This repository is a strict rewrite of `../justqiuv2`, not a product re-imagination.

## Mission

Build a production-grade React + Vite frontend and Go backend that preserves legacy behavior while improving:

- code quality
- UI polish
- responsiveness
- observability
- performance

## Hard Rules

1. Do not change API paths, methods, request fields, response fields, or status code behavior.
2. Do not change finance formulas.
3. Do not replace session auth with JWT for backoffice.
4. Do not replace Toko bearer tokens with JWT.
5. Do not collapse `pending`, `settle`, and `nexusggr` into one balance field.
6. Do not expose upstream raw payload fields that legacy hides.
7. Do not break role scoping: `dev` and `superadmin` see all, others are owner-scoped.
8. Only `dev` may mutate transactions, matching legacy policy.

## Source of Truth

Read and compare against:

- `../justqiuv2/routes/api.php`
- `../justqiuv2/app/Http/Controllers/Api/*`
- `../justqiuv2/app/Jobs/*`
- `../justqiuv2/app/Console/Commands/SettlePendingBalancesCommand.php`
- `../justqiuv2/app/Filament/**/*`
- `../justqiuv2/tests/Feature/*`
- `blueprint.md`

If implementation and blueprint diverge, verify against legacy code and tests first.

## Preferred Stack

Frontend:

- React
- Vite
- TypeScript
- shadcn/ui
- TanStack Query
- TanStack Table
- React Hook Form
- Zod
- lucide-react
- Framer Motion

Backend:

- Go
- chi
- pgx
- sqlc
- goose
- Redis
- asynq
- robfig/cron

## Mandatory shadcn Steps

From `apps/web`:

```bash
npx shadcn@latest init
npx shadcn@latest add dashboard-01
npx shadcn@latest add sidebar-01
npx shadcn@latest add login-01
npx shadcn@latest add signup-01
```

Then add missing primitives as needed, but keep shadcn as the default styling source.

## MCP Requirement

Codex should have shadcn MCP configured globally:

```toml
[mcp_servers.shadcn]
command = "npx"
args = ["shadcn@latest", "mcp"]
```

Recommended extra MCP capabilities if available in the client:

- browser automation
- PostgreSQL read-only
- Redis read-only

## UI Rules

- Inter for sans, Fira Code for mono
- dark, light, system toggle must exist
- dashboard must feel premium and operational, not generic
- matrix animation is allowed, but must degrade cleanly with reduced motion
- every admin table must support search, filter, pagination, and audit-friendly detail views

## Backend Rules

- explicit transactions for money movement
- `SELECT ... FOR UPDATE` on balance mutations
- keep request DTOs separate from DB records
- structured logs only
- all external calls wrapped in adapter interfaces
- queue jobs must be idempotent

## Test Gates

Do not call a feature done until these pass:

- contract tests for touched endpoints
- unit tests for touched formulas
- worker tests for touched jobs
- e2e tests for touched UI flows

## Known Legacy Caveat

`/api/v1/test` exists as a route in legacy but no controller implementation was found. Treat it as an unresolved parity issue, not an invitation to invent behavior.
