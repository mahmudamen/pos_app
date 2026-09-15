# POS Go — Delivery Plan & Operations Runbook

Status: **v1.0 deployed and verified green on production.** DNS live, trusted
Let's Encrypt cert installed + auto-renewed.

## What is live on the VPS (197.44.6.42)

| Piece            | State |
|------------------|-------|
| Stack            | Docker Compose `pos-prod-*` (api, postgres:16-alpine, redis:7-alpine), all `unless-stopped` |
| API              | `pos-api:latest` bound to **127.0.0.1:8080** (no external bypass) |
| HTTPS            | nginx reverse proxy on :443 with trusted **Let's Encrypt** cert (auto-renew) + HTTP→HTTPS 301 |
| Database         | migrate applied to v26 (`store_emails` added, `pos_app_rls` granted INSERT/SELECT) |
| Firewall (ufw)   | default-deny; open **22, 80, 443**, Odoo 8070, xrdp 3389 (see pending) |
| SSH brute force  | fail2ban active (5 tries / 10 min → 1h ban), key auth working for root |
| Backups          | none yet — see pending |

## Verified green (production, 2026-09-15)

- On-device network E2E via emulator → SSH tunnel → prod: login → browse →
  sell → receipt → logout, **and** onboarding signup → trial store → catalog
  (10 products, 13-digit EAN barcodes, images, descriptions) → relogin → sale.
- `POST /v1/auth/register`: 201 (new trial tenant, plan `trial`, 15-day
  `trial_ends_at`); duplicate email → **409 `email_taken`**; unsupported
  business type → 400.
- Catalog: categories (Drinks/Food/Pastries), products carry `description`,
  `image_url`, valid EAN-13.
- Demo tenant (`demo-restaurant`) unaffected.

## Release notes

See `docs/19_RELEASE_NOTES.md`.

## ~~Pending — requires YOU (DNS)~~ DONE

1. ~~Add DNS `A` record: `api.xamltech.com → 197.44.6.42`~~ ✅ Done (Cloudflare)
2. ~~`certbot --nginx -d api.xamltech.com`~~ ✅ Done — Let's Encrypt cert
   installed, auto-renewal configured; HSTS max-age set to 2 years by the Go
   SecurityHeaders middleware; nginx adds no duplicate security headers.
3. (Recommended) Stop root password login — add your personal SSH public key to
   `/root/.ssh/authorized_keys`, then:
   ```
   sed -i 's/^PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
   echo "PasswordAuthentication no" > /etc/ssh/sshd_config.d/99-key-only.conf
   systemctl restart ssh
   ```
   Do **not** run this until your own key is installed (you currently connect
   by password). fail2ban already defends against brute force meanwhile.

## Pending — ops (I can do when asked)

- **Backups**: `pg_dump` of the `pos` docker volume to `/backups` with retention
  (mirror `scripts/backup.sh` from the repo) + cron + verified restore.
- **xrdp (3389)**: open to Anywhere — restrict to your office IP or close it.
- **Odoo 8070**: confirm it is still needed publicly; consider moving behind
  nginx with auth.
- **Samba 139/445**: listening but ufw-blocked; stop and disable the `smbd`
  services if unused.
- **Docker watchtower / image rebuilds**: pin images and rebuild via the
  compose file on rollout (already digest-pinned in `docker-compose.prod.yml`).

## Rollout procedure (future releases)

From the dev checkout:
```
# 1. run local checks
make check                        # fmt + vet + test
flutter analyze && flutter test    # client

# 2. sync code + migrations to the VPS
rsync -az --delete --exclude='.env*' --exclude='bin/' --exclude='.git' \
  ./ Backend/... root@197.44.6.42:/opt/pos/...
# 3. deploy
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml \
  build migrate api
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml \
  run --rm migrate
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml \
  up -d api
# 4. verify
curl -s https://api.xamltech.com/health/live
```