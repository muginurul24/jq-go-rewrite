# JustQiuV2 Rewrite

Strict rewrite dari `../justqiuv2` ke:

- Frontend: React + Vite + TypeScript
- Backend: Go + chi + pgx
- UI: shadcn/ui
- Infra utama: PostgreSQL + Redis

Target utama repo ini adalah parity terhadap legacy Laravel sambil menaikkan kualitas implementasi, performa, observability, dan UX.

CI sekarang juga memverifikasi:

- backend test + build
- frontend lint + build
- managed Playwright E2E
- ops preflight (`smoke-runtime` + `reconcile-finance` + `/metrics`)

## Status Implementasi

Sudah hidup saat ini:

- backoffice session auth + CSRF
- backoffice MFA/TOTP challenge + profile setup/disable
- login, register, logout, profile
- dashboard shell + theme dark/light/system
- users, tokos, players
- transactions list/detail/export CSV/XLSX
- banks
- withdrawal wizard
- NexusGGR topup
- call-management end-to-end
- public compatibility API utama `/api/v1/*`
- webhook QRIS dan disbursement
- worker queue + scheduler settle weekday 16:00 Asia/Jakarta
- DB integration test untuk side effect finansial webhook
- DB integration test untuk auth + MFA encryption/recovery flow
- managed Playwright E2E untuk auth, route smoke, audit flow, dan CRUD smoke

Masih ada area yang perlu disempurnakan sebelum deployment production final:

- hardening production di environment target sebenarnya
- staging rehearsal dan provider smoke pada environment target

## Struktur Runtime

Saat development, jalankan 4 proses:

1. `apps/web` untuk Vite frontend di `http://localhost:5173`
2. `apps/api/cmd/server` untuk API di `http://localhost:8080`
3. `apps/api/cmd/worker` untuk queue worker Redis
4. `apps/api/cmd/scheduler` untuk cron settlement

Frontend Vite sudah mem-proxy request berikut ke backend:

- `/api`
- `/backoffice/api`
- `/health`

## Prasyarat

Minimal tool yang dibutuhkan:

- Node.js 22+
- pnpm 10+
- Go 1.26+
- PostgreSQL 17+
- Redis 8+

Opsional namun direkomendasikan:

- Docker Compose atau Podman Compose untuk infra PostgreSQL + Redis

## Setup Environment

Salin file env terlebih dahulu:

```bash
cp .env.example .env
```

Nilai minimum yang wajib benar:

- `DB_HOST`
- `DB_PORT`
- `DB_DATABASE`
- `DB_USERNAME`
- `DB_PASSWORD`
- `REDIS_HOST`
- `REDIS_PORT`
- `SESSION_SECRET`
- `CSRF_SECRET`
- `CSRF_TRUSTED_ORIGINS`
- `TOKEN_DISPLAY_ENCRYPTION_KEY`

Untuk local dev, default `.env.example` sudah aman dipakai selama PostgreSQL dan Redis disiapkan sesuai nilainya.

Semua command Go di repo ini sekarang otomatis mencari `.env` ke parent directory, jadi `pnpm ...` dan `make ...` yang dijalankan dari root repo akan tetap membaca `.env` root dengan benar.

Untuk mode Vite dev, biarkan `CSRF_TRUSTED_ORIGINS` berisi host frontend seperti default di `.env.example`. Ini penting agar POST backoffice dari `localhost:5173` atau `localhost:4173` tidak ditolak middleware CSRF.

## Menjalankan Tanpa Docker/Podman

Mode ini cocok jika PostgreSQL dan Redis terpasang langsung di host.

### 1. Install dependency project

```bash
pnpm install --recursive
```

### 2. Install PostgreSQL dan Redis

Contoh Ubuntu/Debian:

```bash
sudo apt update
sudo apt install -y postgresql redis-server
sudo systemctl enable --now postgresql redis-server
```

Contoh macOS:

```bash
brew install postgresql@17 redis
brew services start postgresql@17
brew services start redis
```

### 3. Buat database dan user

Paling aman gunakan user khusus, lalu cocokkan ke `.env`.

Contoh:

```bash
sudo -u postgres createuser --createdb justqiu
sudo -u postgres psql -c "ALTER USER justqiu WITH PASSWORD 'justqiu';"
sudo -u postgres createdb -O justqiu justqiuv2_rewrite
```

Lalu ubah `.env`:

```env
DB_DATABASE=justqiuv2_rewrite
DB_USERNAME=justqiu
DB_PASSWORD=justqiu
```

### 4. Jalankan migrasi

Dari root repo:

```bash
make migrate-up
```

Alternatif via npm script:

```bash
pnpm migrate:up
```

Jika targetnya adalah database production legacy yang tabel-tabelnya sudah ada dan Anda hanya ingin mulai memakai goose versioning rewrite tanpa menjalankan DDL lagi, gunakan baseline:

```bash
make migrate-baseline
```

atau:

```bash
pnpm migrate:baseline
```

`baseline` aman untuk skenario cutover ke database legacy yang sudah kompatibel, karena:

- tidak menjalankan `up`
- tidak menjalankan `reset`
- hanya membuat/menandai `goose_db_version` ke versi migrasi rewrite terbaru

Jangan gunakan `migrate:seed` atau `migrate:reset` pada database production legacy.

Jika rewrite dipasang side-by-side pada VPS yang masih menjalankan legacy dengan database dan Redis yang sama, nyalakan hanya API rewrite terlebih dahulu. Worker dan scheduler rewrite harus tetap mati sampai cutover window supaya callback queue dan settlement tidak diproses ganda.

Untuk reset penuh database lalu seed akun dev default:

```bash
make migrate-seed
```

atau:

```bash
pnpm migrate:seed
```

Command ini akan:

- menjalankan reset database
- menjalankan ulang seluruh migrasi
- membuat row `incomes` baseline legacy:
  - `ggr=7`
  - `fee_transaction=3`
  - `fee_withdrawal=15`
  - `amount=0`
- membuat akun dev default:
  - username: `justqiu`
  - password: `justqiu`
  - email: `justqiu@local.test`

Untuk reset penuh database dan membuat dataset demo yang lebih kaya untuk audit UI/backoffice:

```bash
make migrate-seed-demo
```

atau:

```bash
pnpm migrate:seed:demo
```

## Audit Dengan Clone Database Production

Kalau Anda sudah membuat clone database production lokal, jalankan rewrite di atas clone itu dengan override `DB_DATABASE`, lalu audit browser khusus clone:

```bash
DB_DATABASE=justqiuv2_prod_clone pnpm dev:api
DB_DATABASE=justqiuv2_prod_clone pnpm dev:worker
DB_DATABASE=justqiuv2_prod_clone pnpm dev:scheduler
pnpm dev:web
```

Lalu jalankan Playwright audit terhadap data real clone:

```bash
PLAYWRIGHT_BASE_URL=http://localhost:5173 \
E2E_DEV_USERNAME=localproddev \
E2E_DEV_PASSWORD=justqiu8 \
pnpm test:web:e2e:clone
```

Suite ini memang `opt-in` dan tidak ikut dijalankan oleh CI managed biasa. Ia hanya aktif saat script clone di atas dipakai, karena membutuhkan database clone production dan environment upstream yang siap.

Suite ini memeriksa:

- semua route backoffice utama bisa dibuka
- tidak ada `pageerror`, `console.error`, React key warning, atau HTTP `5xx`
- detail transaksi dan export CSV hidup
- select toko/bank pada topup dan withdrawal tidak blank
- providers, games, dan call-management tetap usable pada data clone

Untuk menjalankan preflight staging penuh terhadap clone yang sama:

```bash
DB_DATABASE=justqiuv2_prod_clone \
SMOKE_USERNAME=localproddev \
SMOKE_PASSWORD=justqiu8 \
SMOKE_TOKO_TOKEN='<token hasil regenerate toko>' \
pnpm preflight:staging
```

`preflight:staging` sekarang menghormati override env yang Anda kirim dari shell. Jadi:

- `DB_DATABASE` tidak lagi diam-diam kembali ke database demo dari `.env`
- `SMOKE_USERNAME` dan `SMOKE_PASSWORD` bisa diarahkan ke akun clone/staging
- `SMOKE_TOKO_TOKEN` bisa diarahkan ke token bearer hasil regenerate di environment target

Untuk validasi packaging deploy via Podman:

```bash
podman build -f apps/api/Dockerfile -t justqiuv2-rewrite-api:test .
podman build -f apps/web/Dockerfile -t justqiuv2-rewrite-web:test .
```

Image reference di Dockerfile dan compose sudah dibuat fully qualified agar build tetap stabil di Docker maupun Podman.

Dataset demo ini menambahkan:

- 1 owner demo:
  - username: `demo-owner`
  - password: `justqiu`
- 1 toko demo lengkap dengan token API
- 1 rekening bank demo
- 1 player demo
- 6 transaksi demo
- `incomes.amount=65000` agar overview dan transaction audit punya data nyata

Smoke account default untuk runtime script:

- username: `justqiu`
- password: `justqiu`

Cek status migrasi:

```bash
make migrate-status
```

Untuk database legacy yang sudah dibaseline, status harus menunjukkan versi migrasi rewrite terbaru sebagai applied sebelum aplikasi rewrite diarahkan ke production.

### 5. Jalankan semua proses aplikasi

Terminal 1:

```bash
pnpm dev:web
```

Terminal 2:

```bash
pnpm dev:api
```

Terminal 3:

```bash
pnpm dev:worker
```

Terminal 4:

```bash
pnpm dev:scheduler
```

### 6. Verifikasi runtime

Health check:

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

Expected:

- `/health/live` mengembalikan `200`
- `/health/ready` mengembalikan `200` dan service `postgres` + `redis` status `ok`

UI:

- buka `http://localhost:5173/login`
- login dengan akun seed:
  - username: `justqiu`
  - password: `justqiu`

## Smoke dan Reconciliation

Setelah API berjalan, gunakan dua command ini untuk validasi staging atau production-like environment:

Runtime smoke:

```bash
pnpm smoke:runtime
```

atau:

```bash
make smoke-runtime
```

Environment variable opsional:

- `SMOKE_BASE_URL`
- `SMOKE_USERNAME`
- `SMOKE_PASSWORD`
- `SMOKE_TOKO_TOKEN`

`SMOKE_TOKO_TOKEN` dipakai untuk ikut mengetes endpoint public bearer-auth seperti `/api/v1/balance` dan `/api/v1/merchant-active`.

Jika env NexusGGR lengkap (`NEXUSGGR_BASE_URL`, `NEXUSGGR_AGENT_CODE`, `NEXUSGGR_AGENT_TOKEN`), script juga akan mengetes `/api/v1/providers`.

Snapshot reconciliation finansial:

```bash
pnpm reconcile:finance
```

atau:

```bash
make reconcile-finance
```

Preflight staging lengkap:

```bash
pnpm preflight:staging
```

atau:

```bash
make preflight-staging
```

Command ini akan menjalankan:

- runtime smoke
- finance reconciliation snapshot
- probe `/metrics`

Script ini mengeluarkan:

- total record inti
- distribusi role user
- coverage toko dan balance
- total `pending/settle/nexusggr`
- snapshot `incomes`
- total transaksi per `category/type/status`
- pending transaction lebih tua dari 30 menit
- consistency checks seperti toko tanpa row balance

Runbook cutover ada di [docs/production-cutover.md](./docs/production-cutover.md).

### 7. Promosikan user pertama menjadi operator global

Jika Anda tidak memakai `migrate-seed`, register dari UI membuat user role default `user`. Untuk akses operator penuh, promosikan manual via SQL:

```sql
UPDATE users
SET role = 'dev', is_active = TRUE
WHERE email = 'email-anda@example.com';
```

Role yang valid:

- `dev`
- `superadmin`
- `admin`
- `user`

Jika ingin parity akses terluas seperti legacy, gunakan `dev`.

## Menjalankan Dengan Docker Compose atau Podman Compose

Ada dua mode:

- infra-only: hanya PostgreSQL + Redis di container, aplikasi tetap dijalankan dari host
- full stack: `web + api + worker + scheduler + postgres + redis` semua di container

### 1. Install dependency project

```bash
pnpm install --recursive
cp .env.example .env
```

### 2. Naikkan infra-only

Dengan Docker:

```bash
docker compose up -d
```

Dengan Podman:

```bash
podman compose up -d
```

Lihat status:

```bash
docker compose ps
```

atau:

```bash
podman compose ps
```

Compose file yang ada sekarang menjalankan:

- PostgreSQL 17 di `localhost:5432`
- Redis 8 di `localhost:6379`

Jika port bentrok, ubah `docker-compose.yml` dan `.env` bersama-sama.

### 3. Jalankan migrasi

Jika memakai infra-only, jalankan dari host:

```bash
make migrate-up
```

Atau reset penuh + seed baseline legacy (`incomes`) + akun dev default:

```bash
make migrate-seed
```

atau:

```bash
pnpm migrate:seed
```

Atau pakai dataset demo:

```bash
pnpm migrate:seed:demo
```

Jika memakai full compose stack, jalankan migrasi dalam container:

```bash
pnpm compose:migrate
```

### 4. Jalankan proses aplikasi dari host

Terminal 1:

```bash
pnpm dev:web
```

Terminal 2:

```bash
pnpm dev:api
```

Terminal 3:

```bash
pnpm dev:worker
```

Terminal 4:

```bash
pnpm dev:scheduler
```

### 5. Jalankan full compose stack

```bash
pnpm compose:up:full
```

Default frontend container expose ke:

- `http://localhost:8088`

Untuk mematikan full compose:

```bash
pnpm compose:down:full
```

### 6. Verifikasi

```bash
curl http://localhost:8080/health/ready
```

Mode hybrid:

- buka `http://localhost:5173`

Mode full compose:

- buka `http://localhost:8088`

### 7. Mematikan infra-only

Dengan Docker:

```bash
docker compose down
```

Dengan Podman:

```bash
podman compose down
```

Jika ingin menghapus volume data:

```bash
docker compose down -v
```

atau:

```bash
podman compose down -v
```

## Alur Development Harian

Urutan paling aman:

1. pastikan PostgreSQL dan Redis sudah hidup
2. jalankan `make migrate-up`
3. jalankan `pnpm dev:web`
4. jalankan `pnpm dev:api`
5. jalankan `pnpm dev:worker`
6. jalankan `pnpm dev:scheduler`
7. buka UI, register/login, lalu smoke test modul yang dikerjakan

## Command Penting

Root:

```bash
pnpm install --recursive
pnpm dev:web
pnpm dev:api
pnpm dev:worker
pnpm dev:scheduler
pnpm build:web
pnpm test:web:e2e:list
pnpm test:web:e2e
pnpm test:web:e2e:managed
pnpm build:api
pnpm migrate:up
pnpm migrate:seed:demo
pnpm compose:migrate
pnpm compose:up:full
pnpm compose:down:full
pnpm migrate:status
pnpm test:api
pnpm test:api:auth
pnpm test:api:webhooks
```

Makefile:

```bash
make up
make down
make migrate-up
make migrate-seed-demo
make migrate-status
make compose-migrate
make up-full
make down-full
make test-api
make test-api-auth-integration
make test-api-webhooks-integration
make test-web-e2e-managed
make test-web-e2e-list
```

## Testing

### Frontend

```bash
cd apps/web
pnpm lint
pnpm build
```

### Playwright E2E auth + MFA

Suite Playwright yang sudah dipasang sekarang mencakup:

- auth + MFA
- route smoke untuk backoffice utama
- CRUD lokal `users -> tokos -> banks`
- audit flow transaksi/topup/withdrawal shell

Prasyarat sebelum menjalankan:

1. jalankan `pnpm migrate:seed:demo`
2. jalankan `pnpm dev:api`
3. jalankan `pnpm dev:web`
4. install browser test sekali saja:

```bash
cd apps/web
pnpm e2e:install
```

Lihat daftar test yang terdeteksi:

```bash
pnpm test:web:e2e:list
```

Jalankan test:

```bash
pnpm test:web:e2e
```

Atau biarkan Playwright men-start `api` dan `web` sendiri:

```bash
pnpm test:web:e2e:managed
```

Jika butuh override runtime:

- `PLAYWRIGHT_BASE_URL`
- `E2E_DEV_USERNAME`
- `E2E_DEV_PASSWORD`
- `E2E_DEV_EMAIL`

### Backend

```bash
cd apps/api
go test ./...
go build ./...
```

### Seed integration test

Test ini memvalidasi dua profile seed:

- `development`
- `demo`

Jalankan:

```bash
cd apps/api
go test ./internal/seed -run 'Test(DevelopmentSeed|DemoSeed)$' -v
```

### DB integration test auth dan MFA

Test ini memvalidasi state yang paling sensitif untuk backoffice auth:

- login dengan username atau email
- user inactive ditolak
- setup MFA menyimpan secret dan recovery code dalam bentuk terenkripsi
- login challenge menerima TOTP valid
- recovery code hanya bisa dipakai sekali
- disable MFA benar-benar menghapus kolom MFA di database

Jalankan:

```bash
cd apps/api
go test ./internal/auth -run Integration -v
```

Atau dari root:

```bash
pnpm test:api:auth
```

Catatan:

- test ini memakai `embedded-postgres`
- PostgreSQL lokal tidak wajib
- run pertama akan mengunduh binary PostgreSQL embedded
- butuh internet dan writable temp directory saat run pertama

### DB integration test webhook finansial

Test ini memvalidasi side effect SQL penuh untuk:

- QRIS regular deposit
- QRIS NexusGGR topup
- disbursement success
- disbursement failure refund settle

Jalankan:

```bash
cd apps/api
go test ./internal/modules/webhooks -run Integration -v
```

Catatan:

- test ini memakai `embedded-postgres`
- PostgreSQL lokal tidak wajib untuk test ini
- run pertama akan mengunduh binary PostgreSQL embedded
- butuh internet dan writable temp directory saat run pertama

## Route Penting Saat Ini

Backoffice:

- `GET /backoffice/api/auth/bootstrap`
- `POST /backoffice/api/auth/login`
- `POST /backoffice/api/auth/register`
- `POST /backoffice/api/auth/logout`
- `POST /backoffice/api/auth/mfa/login/verify`
- `GET /backoffice/api/auth/mfa`
- `POST /backoffice/api/auth/mfa/setup`
- `POST /backoffice/api/auth/mfa/confirm`
- `POST /backoffice/api/auth/mfa/disable`
- `GET /backoffice/api/users`
- `GET /backoffice/api/tokos`
- `GET /backoffice/api/players`
- `GET /backoffice/api/transactions`
- `GET /backoffice/api/banks`
- `GET /backoffice/api/withdrawal/bootstrap`
- `GET /backoffice/api/call-management/bootstrap`

Public compatibility layer:

- `/api/v1/providers`
- `/api/v1/games`
- `/api/v1/games/v2`
- `/api/v1/user/create`
- `/api/v1/user/deposit`
- `/api/v1/user/withdraw`
- `/api/v1/user/withdraw-reset`
- `/api/v1/transfer/status`
- `/api/v1/game/launch`
- `/api/v1/money/info`
- `/api/v1/game/log`
- `/api/v1/call/*`
- `/api/v1/control/*`
- `/api/v1/generate`
- `/api/v1/check-status`
- `/api/v1/balance`
- `/api/v1/merchant-active`

Webhook:

- `POST /api/webhook/qris`
- `POST /api/webhook/disbursement`

## Troubleshooting

### `health/ready` gagal pada `postgres`

Periksa:

- database sudah dibuat
- `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`
- migrasi sudah dijalankan

### `health/ready` gagal pada `redis`

Periksa:

- Redis hidup di host/port yang benar
- `REDIS_PASSWORD` cocok
- `REDIS_DB` dan `REDIS_CACHE_DB` valid

### login/register berhasil tapi akses admin terbatas

Kemungkinan user masih role `user`. Promosikan ke `dev` atau `superadmin` via SQL.

### token/cookie backoffice tidak tersimpan

Periksa:

- frontend dibuka dari `http://localhost:5173`
- backend di `http://localhost:8080`
- `SESSION_DOMAIN=localhost`
- browser tidak memblokir cookie lokal

### migrasi gagal

Periksa:

- database kosong atau version table goose konsisten
- koneksi DB benar
- user DB punya hak create table/index

## Catatan Production

Sebelum production nyata:

- set `APP_ENV=production`
- set `APP_DEBUG=false`
- set `SESSION_SECURE_COOKIE=true`
- gunakan secret acak yang benar-benar baru
- pasang TLS di reverse proxy
- jalankan API, worker, scheduler sebagai service terpisah
- aktifkan observability exporter jika dipakai
- siapkan backup PostgreSQL dan Redis
- baca runbook cutover di [docs/production-cutover.md](docs/production-cutover.md)

## CI

CI dasar tersedia di:

- `.github/workflows/ci.yml`

Pipeline ini menjalankan:

- frontend lint
- frontend build
- backend test
- backend build

Repo ini sudah cukup untuk dev runtime nyata, demo seed, container packaging, dan parity verification yang lebih keras. Cutover production tetap harus melalui staging rehearsal dan reconciliation finansial terlebih dahulu.
