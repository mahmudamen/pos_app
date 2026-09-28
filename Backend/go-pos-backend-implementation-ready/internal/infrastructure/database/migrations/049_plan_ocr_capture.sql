-- +goose Up
-- Add `ocr_capture` to the plan catalog so invoice capture is a listed,
-- priced-in capability rather than an unpriced accident of the build.
--
-- Why this migration exists: until now nothing in the codebase read
-- `plans.features` to make a decision. It was serialised into SaaS responses
-- and rendered in the /pricing comparison table, and `POST /v1/purchases/ocr`
-- was reachable by anyone holding the `inventory.adjust` role, regardless of
-- what they paid for. internal/plans is the first real read of the column and
-- /v1/purchases/ocr now enforces it.
--
-- Every seeded plan gains the feature, deliberately. The metering windows
-- (ocr_usage + ocr_credits_remaining, migration 036) are the intended control
-- surface for this feature, and they already answer 402 when a tenant is out
-- of allowance. The plan row is the entitlement, the meter is the throttle:
-- gating on both would mean a starter shop is refused a scan it has paid for,
-- and a bug in the meter locks a shop out of stock-taking entirely. If OCR
-- becomes a price differentiator, remove it from `starter`/`trial` here in
-- its own migration and the handler needs no change.
--
-- Dedupe-guarded: re-running against a catalog that already lists the feature
-- is a no-op. Plain UPDATE (goose cannot parse DO $$...$$ blocks).
UPDATE plans
SET features = features || '["ocr_capture"]'::jsonb,
    updated_at = now()
WHERE NOT features @> '["ocr_capture"]'::jsonb;

-- +goose Down
UPDATE plans
SET features = features - 'ocr_capture',
    updated_at = now()
WHERE features @> '["ocr_capture"]'::jsonb;
