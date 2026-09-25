# Static assets served by Caddy

Mounted read-only at `/srv/web` by `docker-compose.prod.yml`.

- `admin/` — the built React console (`web_admin/`), published on the SaaS
  domain under `/admin/`. Build it with `npm run build` and copy
  `web_admin/dist/` here, or publish it from the host web root.
- `apk/` — the POS.Go Android package, published on the POS domain under `/apk/`.

Both locations are optional: when a directory is missing the request is proxied
to the Go app, which answers 404 for paths it does not serve.
