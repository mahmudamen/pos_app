# 27 — Main-site migration: Odoo → Go (xamltech.com)

Replaces the Odoo-powered marketing site on `xamltech.com` with a Go/Gin
bilingual site built from the POS.Go landing pattern. Odoo addons + the
`ZXCV` database are archived in the handover at `/opt/handover/` (pulled to
`odoo-handover/` in the workspace root).

Status: **all-but-DNS done, tested locally.** The Go site is live and healthy
on the VPS at `127.0.0.1:8069` (systemd `xamltech-site`). Odoo +
`hr.xamltech.com` are already **down** (stopped+disabled, vhost disconnected).
The full `xamltech.com.conf` vhost is **enabled** in nginx and verified
end-to-end through TLS with a **self-signed placeholder cert** —
`https://xamltech.com` returns 200 for `/`, `/ar`, `/pricing`, etc., with
HSTS + Referrer-Policy from nginx. The only remaining steps need the
Cloudflare origin re-point (your dashboard — no Cloudflare API creds): rotate
the A records, then `certbot certonly --webroot` + swap the two
`ssl_certificate*` lines in `xamltech.com.conf` to the LE paths and reload.

---

## What is built

- `internal/mainsite/` — the site package (see module map below).
- `cmd/site/main.go` — standalone server binary, `SITE_ADDR` env (default
  `0.0.0.0:8069`, **overridden to `127.0.0.1:8069` in production**), adds
  `/health/live`, graceful shutdown, GIN_MODE=release.
- Separate from `internal/transport/server` (the POS API router) — do NOT put
  the site there; it runs as its own process and does not touch the API.

### Module map

| File | Purpose |
|------|---------|
| `i18n.go` | `Lang`/`EN`/`AR`, `Dir()`, `Other()`, `bi{En,Ar}` + `L(l)` + `more(en,ar)` |
| `brand.go` | brand constants, `wa()`/`mailto()` link helpers (WhatsApp/mailto) |
| `model.go` | page/section/plan/card types, `nav()` active keys, `urlEncode` |
| `content.go` | closing/final band, home, about, solutions hub, ERP, IT infra, web |
| `content2.go` | POS, agriculture, AI, addons (8), pricing (8 + FAQ), portfolio, testimonials, insights + 2 guides, knowledge, networktool, contact, demo, apk, privacy, terms, `registry()` |
| `css.go` | full inline stylesheet `siteCSS` (brand palette, RTL-aware logical props) |
| `templates.go` | `siteTemplates` layout + partials (`hero`, `sec` dispatcher, plan/card/faq/CTA, header, footer) |
| `handlers.go` | `viewData`, template funcs (`t`, `langURL`, `dict`, `moreText`), `siteTmpl`, `orgJSON()`, `buildView`, `Handler()`, `renderNotFound`, `Register`, `robotsTxt`, `sitemapXML`, legacy-redirect map |
| `assets.go` | embedded `faviconSVG` |

### Content model

Every bilingual value is a co-located `bi{En,Ar}` struct field (single source
of truth per page/section) rendered via the `t` template func. Sections are
data-driven (`sectionKind`: features/cards/steps/plans/stats/split/prose/faq/cta),
so adding a page = adding a `page` + `section` in `content2.go`.

### Route set (EN at `/`, AR at `/ar/*`, hreflang alternates)

`/`, `/about`, `/solutions`, `/solutions/erp`, `/solutions/it-infrastructure`,
`/solutions/web`, `/odoo-pos-egypt`, `/erp-agriculture-egypt`, `/ai-solutions`,
`/addons-store`, `/pricing`, `/portfolio`, `/testimonials`, `/insights`,
`/insights/ai-marketing-copilot`, `/insights/ai-pos-copilot`, `/knowledge`,
`/networktool`, `/contact-us`, `/request-demo`, `/apk-download`,
`/privacy-policy`, `/terms`.

Legacy 301s from: `/contactus`, `/contact`, `/private`, `/privacy`, `/store`,
`/shop`, `/blog`.

### Addon catalogue (bilingual, shared with pricing page)

POS Security €99 · POS Customization Kit €89 · POS Cutting Orders €149
(best seller, 4.9) · POS Stock Real-Time €129 (top rated, 4.8) · HDA
Collection €249 · Advanced Agriculture Free · AI Marketing Copilot €79 · AI
POS Copilot €59. Pricing page also shows USD equivalents ($99/$89/$149/$129/
$249/$79/$59).

---

## Deployed to production (staged)

- Binary: `/opt/xamltech-site/xamltech-site` (linux/amd64, static, v1.0.0).
- systemd: `xamltech-site.service` — `User=www-data`, `EnvironmentFile=/etc/xamltech-site.env`
  (`SITE_ADDR=127.0.0.1:8069`), hardening (`NoNewPrivileges`, `ProtectSystem=strict`,
  `ProtectHome`, `PrivateTmp`), `Restart=on-failure`. Enabled + active.
- nginx: `xamltech.com.conf` **enabled** — ACME webroot on port 80 and the
  443 block proxying to the Go site. TLS currently uses a **self-signed
  placeholder** (`/etc/nginx/ssl/xamltech.com/{cert,key}.pem`); swap to the
  Let's Encrypt paths after certbot (see runbook). HSTS + Referrer-Policy come
  from nginx; `X-Content-Type-Options` comes only from the Go app (deduped).
- Odoo: `odoo.service` stopped + disabled; `hr.Xamltech.conf` moved to
  `sites-available/hr.Xamltech.conf.disabled`. Port `8069` is now held by the
  Go site; `hr.xamltech.com` is down (accepted scope).
- Verified live on the box: `/` 200 (30 KB), `/ar` 200 (32 KB), `/health/live` 200,
  unknown path 404, `robots.txt` + `sitemap.xml` + hreflang alternates correct,
  `dir="rtl"` + Arabic copy + WhatsApp float on the Arabic home, no template
  artifacts.

---

## Cutover runbook

### 1. Re-point the Cloudflare origin (you — needs dashboard)

`xamltech.com` and `www.xamltech.com` currently resolve to Cloudflare edge
IPs (not this box). In the Cloudflare dashboard:

- Point both the apex and `www` records for `xamltech.com` at **197.44.6.42**
  (the same VPS already serving `api.xamltech.com` / `posgo.xamltech.com`).
- Leave them orange-cloud (proxied) so Cloudflare keeps its edge TLS and the
  origin box stays hidden. `hr.xamltech.com` is a different hostname and is
  unaffected by this switch.
- If you have no Cloudflare API creds, this is a manual dashboard change — I
  cannot do it for you.

### 2. Issue the Let's Encrypt cert (me — after step 1 propagates)

The vhost is already enabled with a **self-signed placeholder cert** so the
whole chain is testable pre-cutover. Once the domain resolves here, issue the
real cert against the ACME webroot already served on port 80:

```bash
certbot certonly --webroot -w /var/www/certbot \
  -d xamltech.com -d www.xamltech.com --non-interactive --agree-tos \
  -m ops@xamltech.com
```

Note: certbot fails for a domain with no DNS record — if `www.xamltech.com`
has no record yet (checked 2026-10-10: none), either add it in Cloudflare
first or drop `-d www.xamltech.com` from this command.

### 3. Swap the cert in nginx + verify (me)

```bash
sed -i "s|/etc/nginx/ssl/xamltech.com/cert.pem|/etc/letsencrypt/live/xamltech.com/fullchain.pem|" /etc/nginx/sites-available/xamltech.com.conf
sed -i "s|/etc/nginx/ssl/xamltech.com/key.pem|/etc/letsencrypt/live/xamltech.com/privkey.pem|" /etc/nginx/sites-available/xamltech.com.conf
nginx -t && systemctl reload nginx
```

The ACME webroot location can stay (harmless), or `rm` it later.

Verify from outside (any machine with internet):

```bash
curl -s -o /dev/null -w "%{http_code}\n" https://xamltech.com/          # 200
curl -s -o /dev/null -w "%{http_code}\n" https://xamltech.com/ar        # 200
curl -s -o /dev/null -w "%{http_code}\n" https://www.xamltech.com/      # 200
curl -sI https://xamltech.com/ | grep -iE "strict-transport|content-type"  # HSTS from nginx
```

### 4. Stop Odoo — DONE

Odoo has already been stopped, disabled, and its `hr.xamltech.com` vhost
disconnected (freeing `8069`, which the Go site now binds). `hr.xamltech.com`
is down, as intended. For reference, the commands used were:

```bash
systemctl stop odoo && systemctl disable odoo          # frees 0.0.0.0:8069
mv /etc/nginx/sites-enabled/hr.Xamltech.conf /etc/nginx/sites-available/hr.Xamltech.conf.disabled
nginx -t && systemctl reload nginx
```

To undo: `systemctl enable --now odoo` + restore the hr vhost symlink + reload
nginx. (Moving the site to `8069` was optional; nginx proxies to whatever port
the site binds, so `8071` or `8069` are equivalent.)

### 5. Rollback

Any step before (but not including) step 4 is non-destructive:

- Re-point Cloudflare back to the old origin.
- `rm /etc/nginx/sites-enabled/xamltech.com.conf && systemctl reload nginx`.
- `systemctl disable --now xamltech-site` (site down, Odoo untouched).

After step 4, rollback also needs `systemctl enable --now odoo` + restore the
hr vhost symlink + reload nginx.

---

## Notes / gotchas

- The site is a **separate process** from `pos-api`; do not add it to
  `docker-compose.prod.yml` or `server.Register`. It is a plain Go binary
  under systemd.
- `SITE_ADDR` in `/etc/xamltech-site.env` controls the bind. It is
  `127.0.0.1:8069` (the slot Odoo used to hold). Change it and
  `systemctl restart xamltech-site` if a different port is ever needed.
- No external assets: all CSS is inline, fonts are system stacks, favicon is
  embedded SVG. Zero third-party requests, so no CSP/CDN surprises.
- Arabic is the default `Lang`; English is at `/` and Arabic at `/ar`. The
  language switch uses logical CSS properties so RTL flows correctly.
- Demo credentials shown on the site (`demo` / `demo` @
  `demo18.xamltech.com`) are read-only marketing content — do not treat as
  secrets.
