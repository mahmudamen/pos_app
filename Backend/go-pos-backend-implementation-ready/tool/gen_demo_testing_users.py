#!/usr/bin/env python3
"""Generate the demo testing credential set: 5 demo tenants x 5 users.

Every user gets a unique, complex password (upper + lower + digit + symbol,
20 chars) that is bcrypt-hashed exactly the way the API hashes logins, so the
seed can be applied straight into `users.password_hash`.

Outputs (all under .secrets/, which is gitignored -- never commit these):
  .secrets/testing/demo_users_seed.sql   idempotent upsert of the 5 users/tenant
  .secrets/testing/demo-users.csv        credentials listing for humans
  .secrets/testing/demo-users.json       machine-readable form of the same

Run:  python3 tool/gen_demo_testing_users.py
Then: psql "$DATABASE_URL" -f .secrets/testing/demo_users_seed.sql
"""

from __future__ import annotations

import csv
import json
import secrets
import string
import sys
from datetime import datetime, timezone
from pathlib import Path

import bcrypt

REPO = Path(__file__).resolve().parents[1]
WORKSPACE = Path(__file__).resolve().parents[3]
OUT_DIR = WORKSPACE / ".secrets" / "testing"
BCRYPT_ROUNDS = 10  # matches golang.org/x/crypto/bcrypt default in the API

# 5 demo tenants, one per business vertical (slugs are load-bearing: the Play
# screenshot tours and the docs sign in against demo-restaurant).
TENANTS = [
    ("demo-restaurant", "Demo Restaurant", "restaurant"),
    ("demo-book-store", "Demo Book Store", "book_store"),
    ("demo-mobile-shop", "Demo Mobile Shop", "mobile_shop"),
    ("demo-computer-shop", "Demo Computer Shop", "computer_shop"),
    ("demo-grocery", "Demo Grocery", "grocery"),
]

# 5 users per tenant: two managers so manager-approval paths can be exercised
# by two different people, plus a guest account (account_type=guest).
# The local part of the address is <role>; the full address is built as
# demo-<tenant>-<role>@xamltech.com, e.g. demo-restaurant-manager@xamltech.com.
USERS = [
    ("owner", "owner", "demo", "Store Owner"),
    ("manager", "manager", "demo", "Store Manager"),
    ("manager2", "manager", "demo", "Second Manager"),
    ("cashier", "cashier", "demo", "Senior Cashier"),
    ("guest", "cashier", "guest", "Guest Cashier"),
]

EMAIL_DOMAIN = "xamltech.com"

# SaaS control-panel operators live in the platform tenant (slug `saas`), NOT in
# a demo shop. `saas:admin` is granted to the saas_admin role only
# (internal/transport/http/permissions.go), so owner/manager would 403 on every
# /v1/saas/* and /v1/platform/* route -- the display name carries the
# owner/manager distinction instead.
SAAS_TENANT_SLUG = "saas"
SAAS_OPERATORS = [
    ("mahmudamen@xamltech.com", "Mahmoud Amen (Owner)", "saas_admin", "standard"),
    ("osama@xamltech.com", "Osama (Manager)", "saas_admin", "standard"),
]

# No $ ` ' \ or " -- keeps the password safe inside SQL literals and shells.
SYMBOLS = "!#%&*+-=?@"
ALPHABET = string.ascii_letters + string.digits + SYMBOLS


def generate_password() -> str:
    """20-char password guaranteeing all four character classes."""
    required = [
        secrets.choice(string.ascii_uppercase),
        secrets.choice(string.ascii_lowercase),
        secrets.choice(string.digits),
        secrets.choice(SYMBOLS),
    ]
    body = [secrets.choice(ALPHABET) for _ in range(20 - len(required))]
    chars = required + body
    secrets.SystemRandom().shuffle(chars)
    return "".join(chars)


def sql_literal(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def main() -> int:
    if len(sys.argv) > 1:
        print(f"unexpected argument: {sys.argv[1]!r}", file=sys.stderr)
        return 2

    OUT_DIR.mkdir(parents=True, exist_ok=True)
    generated_at = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")

    rows: list[dict[str, str]] = []
    seen: set[str] = set()
    for slug, name, business_type in TENANTS:
        for local, role, account_type, display in USERS:
            tenant_token = slug.removeprefix("demo-")
            email = f"demo-{tenant_token}-{local}@{EMAIL_DOMAIN}"
            while True:
                password = generate_password()
                if password not in seen:  # never reuse a password across users
                    seen.add(password)
                    break
            rows.append(
                {
                    "tenant_slug": slug,
                    "tenant_name": name,
                    "business_type": business_type,
                    "email": email,
                    "password": password,
                    "role": role,
                    "account_type": account_type,
                    "display_name": display,
                    "password_hash": bcrypt.hashpw(
                        password.encode(), bcrypt.gensalt(rounds=BCRYPT_ROUNDS)
                    ).decode(),
                }
            )

    for email, display, role, account_type in SAAS_OPERATORS:
        while True:
            password = generate_password()
            if password not in seen:
                seen.add(password)
                break
        rows.append(
            {
                "tenant_slug": SAAS_TENANT_SLUG,
                "tenant_name": "POS Go SaaS Platform",
                "business_type": "general",
                "email": email,
                "password": password,
                "role": role,
                "account_type": account_type,
                "display_name": display,
                "password_hash": bcrypt.hashpw(
                    password.encode(), bcrypt.gensalt(rounds=BCRYPT_ROUNDS)
                ).decode(),
            }
        )

    by_slug: dict[str, list[dict[str, str]]] = {}
    for row in rows:
        by_slug.setdefault(row["tenant_slug"], []).append(row)

    blocks = []
    for slug, _name, _type in TENANTS:
        values = ",\n      ".join(
            "(tid, {email}, {hash}, {display}, {role}, {account})".format(
                email=sql_literal(row["email"]),
                hash=sql_literal(row["password_hash"]),
                display=sql_literal(row["display_name"]),
                role=sql_literal(row["role"]),
                account=sql_literal(row["account_type"]),
            )
            for row in by_slug[slug]
        )
        blocks.append(
            f"""    -- {slug}
    SELECT id INTO tid FROM tenants WHERE slug = {sql_literal(slug)};
    IF tid IS NULL THEN
      RAISE NOTICE 'skip % (tenant missing -- run seed_demo.sql first)', {sql_literal(slug)};
    ELSE
      PERFORM set_config('app.current_tenant', tid::text, true);
      INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
      VALUES
        {values}
      ON CONFLICT (tenant_id, email) DO UPDATE SET
        password_hash   = EXCLUDED.password_hash,
        display_name    = EXCLUDED.display_name,
        role            = EXCLUDED.role,
        account_type    = EXCLUDED.account_type,
        tenant_id       = EXCLUDED.tenant_id,
        is_active       = TRUE,
        status          = 'active',
        updated_at      = now();
      RAISE NOTICE '% -> users seeded', {sql_literal(slug)};
    END IF;"""
        )

    saas_rows = [r for r in rows if r["tenant_slug"] == SAAS_TENANT_SLUG]
    saas_values = ",\n      ".join(
        "(tid, {email}, {hash}, {display}, {role}, {account})".format(
            email=sql_literal(row["email"]),
            hash=sql_literal(row["password_hash"]),
            display=sql_literal(row["display_name"]),
            role=sql_literal(row["role"]),
            account=sql_literal(row["account_type"]),
        )
        for row in saas_rows
    )

    sql = f"""-- Demo testing credentials: 5 tenants x 5 users, unique complex passwords.
-- Generated {generated_at} by tool/gen_demo_testing_users.py
-- Idempotent upsert. Run as the database owner, after seed_demo.sql has created
-- the tenants:  psql "$DATABASE_URL" -f .secrets/testing/demo_users_seed.sql
--
-- This file is gitignored. Do not commit it.

DO $$
DECLARE
  tid UUID;
BEGIN
{chr(10).join(blocks)}

  -- SaaS control-panel operators (platform tenant, role saas_admin)
  SELECT id INTO tid FROM tenants WHERE slug = '{SAAS_TENANT_SLUG}';
  IF tid IS NULL THEN
    RAISE NOTICE 'skip {SAAS_TENANT_SLUG} operator seed (platform tenant missing)';
  ELSE
    PERFORM set_config('app.current_tenant', tid::text, true);
    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
    VALUES
      {saas_values}
    ON CONFLICT (tenant_id, email) DO UPDATE SET
      password_hash   = EXCLUDED.password_hash,
      display_name    = EXCLUDED.display_name,
      role            = EXCLUDED.role,
      account_type    = EXCLUDED.account_type,
      tenant_id       = EXCLUDED.tenant_id,
      is_active       = TRUE,
      status          = 'active',
      updated_at      = now();
    RAISE NOTICE '{SAAS_TENANT_SLUG}: % control-panel operators seeded', {len(saas_rows)};
  END IF;

  RAISE NOTICE '=== demo testing users ===';
  FOR tid IN SELECT id FROM tenants WHERE slug LIKE 'demo-%' ORDER BY slug LOOP
    PERFORM set_config('app.current_tenant', tid::text, true);
    RAISE NOTICE '% users=%', (SELECT slug FROM tenants WHERE id = tid),
      (SELECT count(*) FROM users WHERE tenant_id = tid AND is_active);
  END LOOP;
END;
$$;
"""

    (OUT_DIR / "demo_users_seed.sql").write_text(sql, encoding="utf-8")

    header = [
        "tenant_slug", "tenant_name", "business_type", "email", "password",
        "role", "account_type", "display_name",
    ]
    with (OUT_DIR / "demo-users.csv").open("w", newline="", encoding="utf-8") as fh:
        writer = csv.DictWriter(fh, fieldnames=header, extrasaction="ignore")
        writer.writeheader()
        writer.writerows(rows)

    (OUT_DIR / "demo-users.json").write_text(
        json.dumps({"generated_at": generated_at, "users": rows}, indent=2),
        encoding="utf-8",
    )

    print(f"generated {len(rows)} credentials across {len(by_slug)} tenants")
    print(f"  seed sql : {OUT_DIR / 'demo_users_seed.sql'}")
    print(f"  csv      : {OUT_DIR / 'demo-users.csv'}")
    print(f"  json     : {OUT_DIR / 'demo-users.json'}")
    print(f"  unique passwords: {len(seen)}/{len(rows)}")
    for row in rows[:5]:
        print(f"    {row['email']:34} {row['role']:8} {row['account_type']:8} {row['password']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())