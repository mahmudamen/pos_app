-- +goose Up
-- Refuse to store a `users.password_hash` that bcrypt could never have produced.
--
-- Background. On production, `admin@posgo.saas` was found holding a 19-character
-- value instead of a 60-character bcrypt hash: the trailing fragment
-- 'a0/Ntt.YOoBC6TIr1Uu'. bcrypt.CompareHashAndPassword rejects that outright,
-- so the account silently stopped authenticating — a 401 on correct credentials,
-- with no error anywhere: login just reported a wrong password. `support@posgo.saas`
-- held a full hash and returned 200 on the same password, so the fault was
-- invisible from the client side and looked like a bad credential, not a broken
-- row.
--
-- Comparing every retained dump pinned the window exactly: the hash is intact in
-- pos-20261004T001701Z.sql.gz and truncated in pos-20261005T001701Z.sql.gz, with
-- the row's own updated_at at 2026-10-04 14:22, some 28 minutes after that
-- account's last successful login. Nothing in `audit_log` records the write, and
-- Postgres runs with log_statement = none, so the statement is not recoverable
-- from this host. No code path in the application updates password_hash at all
-- (only INSERT), and no substring or slice of it appears in any script, so the
-- cause is unproven — which is precisely why the value is now constrained rather
-- than merely repaired.
--
-- The constraint is the fix; the row repair was manual. bcrypt modular crypt
-- strings are `$2<variant>$<cost>$` followed by 53 base64 characters from the
-- bcrypt alphabet (22 of salt, 31 of digest), giving 60 characters for the
-- `$2a`/`$2b`/`$2y` variants this codebase produces. Anything that does not match
-- is corrupt by definition — there is no legacy plaintext to grandfather in,
-- because a plaintext password in this column would be the worse defect.
--
-- NOT VALID is used deliberately: it enforces on every subsequent write while
-- skipping the scan of existing rows, so adding it cannot lock or fail on a
-- table this size. The VALIDATE below then proves the historical data already
-- complies, turning it into a fully verified constraint. If a deployment still
-- holds a corrupt row, VALIDATE fails loudly here instead of leaving the account
-- quietly broken — which is the outcome this migration exists to prevent.
ALTER TABLE users
    ADD CONSTRAINT users_password_hash_format_chk
    CHECK (password_hash ~ '^\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}$')
    NOT VALID;

ALTER TABLE users
    VALIDATE CONSTRAINT users_password_hash_format_chk;

-- +goose Down
-- Drops the guard. Only safe once every account is known to authenticate: this
-- is the same state the database was in when `admin@posgo.saas` broke, where a
-- corrupt hash is indistinguishable from a wrong password at the login screen.
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_password_hash_format_chk;