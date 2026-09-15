-- +goose Up
-- Units of measure (platform reference data, not tenant-scoped).
CREATE TABLE units (
    code    TEXT PRIMARY KEY,
    name_en TEXT NOT NULL,
    name_ar TEXT NOT NULL,
    symbol  TEXT NOT NULL,
    is_base BOOLEAN NOT NULL DEFAULT FALSE
);

-- Unit conversions, e.g. 1 box = 6 piece, 1 dozen = 12 piece, 1000 g = 1 kg.
CREATE TABLE unit_conversions (
    from_unit TEXT NOT NULL REFERENCES units(code),
    to_unit   TEXT NOT NULL REFERENCES units(code),
    factor    NUMERIC(18,6) NOT NULL CHECK (factor > 0),
    PRIMARY KEY (from_unit, to_unit)
);

-- Retail units for the demo catalog.
INSERT INTO units (code, name_en, name_ar, symbol, is_base) VALUES
    ('piece', 'Piece',        'قطعة',        'pcs', TRUE),
    ('dozen', 'Dozen',        'دستة',        'dz',  FALSE),
    ('box',   'Box',          'علبة',        'box', FALSE),
    ('pack',  'Pack',         'عبوة',        'pack', FALSE),
    ('kg',    'Kilogram',     'كيلو جرام',   'kg',  TRUE),
    ('g',     'Gram',         'جرام',        'g',   FALSE),
    ('liter', 'Liter',        'لتر',         'L',   TRUE),
    ('ml',    'Milliliter',   'ملليلتر',     'ml',  FALSE),
    ('m',     'Meter',        'متر',         'm',   TRUE),
    ('qm',    'Square meter', 'متر مربع',    'm²',  FALSE);

INSERT INTO unit_conversions (from_unit, to_unit, factor) VALUES
    ('dozen', 'piece', 12),
    ('box',   'piece', 6),
    ('pack',  'piece', 1),
    ('g',     'kg',    0.001),
    ('ml',    'liter', 0.001);

-- Every product has a sales unit (default: each / piece).
ALTER TABLE products ADD COLUMN unit TEXT NOT NULL DEFAULT 'piece';
ALTER TABLE products ADD CONSTRAINT products_unit_fk FOREIGN KEY (unit) REFERENCES units(code);

-- +goose Down
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_unit_fk;
ALTER TABLE products DROP COLUMN IF EXISTS unit;
DROP TABLE IF EXISTS unit_conversions;
DROP TABLE IF EXISTS units;