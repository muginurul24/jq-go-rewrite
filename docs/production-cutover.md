# Production Cutover

Dokumen ini adalah runbook cutover minimal untuk menggantikan legacy `../justqiuv2` dengan rewrite React + Go.

## Prinsip

- Jangan cutover tanpa backup final PostgreSQL dan snapshot Redis.
- Jangan cutover tanpa health check, smoke test, dan reconciliation snapshot sebelum dan sesudah switch.
- Jangan pakai akun MFA-enabled untuk smoke automation.

## Preflight

1. Sinkronkan `.env` target production dan pastikan semua secret final sudah terisi.
2. Tentukan mode migrasi pada environment target:

Jika target adalah database rewrite baru yang masih kosong:

```bash
pnpm migrate:up
```

Jika target adalah database production legacy yang tabel intinya sudah ada dan ingin dipakai langsung oleh rewrite:

```bash
pnpm migrate:baseline
```

Jangan gunakan `migrate:seed` pada production.

3. Verifikasi service dasar:

```bash
./scripts/smoke-runtime.sh
```

Jika preflight dijalankan terhadap clone/staging tertentu, kirim override env langsung dari shell agar tidak memakai akun atau database demo:

```bash
DB_DATABASE=justqiuv2_prod_clone \
SMOKE_USERNAME=localproddev \
SMOKE_PASSWORD=justqiu8 \
SMOKE_TOKO_TOKEN='<token hasil regenerate toko>' \
pnpm preflight:staging
```

4. Ambil baseline finansial sebelum switch:

```bash
./scripts/reconcile-finance.sh
```

Simpan output ini sebagai snapshot `pre-cutover`.

## Cutover Window

1. Freeze perubahan administratif di legacy.
2. Backup PostgreSQL legacy.
3. Backup/snapshot Redis legacy jika masih dipakai untuk session/queue.
4. Pastikan API rewrite healthy dan bisa diakses dari host target:

```bash
curl -fsS https://api.bola788.store/health/live
curl -fsS https://api.bola788.store/backoffice/api/auth/bootstrap
```

5. Jika legacy dan rewrite memakai database production yang sama, matikan job legacy sebelum menyalakan job rewrite:

```bash
sudo supervisorctl stop laravel-worker:*
sudo supervisorctl stop laravel-schedule
```

6. Nyalakan worker dan scheduler rewrite:

```bash
sudo supervisorctl start justqiuv2-rewrite-worker
sudo supervisorctl start justqiuv2-rewrite-scheduler
sudo supervisorctl status justqiuv2-rewrite-worker justqiuv2-rewrite-scheduler
```

7. Switch callback provider QRIS/disbursement ke host rewrite.
8. Switch traffic backoffice dan public API ke stack rewrite.

Catatan:

- Jangan menyalakan `justqiuv2-rewrite-worker` dan `justqiuv2-rewrite-scheduler` selama `laravel-worker` dan `laravel-schedule` legacy masih aktif pada database yang sama.
- Pada deployment VPS saat ini, API rewrite boleh hidup side-by-side, tetapi worker dan scheduler rewrite seharusnya tetap `STOPPED` sampai cutover window.

## Immediate Verification

Setelah traffic diarahkan:

```bash
./scripts/smoke-runtime.sh
./scripts/reconcile-finance.sh
```

Bandingkan snapshot `post-cutover` dengan `pre-cutover`.

## Fokus Reconciliation

Periksa khusus:

- total `balances.pending`
- total `balances.settle`
- total `balances.nexusggr`
- jumlah transaksi per `category/type/status`
- `incomes.amount`
- jumlah `pending` yang lebih tua dari 30 menit
- toko tanpa row `balances`

## Rollback Trigger

Rollback ke legacy bila terjadi salah satu kondisi berikut:

- health check rewrite gagal
- webhook callback gagal diproses konsisten
- smoke runtime gagal
- selisih finansial tidak bisa dijelaskan
- auth session/operator login gagal secara sistemik

## Rollback

1. Arahkan traffic kembali ke legacy.
2. Kembalikan callback provider ke legacy.
3. Restore backup bila terjadi mutasi yang tidak dapat direkonsiliasi.
4. Simpan log dan snapshot DB rewrite untuk postmortem.
