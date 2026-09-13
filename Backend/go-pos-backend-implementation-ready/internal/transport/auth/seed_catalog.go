package auth

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type demoProduct struct {
	category    string // category slug
	name        string
	sku         string
	barcode     string // EAN-13
	priceMinor  int64
	costMinor   int64
	stock       int64
	imageURL    string
	description string
}

type demoVertical struct {
	categories [][2]string // {slug, name}
	products   []demoProduct
}

// seedCatalogs returns one curated demo catalog per supported business type:
// real product names, EGP prices, EAN-13 barcodes, Unsplash image URLs and a
// one-line info text. Products reference "categories" lists above them; each
// vertical gets its own barcode number space so a tenant's codes are unique.
func seedCatalogs() map[string]demoVertical {
	return map[string]demoVertical{
		"coffee_shop": {
			categories: [][2]string{{"drinks", "Drinks"}, {"food", "Food"}, {"pastries", "Pastries"}},
			products: []demoProduct{
				{"drinks", "Espresso", "CF-001", "6220010000010", 280, 80, 100, "photo-1509042239860-f550ce710b93", "Bold single-origin shot, roasted in-house daily."},
				{"drinks", "Cappuccino", "CF-002", "6220010000027", 350, 100, 80, "photo-1572442388796-11668a67e53d", "Espresso with steamed-milk foam and cocoa dusting."},
				{"drinks", "Caffè Latte", "CF-003", "6220010000034", 350, 110, 80, "photo-1541167760496-1628856ab772", "Silky steamed milk over a double espresso."},
				{"drinks", "Iced Americano", "CF-004", "6220010000041", 300, 70, 60, "photo-1517701550927-30cf4ba1dba5", "Chilled double espresso on fresh ice."},
				{"drinks", "Mocha", "CF-005", "6220010000058", 400, 130, 50, "photo-1570968915860-54d5c308fa8e", "Espresso, dark chocolate and steamed milk."},
				{"food", "Avocado Toast", "CF-006", "6220010000065", 650, 260, 25, "photo-1541519227354-08fa5d50c44d", "Sourdough, smashed avocado, poached egg, chili flakes."},
				{"pastries", "Butter Croissant", "CF-007", "6220010000072", 250, 80, 40, "photo-1555507036-ab1f4038808a", "Flaky laminated dough baked fresh every morning."},
				{"pastries", "Blueberry Muffin", "CF-008", "6220010000089", 300, 90, 35, "photo-1607958996333-41aef7caefaa", "Soft muffin loaded with wild blueberries."},
				{"pastries", "Chocolate Chip Cookie", "CF-009", "6220010000096", 220, 70, 50, "photo-1499636136210-6f4ee915583e", "Chewy center, crisp edge, Belgian chocolate."},
				{"pastries", "Cheesecake Slice", "CF-010", "6220010000102", 420, 140, 20, "photo-1578985545062-69928b1d9587", "Baked New York style with a graham crust."},
			},
		},
		"restaurant": {
			categories: [][2]string{{"starters", "Starters"}, {"mains", "Main Courses"}, {"desserts", "Desserts"}},
			products: []demoProduct{
				{"starters", "Caesar Salad", "RS-001", "6220020000017", 480, 150, 25, "photo-1551248429-40975aa4de74", "Romaine, parmesan, croutons and garlic dressing."},
				{"starters", "Tomato Soup", "RS-002", "6220020000024", 350, 120, 30, "photo-1547592166-23ac45744acd", "Roasted tomatoes, basil and a swirl of cream."},
				{"mains", "Grilled Chicken Sandwich", "RS-003", "6220020000031", 550, 200, 30, "photo-1553909489-cd47e0907980", "Chargrilled chicken, garlic mayo, fresh greens."},
				{"mains", "Pasta Carbonara", "RS-004", "6220020000048", 650, 250, 20, "photo-1621996346565-e3dbc646d9a9", "Guanciale, egg yolk, pecorino on fresh tagliatelle."},
				{"mains", "Classic Beef Burger", "RS-005", "6220020000055", 620, 260, 30, "photo-1568901346375-23c9450c58cd", "Angus patty, cheddar, tomato and house sauce."},
				{"mains", "Margherita Pizza", "RS-006", "6220020000062", 850, 350, 15, "photo-1565299624946-b28f40a0ae38", "San Marzano tomato, mozzarella, fresh basil."},
				{"mains", "Grilled Steak", "RS-007", "6220020000079", 999, 600, 12, "photo-1600891964092-4316c288032e", "300g ribeye seared over open flame."},
				{"mains", "Fresh Orange Juice", "RS-008", "6220020000086", 450, 180, 40, "photo-1613478223719-2ab802602423", "Cold-pressed blood oranges, no added sugar."},
				{"desserts", "Molten Lava Cake", "RS-009", "6220020000093", 520, 200, 18, "photo-1578985545062-69928b1d9587", "Dark chocolate cake with a warm gooey center."},
				{"desserts", "Tiramisu", "RS-010", "6220020000109", 480, 180, 15, "photo-1571877227200-a0d98ea607e9", "Espresso-soaked savoiardi, mascarpone cream."},
			},
		},
		"retail": {
			categories: [][2]string{{"essentials", "Essentials"}, {"electronics", "Electronics"}, {"clothing", "Clothing"}},
			products: []demoProduct{
				{"essentials", "Weekender Backpack", "RT-001", "6220030000014", 3900, 1800, 25, "photo-1553062407-98eeb64c6a62", "Water-resistant 28L with padded laptop sleeve."},
				{"essentials", "Sports Water Bottle", "RT-002", "6220030000021", 899, 400, 60, "photo-1602143407151-7111542de6e8", "BPA-free stainless steel, 750ml, leak-proof."},
				{"essentials", "Polarized Sunglasses", "RT-003", "6220030000038", 2599, 1200, 20, "photo-1572635196237-14b3f281503f", "UV400 lenses with a lightweight poly frame."},
				{"electronics", "Bluetooth Speaker", "RT-004", "6220030000045", 4999, 2800, 18, "photo-1608043152269-423dbba4e7e1", "Waterproof, 12-hour playtime, punchy bass."},
				{"electronics", "Wireless Headphones", "RT-005", "6220030000052", 3999, 2200, 22, "photo-1505740420928-5e560c06d30e", "Over-ear, active noise canceling, 30h battery."},
				{"electronics", "Power Bank 20000mAh", "RT-006", "6220030000069", 4499, 2400, 30, "photo-1598331668826-20cecc596b86", "Dual USB-C fast charge for phone and tablet."},
				{"clothing", "Cotton Crew T-Shirt", "RT-007", "6220030000076", 3200, 1400, 40, "photo-1521572163474-6864f9cf17ab", "Premium combed cotton, pre-shrunk regular fit."},
				{"clothing", "Running Sneakers", "RT-008", "6220030000083", 5999, 3500, 15, "photo-1542291026-7eec264c27ff", "Breathable mesh upper with responsive midsole."},
				{"clothing", "Denim Jacket", "RT-009", "6220030000090", 4999, 2800, 12, "photo-1551537482-f2075a1d41f2", "Classic blue denim with button front."},
				{"essentials", "Travel Mug 400ml", "RT-010", "6220030000106", 1299, 550, 35, "photo-1517256064527-09c73fc73e38", "Double-wall vacuum keeps drinks hot for hours."},
			},
		},
		"book_store": {
			categories: [][2]string{{"fiction", "Fiction"}, {"non-fiction", "Non-Fiction"}, {"stationery", "Stationery"}},
			products: []demoProduct{
				{"fiction", "The Great Gatsby", "BK-001", "6220040000011", 1299, 500, 25, "photo-1544716278-ca5e3f4abd8c", "Fitzgerald's classic portrait of the Jazz Age."},
				{"fiction", "1984", "BK-002", "6220040000028", 1099, 450, 30, "photo-1531072901881-d3c19c9b70c7", "Orwell's dystopian masterpiece of surveillance."},
				{"fiction", "Dune", "BK-003", "6220040000035", 1499, 600, 18, "photo-1512820790803-83ca734da794", "Herbert's epic science-fiction universe."},
				{"non-fiction", "Sapiens", "BK-004", "6220040000042", 1899, 750, 20, "photo-1589998059171-988d887df646", "A brief history of humankind by Yuval Noah Harari."},
				{"non-fiction", "Atomic Habits", "BK-005", "6220040000059", 1599, 650, 28, "photo-1518791841217-8f162f1e1131", "Tiny changes, remarkable results by James Clear."},
				{"non-fiction", "Thinking, Fast and Slow", "BK-006", "6220040000066", 1999, 850, 15, "photo-1524995997946-a1c2e315a42f", "Kahneman's guide to the two systems of thought."},
				{"stationery", "Notebook A5", "BK-007", "6220040000073", 399, 120, 100, "photo-1544816155-12df9643f363", "120 ruled pages, hardcover, lay-flat binding."},
				{"stationery", "Gel Pen Set", "BK-008", "6220040000080", 599, 180, 80, "photo-1583485088034-697b5bc54ccd", "Five smooth 0.5mm gel pens in assorted inks."},
				{"stationery", "A4 Sketchbook", "BK-009", "6220040000097", 799, 300, 45, "photo-1558591710-4b4a1ae0f04d", "140gsm drawing paper, ideal for mixed media."},
				{"stationery", "Hardcover Journal", "BK-010", "6220040000103", 1099, 450, 35, "photo-1531346878377-a5be20888e57", "Elastic closure, ribbon bookmark, 192 pages."},
			},
		},
		"mobile_shop": {
			categories: [][2]string{{"smartphones", "Smartphones"}, {"accessories", "Accessories"}, {"chargers", "Chargers"}},
			products: []demoProduct{
				{"smartphones", "iPhone 15 Pro", "MP-001", "6220050000018", 99900, 80000, 5, "photo-1591337676887-a217a6970a8a", "6.1in titanium, A17 Pro chip, 48MP camera."},
				{"smartphones", "Samsung Galaxy S24", "MP-002", "6220050000025", 79900, 65000, 6, "photo-1610945265064-0e34e5519bbf", "6.2in AMOLED 120Hz with Galaxy AI."},
				{"smartphones", "Google Pixel 8", "MP-003", "6220050000032", 69900, 56000, 4, "photo-1598327105666-5b89351aff97", "Tensor G3, best-in-class phone camera."},
				{"smartphones", "Xiaomi Redmi Note 13", "MP-004", "6220050000049", 29900, 24000, 8, "photo-1511707171634-5f897ff02aa9", "120Hz AMOLED and 108MP main camera."},
				{"accessories", "Bluetooth Earbuds Pro", "MP-005", "6220050000056", 4900, 2500, 25, "photo-1590658268037-6bf12165a8df", "Active noise canceling with wireless case."},
				{"accessories", "Smart Watch GT4", "MP-006", "6220050000063", 12900, 8000, 15, "photo-1579586337278-3befd40fd17a", "AMOLED display, GPS, 7-day battery."},
				{"accessories", "Silicone Case iPhone 15", "MP-007", "6220050000070", 1999, 500, 40, "photo-1601593346740-925612772716", "Shock-absorbing edge, camera-height lip."},
				{"chargers", "USB-C Fast Charger 65W", "MP-008", "6220050000087", 3499, 1200, 28, "photo-1583863788434-e58a36330cf0", "GaN charger for phone and laptop."},
				{"chargers", "MagSafe Wireless Pad", "MP-009", "6220050000094", 3999, 1500, 20, "photo-1586953208448-b95a79798f07", "15W magnetic fast charge for iPhone."},
				{"chargers", "USB-C Cable 2m Braided", "MP-010", "6220050000100", 599, 250, 60, "photo-1615529182904-14819c35db37", "100W PD-rated braided nylon cable."},
			},
		},
		"computer_shop": {
			categories: [][2]string{{"laptops", "Laptops"}, {"components", "Components"}, {"peripherals", "Peripherals"}},
			products: []demoProduct{
				{"laptops", `MacBook Pro 14"`, "CP-001", "6220060000015", 199900, 160000, 4, "photo-1496181133206-80ce9b88a853", "M3 Pro, 18GB unified memory, 512GB SSD."},
				{"laptops", `Dell XPS 13`, "CP-002", "6220060000022", 129900, 100000, 5, "photo-1525547719571-a2d4ac8945e2", "13.4in InfinityEdge display, ultra-portable."},
				{"laptops", `Lenovo ThinkPad X1`, "CP-003", "6220060000039", 149900, 115000, 3, "photo-1541807084-5c52b6b3adef", "Business-grade durability with long battery."},
				{"components", "SSD NVMe 1TB", "CP-004", "6220060000046", 10999, 7000, 30, "photo-1597852074816-d933c7d2b988", "Gen4 PCIe, 7000MB/s read, 5yr warranty."},
				{"components", "RAM 32GB DDR5", "CP-005", "6220060000053", 9999, 6500, 28, "photo-1590987093287-e5b0d1c80a7f", "5200MHz dual-channel kit for creators."},
				{"components", "RTX 4070 Graphics Card", "CP-006", "6220060000060", 59900, 45000, 5, "photo-1591488320449-011701bb6704", "12GB GDDR6X, 4K gaming and ray tracing."},
				{"peripherals", `27" 4K Monitor`, "CP-007", "6220060000077", 32900, 25000, 8, "photo-1527443224154-c4a3942d3acf", "IPS panel, 99% sRGB, height-adjustable stand."},
				{"peripherals", "Mechanical Keyboard RGB", "CP-008", "6220060000084", 8999, 4000, 20, "photo-1587829741301-dc798b83add3", "Hot-swappable switches with per-key RGB."},
				{"peripherals", "Gaming Mouse Pro", "CP-009", "6220060000091", 5999, 2800, 25, "photo-1527864550417-7fd91fc51a46", "26K DPI optical sensor, 8 programmable buttons."},
				{"peripherals", "USB Headset", "CP-010", "6220060000107", 3999, 2000, 18, "photo-1618366712010-f4ae9c647dcb", "Noise-canceling mic, cushioned ear pads."},
			},
		},
		"grocery": {
			categories: [][2]string{{"produce", "Produce"}, {"dairy", "Dairy"}, {"bakery", "Bakery"}, {"snacks", "Snacks"}},
			products: []demoProduct{
				{"produce", "Apples 1kg", "GR-001", "6220070000012", 299, 120, 60, "photo-1560806887-1e4cd0b6cbd6", "Crisp red apples, grown in Upper Egypt."},
				{"produce", "Bananas 1kg", "GR-002", "6220070000029", 199, 80, 70, "photo-1571771894821-ce9b6c11b08e", "Sweet ripe bananas, farm-fresh daily."},
				{"produce", "Tomatoes 1kg", "GR-003", "6220070000036", 249, 100, 55, "photo-1546470427-e26264be01bf", "Vine-ripened plum tomatoes for salads and sauces."},
				{"dairy", "Whole Milk 1L", "GR-004", "6220070000043", 349, 200, 45, "photo-1550583724-b2692b85b150", "Fresh pasteurized whole milk."},
				{"dairy", "Cheddar Cheese 8oz", "GR-005", "6220070000050", 499, 300, 18, "photo-1486297678162-eb2a19b0a32d", "Sharp cheddar aged 12 months."},
				{"dairy", "Greek Yogurt 500g", "GR-006", "6220070000067", 399, 220, 25, "photo-1571212515416-fef01fc43637", "Thick and creamy, high protein."},
				{"bakery", "Sourdough Loaf", "GR-007", "6220070000074", 499, 180, 20, "photo-1509440159596-0249088772ff", "Slow-fermented artisan sourdough boule."},
				{"bakery", "Whole Wheat Bread", "GR-008", "6220070000081", 399, 150, 30, "photo-1549931319-a545dcf3bc73", "Fresh-baked multigrain loaf, no preservatives."},
				{"snacks", "Trail Mix 200g", "GR-009", "6220070000098", 599, 300, 35, "photo-1599599810769-bcde5a160d32", "Roasted nuts, seeds and dried fruit."},
				{"snacks", "Orange Juice 1L", "GR-010", "6220070000104", 499, 250, 30, "photo-1600271886742-f049cd451bba", "100% pressed orange juice, chilled."},
			},
		},
	}
}

// seedCatalog inserts the demo categories + products for a business type inside
// the caller's transaction. Requires app.current_tenant to be the new tenant.
func seedCatalog(ctx context.Context, tx pgx.Tx, tenantID, businessType string) error {
	vertical, ok := seedCatalogs()[businessType]
	if !ok {
		return nil
	}
	categoryIDs := make(map[string]string, len(vertical.categories))
	for _, cat := range vertical.categories {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO categories (tenant_id, name, slug)
			VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`, tenantID, cat[1], cat[0]).Scan(&id)
		if err != nil {
			return err
		}
		categoryIDs[cat[0]] = id
	}
	for _, p := range vertical.products {
		categoryID, ok := categoryIDs[p.category]
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity, image_url, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'EGP', $8, $9, $10)`,
			tenantID, categoryID, p.name, p.sku, p.barcode, p.priceMinor, p.costMinor, p.stock,
			"https://images.unsplash.com/"+p.imageURL+"?auto=format&fit=crop&w=400&q=60", p.description); err != nil {
			return err
		}
	}
	return nil
}
