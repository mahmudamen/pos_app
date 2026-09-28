-- +goose Up
-- Resolve the dangling `tenants.plan` default.
--
-- Background, and this is a real defect rather than tidying:
--
--   `tenants.plan` has defaulted to the literal 'standard' since migration
--   016, but no row with code 'standard' was ever inserted into `plans`.
--   Every tenant created without an explicit plan therefore holds a plan code
--   that resolves to nothing. 3,187 rows in the dev database are in that state.
--
-- That was harmless while `plans.features` was display-only, because nothing
-- ever joined on it. It stopped being harmless in the same release that made
-- features enforceable: `/v1/purchases/ocr` resolves the tenant's plan with
-- `LEFT JOIN plans p ON p.code = t.plan` and fails closed, so every
-- 'standard' tenant — including every existing customer — would have been
-- refused invoice capture with 402 plan_limit_exceeded, and would have had no
-- way to tell that the cause was a missing catalog row rather than a genuine
-- entitlement gap.
--
-- The fix is a catalog row, not a special case in the handler. Two properties
-- matter:
--
--   * is_active = FALSE, so the public pricing page (which selects
--     `FROM plans WHERE is_active`) is unchanged. 'standard' is a legacy
--     default, not something on sale. If it is ever meant to be sold, flipping
--     is_active is the whole change.
--   * the feature set mirrors `starter`, which is the neutral choice. These
--     tenants are not known to have paid for advanced inventory, restaurant or
--     loyalty, so granting those on the strength of a dangling string would be
--     inventing an entitlement. `ocr_capture` is included because migration
--     049 grants it to every plan by design: the metering windows are the
--     control surface for that feature, not the plan row.
--
-- 'test' gets the same treatment for the same reason: it is written by
-- testutil.SeedTenant's sibling paths and also resolves to nothing, so an
-- integration test would be refused a feature it is entitled to.
--
-- Dedupe-guarded, so a re-run is a no-op. Plain INSERT ... WHERE NOT EXISTS
-- (goose cannot parse DO $$...$$ blocks).
INSERT INTO plans (code, name, description, price_minor, currency, billing_period,
                   features, max_users, max_products, is_active)
SELECT 'standard', 'Standard', 'Legacy default tier for existing stores', 0,
       'EGP', 'monthly',
       '["pos.basic","inventory.basic","dashboard.basic","pos.subdomain","ocr_capture"]'::jsonb,
       3, 100, FALSE
WHERE NOT EXISTS (SELECT 1 FROM plans WHERE code = 'standard');

INSERT INTO plans (code, name, description, price_minor, currency, billing_period,
                   features, max_users, max_products, is_active)
SELECT 'test', 'Test', 'Internal test tier', 0,
       'EGP', 'monthly',
       '["pos.basic","inventory.basic","dashboard.basic","pos.subdomain","ocr_capture"]'::jsonb,
       3, 100, FALSE
WHERE NOT EXISTS (SELECT 1 FROM plans WHERE code = 'test');

-- Backstop: any other tenant holding an unresolvable code is rewritten to
-- 'standard' rather than left to fail closed at the API. This is deliberately
-- a repair rather than a constraint, because adding a foreign key to a code
-- that is a free-text column with 3,000+ existing rows is a larger change than
-- a launch should absorb.
UPDATE tenants t SET plan = 'standard'
WHERE t.plan IS NULL OR t.plan = ''
   OR NOT EXISTS (SELECT 1 FROM plans p WHERE p.code = t.plan);

-- +goose Down
-- Deliberately NOT a full reversal, and this needs saying out loud.
--
-- There is no safe inverse. Moving the 3,187 'standard' tenants anywhere else
-- would downgrade paying customers, and deleting the rows while they still
-- point at them would recreate the exact dangling reference this migration
-- exists to fix — the same failure, with a migration number claiming it was
-- undone.
--
-- So the rows are removed only when nothing references them, which is the
-- empty case in any real deployment. Reversing this migration in production is
-- a no-op by design.
DELETE FROM plans
WHERE code IN ('standard', 'test')
  AND is_active = FALSE
  AND NOT EXISTS (SELECT 1 FROM tenants t WHERE t.plan IN ('standard', 'test'));
