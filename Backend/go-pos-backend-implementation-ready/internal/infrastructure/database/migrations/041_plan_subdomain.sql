-- +goose Up
-- Store subdomains as a plan capability: every plan (including the trial)
-- carries the `pos.subdomain` feature so `{slug}.xamltech.com` is a listed,
-- priced-in perk across the whole catalog. Wildcard DNS + Host-header routing
-- make the subdomain live for all of them the moment the slug exists; the
-- feature flag exists so the plan catalog and the SaaS control panel can
-- advertise it (and it can be toggled/gated later without a schema change).
--
-- Dedupe-guarded: re-running against a catalog that already lists the feature
-- is a no-op. Plain UPDATE (goose cannot parse DO $$...$$ blocks).
UPDATE plans
SET features = features || '["pos.subdomain"]'::jsonb,
    updated_at = now()
WHERE NOT features @> '["pos.subdomain"]'::jsonb;

-- +goose Down
UPDATE plans
SET features = features - 'pos.subdomain',
    updated_at = now()
WHERE features @> '["pos.subdomain"]'::jsonb;