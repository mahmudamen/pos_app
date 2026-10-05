-- Reset the demo/testing tenants so a fresh testing phase can be seeded.
--
-- SCOPE IS DELIBIBERATE: this only removes
--   * the 5 `demo-*` tenants (rebuilt by scripts/seed_demo.sql), and
--   * the throwaway `*probe*` / `*walk*` tenants created while testing.
-- Real customer tenants are never touched. Nothing here is tenant-scoped by
-- RLS, so the script runs as the database owner:
--
--   psql "$DATABASE_URL" -f scripts/demo_testing_reset.sql
--
-- Idempotent, and safe to run inside a transaction that is rolled back for a
-- dry run:  psql "$DATABASE_URL" -1 -f scripts/demo_testing_reset.sql
--            (append a ROLLBACK to the last line to preview the deletions)

BEGIN;

CREATE TEMP TABLE _demo_reset_victims ON COMMIT DROP AS
SELECT id, slug FROM tenants
WHERE slug LIKE 'demo-%'
   OR slug ILIKE '%probe%'
   OR slug ILIKE '%walk%'
   OR name ILIKE '%probe%'
   OR name ILIKE '%walk%';

-- Child tables whose FK to tenants is NO ACTION (verified against
-- pg_constraint on production: floors, product_lots, product_variants and
-- restaurant_tables), so they must be emptied first or the tenant delete is
-- refused. restaurant_tables goes first in case its floor_id FK is not
-- cascading either.
DELETE FROM restaurant_tables
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

DELETE FROM floors
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

DELETE FROM product_variants
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

DELETE FROM product_lots
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

-- FKs declared SET NULL are cleared explicitly as well, so this script does
-- not silently depend on the ON DELETE action staying as written.
DELETE FROM community_invitations
 WHERE inviter_id IN (SELECT id FROM users WHERE tenant_id IN (SELECT id FROM _demo_reset_victims));

DELETE FROM community_members
 WHERE user_id IN (SELECT id FROM users WHERE tenant_id IN (SELECT id FROM _demo_reset_victims));

DELETE FROM installations
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

DELETE FROM registration_risk_events
 WHERE tenant_id IN (SELECT id FROM _demo_reset_victims);

DELETE FROM tenants WHERE id IN (SELECT id FROM _demo_reset_victims);

DO $$
DECLARE
  n BIGINT;
BEGIN
  SELECT count(*) INTO n FROM _demo_reset_victims;
  RAISE NOTICE 'demo testing reset: % tenant(s) removed', n;
END;
$$;

COMMIT;