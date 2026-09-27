# Cloudflare: publish `posgo.xamltech.com`

Step-by-step for putting the POS.Go surface behind Cloudflare DNS and TLS. The
goal is a hostname that an Android till can trust without warnings, that serves
the store API, and that cannot be used to reach the control plane.

> **Status (2026-09-27).** The DNS record from step 2 is live (grey cloud, A
> only) and the origin serves the name: TLS terminates in **host nginx**, not in
> the `caddy` container — see `deployments/nginx/posgo.xamltech.conf`, which is
> the file installed on the VPS, and "The edge is host nginx" in
> `docs/11_OPERATIONS.md`. Steps 3 and 4 are decisions you still have to make in
> the Cloudflare dashboard (turning on the orange cloud changes them); step 5's
> Caddy commands do not apply to this host, use the nginx file instead.

- [1. Prerequisites](#1-prerequisites)
- [2. The DNS record](#2-the-dns-record)
- [3. TLS mode: pick one](#3-tls-mode-pick-one)
- [4. Cloudflare rules that break a POS client](#4-cloudflare-rules-that-break-a-pos-client)
- [5. Origin side: make the server serve the hostname](#5-origin-side-make-the-server-serve-the-hostname)
- [6. Verify end to end](#6-verify-end-to-end)
- [7. Ship the release build](#7-ship-the-release-build)
- [8. Rollback](#8-rollback)
- [9. Reference](#9-reference)

## 1. Prerequisites

| Item | Value |
|---|---|
| Origin IP | `197.44.6.42` |
| Hostname | `posgo.xamltech.com` |
| What it serves | POS.Go landing, `/selforder`, `/apk`, and the store API `/v1/*` (everything except `/v1/saas/*` and `/v1/platform/*`) |
| Edge TLS | Cloudflare (443), HTTP 80 redirected to HTTPS |
| Origin TLS | Caddy inside `pos-prod-caddy`, which obtains a certificate for all three hostnames automatically |
| Firewall | `ufw` must already allow 80 and 443 (it does) |

Confirm you can reach the box before touching DNS:

```bash
ssh -o BatchMode=yes -i ~/.ssh/id_ed25519 root@197.44.6.42 'docker ps --format "{{.Names}}\t{{.Status}}" | grep pos-prod'
```

You need in the Cloudflare account: the zone `xamltech.com`, and permission to
edit DNS and SSL/TLS settings. If the zone is on the Free plan, everything below
still works; only the WAF/Page-Rule features in step 4 are limited.

## 2. The DNS record

Cloudflare dashboard → **xamltech.com** → **DNS** → **Add record**:

| Field | Value |
|---|---|
| Type | `A` |
| Name | `posgo` |
| IPv4 address | `197.44.6.42` |
| Proxy status | see step 3 — start with **DNS only** |
| TTL | Auto |

Do **not** add an `AAAA` record. The VPS has no IPv6, and Cloudflare will hand
resolvers an unreachable AAAA, which stalls the app's first connection for
seconds before it retries over IPv4.

Sanity check once it resolves:

```bash
dig +short posgo.xamltech.com A
# 197.44.6.42   (grey cloud)  or  104.x / 172.x (orange cloud)
```

## 3. TLS mode: pick one

The choice is whether Cloudflare's edge is in the request path.

### Option A — proxied (orange cloud), Full (strict) — recommended

Cloudflare terminates the client connection and re-encrypts to the origin.
Keeps DDoS protection and hides the origin IP.

1. Edit the `posgo` record → **Proxy status: Proxied**.
2. **SSL/TLS** → **Overview** → set **Full (strict)**.

   Do **not** use "Flexible": it sends plaintext from the edge to the origin,
   which the app forbids anyway (`usesCleartextTraffic=false`), so requests
   would fail closed.
3. The origin must present a certificate Cloudflare trusts. Two ways:

   - **Keep Caddy's Let's Encrypt certificate (simplest).** LE is publicly
     trusted, so Full (strict) accepts it. Caddy obtains it during step 5 — and
     it can only do that while the record is **grey-cloud**, because the ACME
     challenge has to reach the origin directly. That is why step 2 starts grey.
   - **Or install a Cloudflare Origin CA certificate** for
     `*.xamltech.com` + `xamltech.com` in Cloudflare → **SSL/TLS** → **Origin
     Server**, then mount it into Caddy and drop `tls` auto-provisioning for
     that host. More moving parts; only worth it if you do not want to depend on
     LE rate limits.

4. Cloudflare → **SSL/TLS** → **Edge Certificates** → **Always Use HTTPS: On**.
   Then add a Redirect rule (or an equivalent Page Rule) that **skips**
   `http://*.xamltech.com/.well-known/acme-challenge/*`, otherwise the edge
   answers the ACME challenge with a 301 and Caddy can never renew.

   Scope that exclusion to all three hostnames, not just this one: Caddy
   provisions `xamltech.com` and `api.xamltech.com` from the same Caddyfile, and
   their renewals break under the same rule.

### Option B — DNS only (grey cloud)

Cloudflare is used purely as authoritative DNS. The Caddy certificate is
presented straight to the phone, so there is no edge TLS mode to get wrong, and
nothing about Cloudflare's rules can break the client.

The cost: the origin IP is public and there is no DDoS protection in front of
the login endpoint.

Pick B if the account's SSL settings are managed by someone else, or if you want
the shortest possible path to a working hostname. Pick A otherwise.

## 4. Cloudflare rules that break a POS client

The app is a native Android client, not a browser. Several Cloudflare features
that are invisible to a browser will break it. Before going to production,
confirm all of these:

| Setting | Where | Requirement |
|---|---|---|
| Bot Fight Mode / Super Bot Fight Mode | Security → Bots | **Off**, or a WAF skip rule for `/v1/*`. The app has no JS challenge client and every request would get a 403. |
| Managed WAF rules | Security → WAF | Either off, or a skip rule for `posgo.xamltech.com/v1/*`. The app's paths (`/v1/variants/:id`, `/v1/users/:id/pin`) can trip generic rules. |
| Rate limiting on `/v1/auth/login` | Security → Rate limiting | **Do not add one.** The API already rate-limits login per IP (`LoginRateLimit`). A second, stricter limit in front of it will lock merchants out. |
| Under Attack Mode / Security Level | Security → Settings | Off during rollout. It challenges every request. |
| Browser Integrity Check | Security → Settings | Off, or skipped for `/v1/*`. |
| Cache Rules | Cache → Rules | Nothing under `/v1/*` may be cached. The API sends `no-store`; do not override it. |
| Rocket Loader / Auto Minify / Email Obfuscation | Speed → Optimization | Off. They rewrite HTML only, but they also rewrite responses on some paths. |
| IPv6 Compatibility | Network | Fine to leave on, since the record is A-only. |

**Per-IP rate limiting and `X-Forwarded-For`.** The API derives the client IP
from `X-Forwarded-For` and counts login failures per IP. Cloudflare *appends*
the real client IP to any inbound `X-Forwarded-For` it receives, and Gin's
default trusted-proxy configuration accepts the **leftmost** entry. So a client
that sends its own `X-Forwarded-For: 203.0.113.9` can rotate its apparent
identity and defeat the login limiter — with or without Cloudflare in front.

**This is already fixed in the app**, in `server.Register`, so it covers the
API, the OpenAPI generator and the tests with one policy
(`internal/config/clientip.go`, applied by
`httptransport.ConfigureTrustedProxies`):

```
trusted: 127.0.0.1/32, ::1/128, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
       + Cloudflare's published edge ranges (ips-v4 + ips-v6)
```

Gin then walks `X-Forwarded-For` right-to-left and returns the first **untrusted**
hop: the Docker-network Caddy hop and the Cloudflare edge are skipped, and
everything to the left of them — the part a client controls — is discarded. The
applied list is logged at startup (`client IP trust configured`) and counted on
`/metrics`-bound logs, so you can confirm it after a deploy.

Cloudflare's ranges are only needed because this hostname is orange-clouded. If
you stay grey-cloud, the header is just `<client>` appended by Caddy and the
edge ranges are harmless. Override the whole list with `TRUSTED_PROXIES`
(comma-separated CIDRs) if you add a proxy in front of Caddy; `0.0.0.0/0` and
`::/0` are rejected at startup because they hand the client address back to
whoever sent the header.

Regression coverage: `internal/transport/http/clientip_test.go` proves a
spoofed leftmost entry is ignored and that four spoofed identities cannot
bypass a three-attempt login limit, plus
`internal/transport/server/router_test.go` for the wiring. Before this landed,
treat the login limiter as advisory only.

## 5. Origin side: make the server serve the hostname

Do this **while the record is still grey-cloud**, so the edge can solve the ACME
challenge.

**On the production VPS the edge is host nginx** (only `postgres`, `redis`,
`migrate` and `api` run there), so the Caddy steps below do not apply. Use the
committed vhost instead:

```bash
ssh -o BatchMode=yes -i ~/.ssh/id_ed25519 root@197.44.6.42
cd /opt/pos/Backend/go-pos-backend-implementation-ready

install -m 644 deployments/nginx/posgo.xamltech.conf \
    /etc/nginx/sites-available/posgo.xamltech.conf
ln -sfn /etc/nginx/sites-available/posgo.xamltech.conf \
    /etc/nginx/sites-enabled/posgo.xamltech.conf
```

Order matters, because `nginx -t` fails while the certificate files are absent:

1. Enable only the port-80 block (ACME path + 301 to https), then
   `nginx -t && systemctl reload nginx`.
2. `certbot certonly --nginx -d posgo.xamltech.com`.
3. Enable the 443 block and reload again.
4. `ls -la /var/www/apk/pos_go.apk` — the APK the landing links to has to be
   physically present.
5. `certbot renew --cert-name posgo.xamltech.com --dry-run` to prove renewal.

The API reads these keys from `.env.prod`:

```bash
grep -E '^(COMPANY_DOMAIN|SAAS_DOMAIN|POS_DOMAIN)=' .env.prod
# COMPANY_DOMAIN=xamltech.com
# SAAS_DOMAIN=api.xamltech.com
# POS_DOMAIN=posgo.xamltech.com
```

They are currently at their compose defaults and surface routing is **off**
(`SURFACE_ROUTING_ENABLED=false`), so the host serves the same routes as
`api.xamltech.com`. That is enough for the Android client; turning routing on
(`/v1/saas/*`, `/v1/platform/*` and `/admin/` move off this hostname) is a
separate change with its own check — see step 3 of this document and
`docs/11_OPERATIONS.md` → "Three public domains".

If you do run the containerised edge instead (`deployments/caddy/Caddyfile`):

```bash
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml \
  up -d --build caddy api
```

`caddy_data` persists the certificates, so this only has to happen once.
Caddy obtains certificates for all three hostnames on start; a failure here
means port 80 is blocked or the record still resolves to Cloudflare.

## 6. Verify end to end

Run every command; they check different layers.

```bash
# a. DNS
dig +short posgo.xamltech.com A

# b. edge TLS and the certificate chain the phone will see
curl -sSI https://posgo.xamltech.com/health/live | head -20
openssl s_client -connect posgo.xamltech.com:443 -servername posgo.xamltech.com </dev/null 2>/dev/null \
  | openssl x509 -noout -subject -issuer -dates

# c. the POS surface answers
curl -s https://posgo.xamltech.com/health/live
curl -s https://posgo.xamltech.com/v1/meta/countries

# d. the control plane is NOT here (expect 404 not_available_on_host)
curl -s -o /dev/null -w '%{http_code}\n' https://posgo.xamltech.com/v1/saas/summary
curl -s -o /dev/null -w '%{http_code}\n' https://posgo.xamltech.com/admin/

# e. an unknown Host is refused by the app, not by the edge.
#    Do this on the VPS, straight at the container, where the app sees the
#    header: the Cloudflare edge will never route an unknown Host to this
#    origin, so testing through the public name proves nothing.
ssh -o BatchMode=yes -i ~/.ssh/id_ed25519 root@197.44.6.42 \
  'curl -s -H "Host: posgo.example.org" -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/health/live'
# expect 404 (unknown_host)

# f. a real sign-in round trip against this hostname
curl -s -X POST https://posgo.xamltech.com/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"tenant_id":"<store>","email":"<user>","password":"<pass>","device_id":"cf-check","device_name":"cf-check"}'

# g. the install page and the APK the landing page links to
curl -s https://posgo.xamltech.com/ | grep -o '/apk/[^"]*'
curl -s -o /dev/null -w '%{http_code} %{content_type} %{size_download}\n' \
  https://posgo.xamltech.com/apk/pos_go.apk
```

Expected: (b) a certificate whose subject or SAN covers `posgo.xamltech.com` and
whose issuer is either Cloudflare or a publicly trusted CA — never
"self-signed"; (c) `200`; (d) `404`; (f) `200` with a `{"data":...}` envelope
for store staff, or `403 wrong_surface` if you used a `saas_admin` account,
which is the correct answer on this host.

From a real phone, with the release APK installed, confirm sign-in, one sale,
one receipt preview, and one offline sale replayed after the network returns.

## 7. Ship the release build

The store build already points at this hostname. Rebuild only if you changed
the domain:

```bash
cd Flutter/pos_go_app
flutter build appbundle --release \
  --dart-define=API_BASE_URL=https://posgo.xamltech.com \
  --dart-define=SAAS_API_BASE_URL=https://api.xamltech.com
```

Artefacts, screenshots and the upload order are in
`Flutter/pos_go_app/play_store/README.md`.

Note that ESC/POS receipt printers are reached over the **local network** on
port 9100. Cloudflare does not proxy 9100 and never will; a printer configured
with a public address will stop working the moment the cloud is enabled.

## 8. Rollback

If the edge breaks sign-in after you flip to the orange cloud:

1. Set the record back to **DNS only**. The phone then talks straight to Caddy
   and everything that worked grey-cloud works again.
2. If you want to keep the orange cloud, drop **SSL/TLS** → **Overview** to
   **Full** (not strict) while the origin certificate is sorted out.
3. If `/v1/*` is being challenged, turn off Bot Fight Mode first — it is the
   cause nine times out of ten.
4. Origin state is untouched by any of this; `docker compose ps` and
   `docker logs pos-prod-caddy` still tell you whether the server is fine.

## 9. Reference

- Surface split, error codes (`not_available_on_host`, `wrong_surface`,
  `unknown_host`): `docs/11_OPERATIONS.md` → "Three public domains".
- Caddy vhosts: `deployments/caddy/Caddyfile` (`POS_DOMAIN`, `SAAS_DOMAIN`,
  `COMPANY_DOMAIN` from `.env.prod`).
- nginx equivalent, if the host does not use the containerized edge:
  `deployments/nginx/xamltech_surfaces.conf.example`.
- One-shot provisioning: `scripts/provision_vps.sh` (env-var switches for the
  three domains, `SKIP_*` stages).
- Live release runbook and its DNS section: `docs/20_DELIVERY_PLAN.md`.
