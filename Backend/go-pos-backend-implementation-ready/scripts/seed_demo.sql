-- Demo data seed for SaaS POS: one tenant per business type.
-- Idempotent: safe to run multiple times (skips tenants that already exist).
-- Run as the database owner:  psql $DATABASE_URL -f scripts/seed_demo.sql
--
-- Business types: restaurant, book_store, mobile_shop, computer_shop, grocery
-- Each tenant gets: owner@<domain> / admin, admin@<domain> / admin, cashier@<domain> / admin

DO $$
DECLARE
  tid UUID;
  c   UUID;
BEGIN
  -- --------------------------------------------------------------------------
  -- 1. Demo Restaurant (cafe + food)
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'demo-restaurant') THEN
    INSERT INTO tenants (name, slug, business_type, address) VALUES ('Demo Restaurant', 'demo-restaurant', 'restaurant', '35 Nile Corniche, Cairo') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'owner@demo-restaurant.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Restaurant Owner', 'owner', 'demo'),
      (tid, 'admin@demo-restaurant.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Restaurant Admin', 'manager', 'demo'),
      (tid, 'cashier@demo-restaurant.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Restaurant Cashier', 'cashier', 'demo'),
      (tid, 'guest@demo-restaurant.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Guest', 'cashier', 'guest');

    INSERT INTO categories (tenant_id, name, slug) VALUES
      (tid, 'Drinks', 'drinks'), (tid, 'Food', 'food'), (tid, 'Pastries', 'pastries'), (tid, 'Desserts', 'desserts');

    INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity) VALUES
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='drinks'), 'Espresso', 'ESP-001', '1000000000001', 280, 80, 'EGP', 100),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='drinks'), 'Latte', 'LAT-001', '1000000000002', 350, 100, 'EGP', 80),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='drinks'), 'Cappuccino', 'CAP-001', '1000000000003', 350, 100, 'EGP', 80),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='drinks'), 'Iced Americano', 'ICE-001', '1000000000004', 300, 70, 'EGP', 60),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='drinks'), 'Matcha Latte', 'MAT-001', '1000000000005', 400, 120, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='food'), 'Grilled Sandwich', 'SND-001', '2000000000001', 550, 200, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='food'), 'Caesar Salad', 'SAL-001', '2000000000002', 480, 150, 'EGP', 25),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='food'), 'Pasta Carbonara', 'PAS-001', '2000000000003', 650, 250, 'EGP', 20),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='food'), 'Club Wrap', 'WRP-001', '2000000000004', 500, 180, 'EGP', 25),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='pastries'), 'Croissant', 'CRN-001', '3000000000001', 250, 80, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='pastries'), 'Blueberry Muffin', 'MUF-001', '3000000000002', 300, 90, 'EGP', 35),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='pastries'), 'Chocolate Brownie', 'BRN-001', '3000000000003', 280, 85, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='desserts'), 'Tiramisu', 'TIR-001', '4000000000001', 420, 150, 'EGP', 20),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='desserts'), 'New York Cheesecake', 'CHS-001', '4000000000002', 450, 170, 'EGP', 18);
  END IF;

  -- --------------------------------------------------------------------------
  -- 2. Demo Book Store
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'demo-book-store') THEN
    INSERT INTO tenants (name, slug, business_type, address) VALUES ('Demo Book Store', 'demo-book-store', 'book_store', '12 Kasr El Eini St, Cairo') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'owner@demo-book-store.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Book Store Owner', 'owner', 'demo'),
      (tid, 'admin@demo-book-store.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Book Store Admin', 'manager', 'demo'),
      (tid, 'cashier@demo-book-store.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Book Store Cashier', 'cashier', 'demo'),
      (tid, 'guest@demo-book-store.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Guest', 'cashier', 'guest');

    INSERT INTO categories (tenant_id, name, slug) VALUES
      (tid, 'Fiction', 'fiction'), (tid, 'Non-Fiction', 'non-fiction'), (tid, 'Stationery', 'stationery'), (tid, 'Textbooks', 'textbooks');

    INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity) VALUES
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='fiction'), 'The Great Gatsby', 'BK-001', '1000000000101', 1299, 500, 'EGP', 50),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='fiction'), '1984', 'BK-002', '1000000000102', 1099, 450, 'EGP', 60),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='fiction'), 'To Kill a Mockingbird', 'BK-003', '1000000000103', 1199, 480, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='fiction'), 'Dune', 'BK-004', '1000000000104', 1499, 600, 'EGP', 25),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='non-fiction'), 'Sapiens', 'BK-005', '1000000000105', 1899, 750, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='non-fiction'), 'Atomic Habits', 'BK-006', '1000000000106', 1599, 650, 'EGP', 35),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='stationery'), 'Notebook A5', 'ST-001', '1000000000107', 399, 120, 'EGP', 200),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='stationery'), 'Gel Pen Set', 'ST-002', '1000000000108', 599, 180, 'EGP', 150),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='textbooks'), 'Calculus Volume 1', 'TB-001', '1000000000109', 4999, 2000, 'EGP', 15),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='textbooks'), 'Physics for Dummies', 'TB-002', '1000000000110', 2499, 1000, 'EGP', 20),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='textbooks'), 'Organic Chemistry', 'TB-003', '1000000000111', 5999, 2500, 'EGP', 12);
  END IF;

  -- --------------------------------------------------------------------------
  -- 3. Demo Mobile Shop
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'demo-mobile-shop') THEN
    INSERT INTO tenants (name, slug, business_type, address) VALUES ('Demo Mobile Shop', 'demo-mobile-shop', 'mobile_shop', '24 Tahrir Sq, Cairo') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'owner@demo-mobile-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Mobile Shop Owner', 'owner', 'demo'),
      (tid, 'admin@demo-mobile-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Mobile Shop Admin', 'manager', 'demo'),
      (tid, 'cashier@demo-mobile-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Mobile Shop Cashier', 'cashier', 'demo'),
      (tid, 'guest@demo-mobile-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Guest', 'cashier', 'guest');

    INSERT INTO categories (tenant_id, name, slug) VALUES
      (tid, 'Smartphones', 'smartphones'), (tid, 'Accessories', 'accessories'), (tid, 'Cases', 'cases'), (tid, 'Chargers', 'chargers');

    INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity) VALUES
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='smartphones'), 'iPhone 15 Pro', 'MP-001', '1000000000201', 99900, 80000, 'EGP', 8),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='smartphones'), 'Samsung Galaxy S24', 'MP-002', '1000000000202', 79900, 65000, 'EGP', 10),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='smartphones'), 'Google Pixel 8', 'MP-003', '1000000000203', 69900, 56000, 'EGP', 6),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='smartphones'), 'Xiaomi Redmi Note 13', 'MP-004', '1000000000204', 29900, 24000, 'EGP', 15),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='accessories'), 'Bluetooth Earbuds Pro', 'AC-001', '1000000000205', 4900, 2500, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='accessories'), 'Smart Watch GT4', 'AC-002', '1000000000206', 12900, 8000, 'EGP', 20),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='cases'), 'Silicone Case iPhone 15', 'CS-001', '1000000000207', 1999, 500, 'EGP', 50),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='cases'), 'Clear Case Samsung S24', 'CS-002', '1000000000208', 1499, 400, 'EGP', 55),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='chargers'), 'USB-C Fast Charger 65W', 'CH-001', '1000000000209', 3499, 1200, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='chargers'), 'MagSafe Wireless Charger', 'CH-002', '1000000000210', 3999, 1500, 'EGP', 25),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='chargers'), 'Power Bank 20000mAh', 'CH-003', '1000000000211', 4499, 2000, 'EGP', 22);
  END IF;

  -- --------------------------------------------------------------------------
  -- 4. Demo Computer Shop
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'demo-computer-shop') THEN
    INSERT INTO tenants (name, slug, business_type, address) VALUES ('Demo Computer Shop', 'demo-computer-shop', 'computer_shop', '7 El-Thawra St, Alexandria') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'owner@demo-computer-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Computer Shop Owner', 'owner', 'demo'),
      (tid, 'admin@demo-computer-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Computer Shop Admin', 'manager', 'demo'),
      (tid, 'cashier@demo-computer-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Computer Shop Cashier', 'cashier', 'demo'),
      (tid, 'guest@demo-computer-shop.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Guest', 'cashier', 'guest');

    INSERT INTO categories (tenant_id, name, slug) VALUES
      (tid, 'Laptops', 'laptops'), (tid, 'Components', 'components'), (tid, 'Peripherals', 'peripherals'), (tid, 'Software', 'software');

    INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity) VALUES
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='laptops'), 'MacBook Pro 14"', 'CP-001', '1000000000301', 199900, 160000, 'EGP', 4),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='laptops'), 'Dell XPS 13', 'CP-002', '1000000000302', 129900, 100000, 'EGP', 6),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='laptops'), 'Lenovo ThinkPad X1', 'CP-003', '1000000000303', 149900, 115000, 'EGP', 5),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='components'), 'SSD NVMe 1TB', 'CC-001', '1000000000304', 10999, 7000, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='components'), 'RAM 32GB DDR5', 'CC-002', '1000000000305', 9999, 6500, 'EGP', 28),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='components'), 'RTX 4070 Graphics Card', 'CC-003', '1000000000306', 59900, 45000, 'EGP', 7),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='peripherals'), 'Mechanical Keyboard RGB', 'CPE-001', '1000000000307', 8999, 4000, 'EGP', 25),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='peripherals'), 'Gaming Mouse Pro', 'CPE-002', '1000000000308', 5999, 2800, 'EGP', 32),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='peripherals'), '4K Monitor 27"', 'CPE-003', '1000000000309', 32900, 25000, 'EGP', 10),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='software'), 'Windows 11 Pro License', 'CSW-001', '1000000000310', 19999, 10000, 'EGP', 100),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='software'), 'Office Suite 1-Year', 'CSW-002', '1000000000311', 9999, 5000, 'EGP', 80);
  END IF;

  -- --------------------------------------------------------------------------
  -- 5. Demo Grocery
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'demo-grocery') THEN
    INSERT INTO tenants (name, slug, business_type, address) VALUES ('Demo Grocery', 'demo-grocery', 'grocery', '3 Zamalek St, Cairo') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'owner@demo-grocery.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Grocery Owner', 'owner', 'demo'),
      (tid, 'admin@demo-grocery.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Grocery Admin', 'manager', 'demo'),
      (tid, 'cashier@demo-grocery.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Grocery Cashier', 'cashier', 'demo'),
      (tid, 'guest@demo-grocery.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'Guest', 'cashier', 'guest');

    INSERT INTO categories (tenant_id, name, slug) VALUES
      (tid, 'Produce', 'produce'), (tid, 'Dairy', 'dairy'), (tid, 'Bakery', 'bakery'), (tid, 'Beverages', 'beverages'), (tid, 'Snacks', 'snacks');

    INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity) VALUES
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='produce'), 'Apples (lb)', 'GR-001', '1000000000401', 299, 120, 'EGP', 80),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='produce'), 'Bananas (lb)', 'GR-002', '1000000000402', 199, 80, 'EGP', 90),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='produce'), 'Tomatoes (lb)', 'GR-003', '1000000000403', 249, 100, 'EGP', 60),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='dairy'), 'Whole Milk 1L', 'GR-004', '1000000000404', 349, 200, 'EGP', 50),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='dairy'), 'Cheddar Cheese 8oz', 'GR-005', '1000000000405', 499, 300, 'EGP', 35),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='dairy'), 'Greek Yogurt 500g', 'GR-006', '1000000000406', 399, 220, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='bakery'), 'Sourdough Loaf', 'GR-007', '1000000000407', 499, 180, 'EGP', 30),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='bakery'), 'Bagel (each)', 'GR-008', '1000000000408', 199, 60, 'EGP', 70),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='beverages'), 'Orange Juice 1L', 'GR-009', '1000000000409', 499, 250, 'EGP', 45),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='beverages'), 'Sparkling Water', 'GR-010', '1000000000410', 249, 100, 'EGP', 65),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='beverages'), 'Cola 2L', 'GR-011', '1000000000411', 299, 150, 'EGP', 55),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='snacks'), 'Potato Chips', 'GR-012', '1000000000412', 399, 180, 'EGP', 40),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='snacks'), 'Chocolate Bar', 'GR-013', '1000000000413', 299, 130, 'EGP', 60),
      (tid, (SELECT id FROM categories WHERE tenant_id=tid AND slug='snacks'), 'Mixed Nuts 200g', 'GR-014', '1000000000414', 599, 300, 'EGP', 38);
  END IF;

  -- --------------------------------------------------------------------------
  -- 6. SaaS platform tenant (internal staff role: saas_admin -> control panel)
  -- --------------------------------------------------------------------------
  IF NOT EXISTS (SELECT 1 FROM tenants WHERE slug = 'saas') THEN
    INSERT INTO tenants (name, slug, business_type, country_code, currency_code, default_language, address)
    VALUES ('POS Go SaaS Platform', 'saas', 'general', 'EG', 'EGP', 'ar', 'Cairo, Egypt') RETURNING id INTO tid;
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type) VALUES
      (tid, 'admin@posgo.saas', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'SaaS Admin', 'saas_admin', 'standard'),
      (tid, 'support@posgo.saas', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2', 'SaaS Support', 'saas_admin', 'standard');
  END IF;

  -- --------------------------------------------------------------------------
  -- Owner backfill: tenants created by an older seed version won't have an
  -- owner role user. Ensure every typed demo tenant has exactly one owner.
  -- --------------------------------------------------------------------------
  FOR tid IN
    SELECT id FROM tenants WHERE slug IN
      ('demo-restaurant', 'demo-book-store', 'demo-mobile-shop', 'demo-computer-shop', 'demo-grocery', 'demo-store')
  LOOP
    IF NOT EXISTS (SELECT 1 FROM users WHERE tenant_id = tid AND role = 'owner') THEN
      PERFORM set_config('app.current_tenant', tid::text, true);
      INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
      SELECT tid, 'owner@' || slug || '.com', '$2a$10$YDywTS/ci9Tzqz12X9KhOuM8OjdfwlXn3eVHus1ykgXXvgRg/8nw2',
             (SELECT name FROM tenants WHERE id = tid) || ' Owner', 'owner', 'demo'
      FROM tenants WHERE id = tid;
    END IF;
  END LOOP;

  -- --------------------------------------------------------------------------
  -- Summary (printed per tenant; queries run inside tenant RLS context)
  -- --------------------------------------------------------------------------
  FOR tid IN
    SELECT id FROM tenants ORDER BY business_type
  LOOP
    PERFORM set_config('app.current_tenant', tid::text, true);
    RAISE NOTICE '=== % | users=% categories=% products=%',
      (SELECT business_type FROM tenants WHERE id = tid),
      (SELECT count(*) FROM users WHERE tenant_id = tid),
      (SELECT count(*) FROM categories WHERE tenant_id = tid),
      (SELECT count(*) FROM products WHERE tenant_id = tid);
  END LOOP;
END;
$$;