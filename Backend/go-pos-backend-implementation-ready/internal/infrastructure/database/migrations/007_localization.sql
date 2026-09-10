-- +goose Up
-- Localization & SaaS tenant metadata
ALTER TABLE tenants
    ADD COLUMN country_code CHAR(2) NOT NULL DEFAULT 'EG',
    ADD COLUMN currency_code CHAR(3) NOT NULL DEFAULT 'EGP',
    ADD COLUMN default_language TEXT NOT NULL DEFAULT 'ar';

-- Reference: countries with their currency (demo: Egypt only)
CREATE TABLE countries (
    code CHAR(2) PRIMARY KEY,
    name_en TEXT NOT NULL,
    name_ar TEXT NOT NULL,
    currency_code CHAR(3) NOT NULL,
    phone_code TEXT NOT NULL
);

-- Reference: currencies (demo: EGP only)
CREATE TABLE currencies (
    code CHAR(3) PRIMARY KEY,
    name_en TEXT NOT NULL,
    name_ar TEXT NOT NULL,
    symbol TEXT NOT NULL,
    digits_after_decimal INT NOT NULL DEFAULT 2
);

INSERT INTO countries (code, name_en, name_ar, currency_code, phone_code) VALUES
    ('EG', 'Egypt', 'مصر', 'EGP', '+20');

INSERT INTO currencies (code, name_en, name_ar, symbol, digits_after_decimal) VALUES
    ('EGP', 'Egyptian Pound', 'جنيه مصري', 'E£', 2);

-- SaaS user lifecycle: standard / demo / guest
ALTER TABLE users ADD COLUMN account_type TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE users ADD CONSTRAINT users_account_type_check
    CHECK (account_type IN ('standard', 'demo', 'guest'));

-- SaaS internal staff get a platform-level role (sees the SaaS control panel)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('manager', 'cashier', 'saas_admin'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('manager', 'cashier'));
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_account_type_check;
ALTER TABLE users DROP COLUMN IF EXISTS account_type;

DROP TABLE IF EXISTS currencies;
DROP TABLE IF EXISTS countries;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS default_language,
    DROP COLUMN IF EXISTS currency_code,
    DROP COLUMN IF EXISTS country_code;