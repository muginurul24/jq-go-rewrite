# VPS Deployment Notes

Dokumen ini mencatat state deploy side-by-side rewrite pada VPS target.

## Path

- rewrite release: `/home/mugiew/projects/justqiuv2-rewrite-go`
- legacy app: `/home/mugiew/projects/livewire-filament-setup`

## Public Hostnames

- legacy backoffice: `https://dash.bola788.store`
- rewrite candidate: `https://api.bola788.store`

## Local Ports on VPS

- rewrite API Go: `127.0.0.1:8080`
- rewrite Nginx entry: `127.0.0.1:8088`
- legacy Octane: `127.0.0.1:8000`

## Service Layout

Supervisor:

- legacy worker: `laravel-worker:*`
- legacy scheduler: `laravel-schedule`
- legacy Octane: `laravel-octane:*`
- rewrite API: `justqiuv2-rewrite-api`
- rewrite worker: `justqiuv2-rewrite-worker`
- rewrite scheduler: `justqiuv2-rewrite-scheduler`

Cloudflared:

- legacy tunnel service: `cloudflared.service`
- rewrite tunnel service: `cloudflared-bola788-api.service`

Nginx:

- conf: `/etc/nginx/conf.d/api.bola788.store.conf`
- static root: `/var/www/justqiuv2-rewrite`

## Safe Current State

Saat legacy dan rewrite masih memakai database production yang sama:

- `justqiuv2-rewrite-api` boleh `RUNNING`
- `justqiuv2-rewrite-worker` harus `STOPPED`
- `justqiuv2-rewrite-scheduler` harus `STOPPED`
- `laravel-worker:*` tetap `RUNNING`
- `laravel-schedule` tetap `RUNNING`

## Cutover Command Sequence

Matikan job legacy:

```bash
sudo supervisorctl stop laravel-worker:*
sudo supervisorctl stop laravel-schedule
```

Nyalakan job rewrite:

```bash
sudo supervisorctl start justqiuv2-rewrite-worker
sudo supervisorctl start justqiuv2-rewrite-scheduler
sudo supervisorctl status justqiuv2-rewrite-api justqiuv2-rewrite-worker justqiuv2-rewrite-scheduler
```

Rollback cepat:

```bash
sudo supervisorctl stop justqiuv2-rewrite-worker
sudo supervisorctl stop justqiuv2-rewrite-scheduler
sudo supervisorctl start laravel-worker:*
sudo supervisorctl start laravel-schedule
```

## Health Checks

```bash
curl -fsS https://api.bola788.store/health/live
curl -fsS https://api.bola788.store/backoffice/api/auth/bootstrap
curl -fsS https://dash.bola788.store
```
