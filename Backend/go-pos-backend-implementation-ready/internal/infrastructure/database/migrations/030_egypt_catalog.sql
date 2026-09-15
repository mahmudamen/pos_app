-- +goose Up
-- Flag tenants whose catalog was seeded from the bundled Egyptian demo data;
-- the price-refresh job only rewrites prices for these tenants, never a real
-- merchant's catalog.
ALTER TABLE tenants ADD COLUMN is_demo_seeded BOOLEAN NOT NULL DEFAULT FALSE;

-- Bundled Egyptian price catalog (GS1-Egypt EAN-13 622xxx). The demo grocery
-- vertical seeds these EXACT items, and the pricing job matches products by
-- barcode to refresh price/cost from here on an interval.
CREATE TABLE egypt_price_catalog (
    barcode     CHAR(13) PRIMARY KEY,
    name        TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT '',
    unit        TEXT NOT NULL DEFAULT 'piece',
    price_minor BIGINT NOT NULL CHECK (price_minor >= 0),
    cost_minor  BIGINT NOT NULL CHECK (cost_minor >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO egypt_price_catalog (barcode, name, category, unit, price_minor, cost_minor) VALUES
    ('6221010000017', 'Juhayna Fresh Milk 1L',    'dairy',     'liter', 4200,  3800),
    ('6221010000024', 'Juhayna Plain Yogurt 500g', 'dairy',     'piece', 2600,  2300),
    ('6221010000031', 'Domty Cheddar Cheese 100g', 'dairy',     'piece', 5500,  5000),
    ('6221010000048', 'Panda Cheese Triangles',    'dairy',     'piece', 3400,  3000),
    ('6221010000062', 'Coca-Cola 1L',              'beverages', 'liter', 2400,  2000),
    ('6221010000086', 'Schweppes Lemon 1L',        'beverages', 'liter', 2600,  2200),
    ('6221010000093', 'Mineral Water 1.5L',        'beverages', 'piece', 1500,  1100),
    ('6221010000109', 'Syrup Mango Juice 1L',      'beverages', 'liter', 3800,  3200),
    ('6221010000116', 'Lipton Yellow Label Tea 50g','pantry',   'piece', 6500,  5800),
    ('6221010000123', 'Nescafe Classic 100g',      'pantry',    'piece', 14500, 13200),
    ('6221010000147', 'El-Mashreq Sugar 1kg',      'pantry',    'kg',    3100,  2900),
    ('6221010000154', 'El-Gomhoria Rice 1kg',      'pantry',    'kg',    4200,  3800),
    ('6221010000178', 'Al-Shark Pasta 400g',       'pantry',    'piece', 1800,  1500),
    ('6221010000185', 'Mazola Corn Oil 1L',        'pantry',    'liter', 13500, 12500),
    ('6221010000208', 'El-Nasr Table Salt 1kg',    'pantry',    'kg',    800,   500),
    ('6221010000215', 'Brown Lentils 1kg',         'pantry',    'kg',    5800,  5200),
    ('6221010000239', 'Chipsy Chips 80g',          'snacks',    'piece', 1600,  1300),
    ('6221010000246', 'Lays Chips 80g',            'snacks',    'piece', 1700,  1400),
    ('6221010000253', 'Oman Chips King',           'snacks',    'piece', 1200,  950),
    ('6221010000307', 'Stella Biscuits 400g',      'snacks',    'piece', 3500,  3000);

-- +goose Down
DROP TABLE IF EXISTS egypt_price_catalog;
ALTER TABLE tenants DROP COLUMN IF EXISTS is_demo_seeded;