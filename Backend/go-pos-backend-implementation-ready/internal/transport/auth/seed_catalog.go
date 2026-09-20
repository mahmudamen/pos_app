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
	unit        string // units.code; empty means “piece”
}

type demoVertical struct {
	categories [][2]string // {slug, name}
	products   []demoProduct
}

// seedCatalogs returns one curated demo catalog per supported business type:
// local-Egyptian products and EGP prices, EAN-13 barcodes (622-prefix GS1
// Egypt), Unsplash image URLs and a one-line info text. Products reference
// "categories" lists above them; each vertical gets its own barcode number
// space so a tenant's codes are unique. The grocery vertical uses the EXACT
// barcodes from the bundled egypt_price_catalog so the price-refresh job can
// re-price it over time.
func seedCatalogs() map[string]demoVertical {
	return map[string]demoVertical{
		"coffee_shop": {
			categories: [][2]string{{"drinks", "Drinks"}, {"food", "Food"}, {"pastries", "Pastries"}},
			products: []demoProduct{
				{"drinks", "Espresso", "CF-001", "6220010000010", 280, 80, 100, "photo-1572442388796-11668a67e53d", "Bold single-origin shot, roasted in-house daily.", "piece"},
				{"drinks", "Cappuccino", "CF-002", "6220010000027", 350, 100, 80, "photo-1572442388796-11668a67e53d", "Espresso with steamed-milk foam and cocoa dusting.", "piece"},
				{"drinks", "Caffe Latte", "CF-003", "6220010000034", 350, 110, 80, "photo-1572442388796-11668a67e53d", "Silky steamed milk over a double espresso.", "piece"},
				{"drinks", "Iced Americano", "CF-004", "6220010000041", 300, 70, 60, "photo-1572442388796-11668a67e53d", "Chilled double espresso on fresh ice.", "piece"},
				{"drinks", "Mocha", "CF-005", "6220010000058", 400, 130, 50, "photo-1572442388796-11668a67e53d", "Espresso, dark chocolate and steamed milk.", "piece"},
				{"food", "Avocado Toast", "CF-006", "6220010000065", 650, 260, 25, "photo-1572442388796-11668a67e53d", "Sourdough, smashed avocado, poached egg, chili flakes.", "piece"},
				{"pastries", "Butter Croissant", "CF-007", "6220010000072", 250, 80, 40, "photo-1572442388796-11668a67e53d", "Flaky laminated dough baked fresh every morning.", "piece"},
				{"pastries", "Blueberry Muffin", "CF-008", "6220010000089", 300, 90, 35, "photo-1572442388796-11668a67e53d", "Soft muffin loaded with wild blueberries.", "piece"},
				{"pastries", "Chocolate Chip Cookie", "CF-009", "6220010000096", 220, 70, 50, "photo-1572442388796-11668a67e53d", "Chewy center, crisp edge, Belgian chocolate.", "piece"},
				{"pastries", "Cheesecake Slice", "CF-010", "6220010000102", 420, 140, 20, "photo-1572442388796-11668a67e53d", "Baked New York style with a graham crust.", "piece"},
			},
		},
		"restaurant": {
			categories: [][2]string{{"starters", "Starters"}, {"mains", "Main Courses"}, {"desserts", "Desserts"}},
			products: []demoProduct{
				{"starters", "Caesar Salad", "RS-001", "6220020000017", 480, 150, 25, "photo-1572442388796-11668a67e53d", "Romaine, parmesan, croutons and garlic dressing.", "piece"},
				{"starters", "Tomato Soup", "RS-002", "6220020000024", 350, 120, 30, "photo-1572442388796-11668a67e53d", "Roasted tomatoes, basil and a swirl of cream.", "piece"},
				{"mains", "Grilled Chicken Sandwich", "RS-003", "6220020000031", 550, 200, 30, "photo-1572442388796-11668a67e53d", "Chargrilled chicken, garlic mayo, fresh greens.", "piece"},
				{"mains", "Pasta Carbonara", "RS-004", "6220020000048", 650, 250, 20, "photo-1572442388796-11668a67e53d", "Guanciale, egg yolk, pecorino on fresh tagliatelle.", "piece"},
				{"mains", "Classic Beef Burger", "RS-005", "6220020000055", 620, 260, 30, "photo-1572442388796-11668a67e53d", "Angus patty, cheddar, tomato and house sauce.", "piece"},
				{"mains", "Margherita Pizza", "RS-006", "6220020000062", 850, 350, 15, "photo-1572442388796-11668a67e53d", "San Marzano tomato, mozzarella, fresh basil.", "piece"},
				{"mains", "Grilled Steak", "RS-007", "6220020000079", 999, 600, 12, "photo-1572442388796-11668a67e53d", "300g ribeye seared over open flame.", "piece"},
				{"mains", "Fresh Orange Juice", "RS-008", "6220020000086", 450, 180, 40, "photo-1572442388796-11668a67e53d", "Cold-pressed blood oranges, no added sugar.", "liter"},
				{"desserts", "Molten Lava Cake", "RS-009", "6220020000093", 520, 200, 18, "photo-1572442388796-11668a67e53d", "Dark chocolate cake with a warm gooey center.", "piece"},
				{"desserts", "Tiramisu", "RS-010", "6220020000109", 480, 180, 15, "photo-1572442388796-11668a67e53d", "Espresso-soaked savoiardi, mascarpone cream.", "piece"},
			},
		},
		"retail": {
			categories: [][2]string{{"essentials", "Essentials"}, {"electronics", "Electronics"}, {"clothing", "Clothing"}},
			products: []demoProduct{
				{"essentials", "Weekender Backpack", "RT-001", "6220030000014", 3900, 1800, 25, "photo-1572442388796-11668a67e53d", "Water-resistant 28L with padded laptop sleeve.", "piece"},
				{"essentials", "Sports Water Bottle", "RT-002", "6220030000021", 899, 400, 60, "photo-1572442388796-11668a67e53d", "BPA-free stainless steel, 750ml, leak-proof.", "piece"},
				{"essentials", "Polarized Sunglasses", "RT-003", "6220030000038", 2599, 1200, 20, "photo-1572442388796-11668a67e53d", "UV400 lenses with a lightweight poly frame.", "piece"},
				{"electronics", "Bluetooth Speaker", "RT-004", "6220030000045", 4999, 2800, 18, "photo-1572442388796-11668a67e53d", "Waterproof, 12-hour playtime, punchy bass.", "piece"},
				{"electronics", "Wireless Headphones", "RT-005", "6220030000052", 3999, 2200, 22, "photo-1572442388796-11668a67e53d", "Over-ear, active noise canceling, 30h battery.", "piece"},
				{"electronics", "Power Bank 20000mAh", "RT-006", "6220030000069", 4499, 2400, 30, "photo-1572442388796-11668a67e53d", "Dual USB-C fast charge for phone and tablet.", "piece"},
				{"clothing", "Cotton Crew T-Shirt", "RT-007", "6220030000076", 3200, 1400, 40, "photo-1572442388796-11668a67e53d", "Premium combed cotton, pre-shrunk regular fit.", "piece"},
				{"clothing", "Running Sneakers", "RT-008", "6220030000083", 5999, 3500, 15, "photo-1572442388796-11668a67e53d", "Breathable mesh upper with responsive midsole.", "piece"},
				{"clothing", "Denim Jacket", "RT-009", "6220030000090", 4999, 2800, 12, "photo-1572442388796-11668a67e53d", "Classic blue denim with button front.", "piece"},
				{"essentials", "Travel Mug 400ml", "RT-010", "6220030000106", 1299, 550, 35, "photo-1572442388796-11668a67e53d", "Double-wall vacuum keeps drinks hot for hours.", "piece"},
			},
		},
		"book_store": {
			categories: [][2]string{{"fiction", "Fiction"}, {"non-fiction", "Non-Fiction"}, {"stationery", "Stationery"}},
			products: []demoProduct{
				{"fiction", "The Great Gatsby", "BK-001", "6220040000011", 1299, 500, 25, "photo-1572442388796-11668a67e53d", "Fitzgerald's classic portrait of the Jazz Age.", "piece"},
				{"fiction", "1984", "BK-002", "6220040000028", 1099, 450, 30, "photo-1572442388796-11668a67e53d", "Orwell's dystopian masterpiece of surveillance.", "piece"},
				{"fiction", "Dune", "BK-003", "6220040000035", 1499, 600, 18, "photo-1572442388796-11668a67e53d", "Herbert's epic science-fiction universe.", "piece"},
				{"non-fiction", "Sapiens", "BK-004", "6220040000042", 1899, 750, 20, "photo-1572442388796-11668a67e53d", "A brief history of humankind by Yuval Noah Harari.", "piece"},
				{"non-fiction", "Atomic Habits", "BK-005", "6220040000059", 1599, 650, 28, "photo-1572442388796-11668a67e53d", "Tiny changes, remarkable results by James Clear.", "piece"},
				{"non-fiction", "Thinking, Fast and Slow", "BK-006", "6220040000066", 1999, 850, 15, "photo-1572442388796-11668a67e53d", "Kahneman's guide to the two systems of thought.", "piece"},
				{"stationery", "Notebook A5", "BK-007", "6220040000073", 399, 120, 100, "photo-1572442388796-11668a67e53d", "120 ruled pages, hardcover, lay-flat binding.", "piece"},
				{"stationery", "Gel Pen Set", "BK-008", "6220040000080", 599, 180, 80, "photo-1572442388796-11668a67e53d", "Five smooth 0.5mm gel pens in assorted inks.", "pack"},
				{"stationery", "A4 Sketchbook", "BK-009", "6220040000097", 799, 300, 45, "photo-1572442388796-11668a67e53d", "140gsm drawing paper, ideal for mixed media.", "piece"},
				{"stationery", "Hardcover Journal", "BK-010", "6220040000103", 1099, 450, 35, "photo-1572442388796-11668a67e53d", "Elastic closure, ribbon bookmark, 192 pages.", "piece"},
			},
		},
		"mobile_shop": {
			categories: [][2]string{{"smartphones", "Smartphones"}, {"accessories", "Accessories"}, {"chargers", "Chargers"}},
			products: []demoProduct{
				{"smartphones", "iPhone 15 Pro", "MP-001", "6220050000018", 99900, 80000, 5, "photo-1572442388796-11668a67e53d", "6.1in titanium, A17 Pro chip, 48MP camera.", "piece"},
				{"smartphones", "Samsung Galaxy S24", "MP-002", "6220050000025", 79900, 65000, 6, "photo-1572442388796-11668a67e53d", "6.2in AMOLED 120Hz with Galaxy AI.", "piece"},
				{"smartphones", "Google Pixel 8", "MP-003", "6220050000032", 69900, 56000, 4, "photo-1572442388796-11668a67e53d", "Tensor G3, best-in-class phone camera.", "piece"},
				{"smartphones", "Xiaomi Redmi Note 13", "MP-004", "6220050000049", 29900, 24000, 8, "photo-1572442388796-11668a67e53d", "120Hz AMOLED and 108MP main camera.", "piece"},
				{"accessories", "Bluetooth Earbuds Pro", "MP-005", "6220050000056", 4900, 2500, 25, "photo-1572442388796-11668a67e53d", "Active noise canceling with wireless case.", "piece"},
				{"accessories", "Smart Watch GT4", "MP-006", "6220050000063", 12900, 8000, 15, "photo-1572442388796-11668a67e53d", "AMOLED display, GPS, 7-day battery.", "piece"},
				{"accessories", "Silicone Case iPhone 15", "MP-007", "6220050000070", 1999, 500, 40, "photo-1572442388796-11668a67e53d", "Shock-absorbing edge, camera-height lip.", "piece"},
				{"chargers", "USB-C Fast Charger 65W", "MP-008", "6220050000087", 3499, 1200, 28, "photo-1572442388796-11668a67e53d", "GaN charger for phone and laptop.", "piece"},
				{"chargers", "MagSafe Wireless Pad", "MP-009", "6220050000094", 3999, 1500, 20, "photo-1572442388796-11668a67e53d", "15W magnetic fast charge for iPhone.", "piece"},
				{"chargers", "USB-C Cable 2m Braided", "MP-010", "6220050000100", 599, 250, 60, "photo-1572442388796-11668a67e53d", "100W PD-rated braided nylon cable.", "piece"},
			},
		},
		"computer_shop": {
			categories: [][2]string{{"laptops", "Laptops"}, {"components", "Components"}, {"peripherals", "Peripherals"}},
			products: []demoProduct{
				{"laptops", "MacBook Pro 14\"", "CP-001", "6220060000015", 199900, 160000, 4, "photo-1572442388796-11668a67e53d", "M3 Pro, 18GB unified memory, 512GB SSD.", "piece"},
				{"laptops", "Dell XPS 13", "CP-002", "6220060000022", 129900, 100000, 5, "photo-1572442388796-11668a67e53d", "13.4in InfinityEdge display, ultra-portable.", "piece"},
				{"laptops", "Lenovo ThinkPad X1", "CP-003", "6220060000039", 149900, 115000, 3, "photo-1572442388796-11668a67e53d", "Business-grade durability with long battery.", "piece"},
				{"components", "SSD NVMe 1TB", "CP-004", "6220060000046", 10999, 7000, 30, "photo-1572442388796-11668a67e53d", "Gen4 PCIe, 7000MB/s read, 5yr warranty.", "piece"},
				{"components", "RAM 32GB DDR5", "CP-005", "6220060000053", 9999, 6500, 28, "photo-1572442388796-11668a67e53d", "5200MHz dual-channel kit for creators.", "piece"},
				{"components", "RTX 4070 Graphics Card", "CP-006", "6220060000060", 59900, 45000, 5, "photo-1572442388796-11668a67e53d", "12GB GDDR6X, 4K gaming and ray tracing.", "piece"},
				{"peripherals", "27\" 4K Monitor", "CP-007", "6220060000077", 32900, 25000, 8, "photo-1572442388796-11668a67e53d", "IPS panel, 99% sRGB, height-adjustable stand.", "piece"},
				{"peripherals", "Mechanical Keyboard RGB", "CP-008", "6220060000084", 8999, 4000, 20, "photo-1572442388796-11668a67e53d", "Hot-swappable switches with per-key RGB.", "piece"},
				{"peripherals", "Gaming Mouse Pro", "CP-009", "6220060000091", 5999, 2800, 25, "photo-1572442388796-11668a67e53d", "26K DPI optical sensor, 8 programmable buttons.", "piece"},
				{"peripherals", "USB Headset", "CP-010", "6220060000107", 3999, 2000, 18, "photo-1572442388796-11668a67e53d", "Noise-canceling mic, cushioned ear pads.", "piece"},
			},
		},
		"grocery": {
			categories: [][2]string{{"dairy", "Dairy"}, {"beverages", "Beverages"}, {"pantry", "Pantry & Staples"}, {"snacks", "Snacks"}, {"fresh", "Fresh Produce"}, {"bakery", "Bakery"}},
			products: []demoProduct{
				{"dairy", "Juhayna Fresh Milk 1L", "GR-001", "6221010000017", 4200, 3800, 45, "photo-1572442388796-11668a67e53d", "Pasteurized fresh milk from Egyptian farms.", "liter"},
				{"dairy", "Juhayna Plain Yogurt 500g", "GR-002", "6221010000024", 2600, 2300, 40, "photo-1572442388796-11668a67e53d", "Thick plain yogurt, no added sugar.", "piece"},
				{"dairy", "Domty Cheddar Cheese 100g", "GR-003", "6221010000031", 5500, 5000, 30, "photo-1572442388796-11668a67e53d", "Mild yellow cheddar slices.", "piece"},
				{"dairy", "Panda Cheese Triangles", "GR-004", "6221010000048", 3400, 3000, 38, "photo-1572442388796-11668a67e53d", "Portioned cream cheese triangles.", "piece"},
				{"beverages", "Coca-Cola 1L", "GR-005", "6221010000062", 2400, 2000, 60, "photo-1572442388796-11668a67e53d", "Chilled classic cola, 1 liter bottle.", "liter"},
				{"beverages", "Schweppes Lemon 1L", "GR-006", "6221010000086", 2600, 2200, 45, "photo-1572442388796-11668a67e53d", "Sparkling lemon-lime soft drink.", "liter"},
				{"beverages", "Mineral Water 1.5L", "GR-007", "6221010000093", 1500, 1100, 90, "photo-1572442388796-11668a67e53d", "Natural drinking water, 1.5 liter bottle.", "piece"},
				{"beverages", "Syrup Mango Juice 1L", "GR-008", "6221010000109", 3800, 3200, 35, "photo-1572442388796-11668a67e53d", "Sweet mango nectar, chilled.", "liter"},
				{"pantry", "Lipton Yellow Label Tea 50g", "GR-009", "6221010000116", 6500, 5800, 25, "photo-1572442388796-11668a67e53d", "Classic black teabags, 50g box.", "piece"},
				{"pantry", "Nescafe Classic 100g", "GR-010", "6221010000123", 14500, 13200, 20, "photo-1572442388796-11668a67e53d", "Instant roasted coffee, 100g jar.", "piece"},
				{"pantry", "El-Mashreq Sugar 1kg", "GR-011", "6221010000147", 3100, 2900, 70, "photo-1572442388796-11668a67e53d", "Refined white sugar, 1 kilogram bag.", "kg"},
				{"pantry", "El-Gomhoria Rice 1kg", "GR-012", "6221010000154", 4200, 3800, 65, "photo-1572442388796-11668a67e53d", "Egyptian short-grain rice, 1kg bag.", "kg"},
				{"pantry", "Al-Shark Pasta 400g", "GR-013", "6221010000178", 1800, 1500, 55, "photo-1572442388796-11668a67e53d", "Durum wheat pasta, 400g pack.", "piece"},
				{"pantry", "Mazola Corn Oil 1L", "GR-014", "6221010000185", 13500, 12500, 25, "photo-1572442388796-11668a67e53d", "Pure corn oil, light for frying.", "liter"},
				{"pantry", "El-Nasr Table Salt 1kg", "GR-015", "6221010000208", 800, 500, 80, "photo-1572442388796-11668a67e53d", "Fine iodized table salt, 1kg.", "kg"},
				{"pantry", "Brown Lentils 1kg", "GR-016", "6221010000215", 5800, 5200, 40, "photo-1572442388796-11668a67e53d", "Whole brown lentils, 1kg bag.", "kg"},
				{"snacks", "Chipsy Chips 80g", "GR-017", "6221010000239", 1600, 1300, 75, "photo-1572442388796-11668a67e53d", "Classic salted potato chips.", "piece"},
				{"snacks", "Lays Chips 80g", "GR-018", "6221010000246", 1700, 1400, 70, "photo-1572442388796-11668a67e53d", "Thin golden potato chips.", "piece"},
				{"snacks", "Oman Chips King", "GR-019", "6221010000253", 1200, 950, 90, "photo-1572442388796-11668a67e53d", "Egyptian spicy chipitos staple.", "piece"},
				{"snacks", "Stella Biscuits 400g", "GR-020", "6221010000307", 3500, 3000, 35, "photo-1572442388796-11668a67e53d", "Butter cookies, 400g pack.", "piece"},
				{"fresh", "Tomatoes 1kg", "GR-021", "6220070000104", 1600, 1100, 55, "photo-1572442388796-11668a67e53d", "Vine-ripened plum tomatoes.", "kg"},
				{"fresh", "Potatoes 1kg", "GR-022", "6220070000111", 1300, 900, 80, "photo-1572442388796-11668a67e53d", "Farm-fresh field potatoes.", "kg"},
				{"fresh", "Bananas 1kg", "GR-023", "6220070000289", 2800, 2100, 50, "photo-1572442388796-11668a67e53d", "Sweet ripe bananas.", "kg"},
				{"fresh", "Onions 1kg", "GR-024", "6220070000357", 1800, 1300, 70, "photo-1572442388796-11668a67e53d", "Medium yellow onions.", "kg"},
				{"fresh", "Strawberries 1kg", "GR-025", "6220070000425", 3500, 2600, 25, "photo-1572442388796-11668a67e53d", "Fresh Egyptian strawberries.", "kg"},
				{"bakery", "Baladi Bread (8 pcs)", "GR-026", "6220070000593", 900, 500, 120, "photo-1572442388796-11668a67e53d", "Traditional whole-wheat flatbread.", "piece"},
			},
		},
		"bakery": {
			categories: [][2]string{{"bread", "Bread"}, {"cakes", "Cakes"}, {"pastries", "Pastries"}},
			products: []demoProduct{
				{"bread", "Baladi Bread (8 pcs)", "BK-001", "6220080000019", 900, 500, 120, "photo-1572442388796-11668a67e53d", "Whole-wheat flatbread, fresh daily.", "piece"},
				{"bread", "Fino Rolls (5 pcs)", "BK-002", "6220080000026", 1000, 600, 100, "photo-1572442388796-11668a67e53d", "Soft white bread rolls.", "piece"},
				{"pastries", "Butter Croissant", "BK-003", "6220080000033", 450, 200, 40, "photo-1572442388796-11668a67e53d", "Flaky butter croissant, baked each morning.", "piece"},
				{"pastries", "Zaatar Manaeesh", "BK-004", "6220080000040", 700, 350, 45, "photo-1572442388796-11668a67e53d", "Thyme-topped baked dough flatbread.", "piece"},
				{"pastries", "Cheese Pie (Fatayer)", "BK-005", "6220080000057", 800, 400, 40, "photo-1572442388796-11668a67e53d", "Melted white cheese folded in dough.", "piece"},
				{"cakes", "Chocolate Cake Slice", "BK-006", "6220080000064", 950, 500, 25, "photo-1572442388796-11668a67e53d", "Moist chocolate sponge with ganache.", "piece"},
				{"cakes", "Basbousa Slice", "BK-007", "6220080000071", 600, 300, 30, "photo-1572442388796-11668a67e53d", "Semolina cake soaked in syrup.", "piece"},
				{"cakes", "Pound Cake Loaf", "BK-008", "6220080000088", 1200, 650, 18, "photo-1572442388796-11668a67e53d", "Buttery vanilla loaf, warmed by the slice.", "piece"},
				{"pastries", "Ghorayeba Cookies 500g", "BK-009", "6220080000095", 1600, 900, 22, "photo-1572442388796-11668a67e53d", "Melt-in-your-mouth shortbread cookies.", "box"},
				{"pastries", "Baklava Tray 500g", "BK-010", "6220080000101", 2500, 1400, 15, "photo-1572442388796-11668a67e53d", "Flaky phyllo, nuts and honey syrup.", "box"},
			},
		},
		"shawerma": {
			categories: [][2]string{{"shawerma", "Shawerma"}, {"grills", "Grills"}, {"sides", "Sides"}},
			products: []demoProduct{
				{"shawerma", "Chicken Shawerma Sandwich", "SW-001", "6220090000016", 850, 400, 60, "photo-1572442388796-11668a67e53d", "Marinated chicken, garlic sauce, pickles.", "piece"},
				{"shawerma", "Beef Shawerma Sandwich", "SW-002", "6220090000023", 1100, 550, 50, "photo-1572442388796-11668a67e53d", "Sliced beef shawerma with tahini.", "piece"},
				{"shawerma", "Mixed Shawerma Plate", "SW-003", "6220090000030", 2500, 1300, 25, "photo-1572442388796-11668a67e53d", "Chicken and beef with rice and salad.", "piece"},
				{"grills", "Chicken Taouk Plate", "SW-004", "6220090000047", 1800, 900, 30, "photo-1572442388796-11668a67e53d", "Grilled skewers with garlic dip.", "piece"},
				{"grills", "Grilled Chicken Quarter", "SW-005", "6220090000054", 1300, 650, 35, "photo-1572442388796-11668a67e53d", "Charcoal-grilled, with fries.", "piece"},
				{"grills", "Kofta Plate", "SW-006", "6220090000061", 2200, 1100, 28, "photo-1572442388796-11668a67e53d", "Minced beef kofta with grilled tomatoes.", "piece"},
				{"grills", "Kebab Plate", "SW-007", "6220090000078", 2600, 1400, 20, "photo-1572442388796-11668a67e53d", "Lamb kebab on flatbread.", "piece"},
				{"sides", "French Fries 250g", "SW-008", "6220090000085", 600, 250, 70, "photo-1572442388796-11668a67e53d", "Crispy golden fries.", "piece"},
				{"sides", "Pickles Box", "SW-009", "6220090000092", 350, 150, 80, "photo-1572442388796-11668a67e53d", "Mixed Egyptian pickles.", "box"},
				{"sides", "Baladi Salad", "SW-010", "6220090000108", 400, 180, 60, "photo-1572442388796-11668a67e53d", "Tomato, cucumber and onion salad.", "piece"},
			},
		},
		"falafel": {
			categories: [][2]string{{"breakfast", "Breakfast"}, {"sandwiches", "Sandwiches"}, {"sides", "Sides"}},
			products: []demoProduct{
				{"breakfast", "Taameya (1 pc)", "FL-001", "6220100000012", 150, 50, 200, "photo-1572442388796-11668a67e53d", "Crispy fava-bean patties.", "piece"},
				{"breakfast", "Foul Medames Bowl", "FL-002", "6220100000029", 450, 200, 80, "photo-1572442388796-11668a67e53d", "Slow-cooked fava beans, cumin and oil.", "piece"},
				{"sandwiches", "Falafel Sandwich", "FL-003", "6220100000036", 600, 250, 90, "photo-1572442388796-11668a67e53d", "Taameya with tahini in baladi bread.", "piece"},
				{"sandwiches", "Foul + Falafel Sandwich", "FL-004", "6220100000043", 750, 350, 70, "photo-1572442388796-11668a67e53d", "Foul, taameya, salad and tahini.", "piece"},
				{"breakfast", "Shakshuka Plate", "FL-005", "6220100000050", 1200, 600, 30, "photo-1572442388796-11668a67e53d", "Eggs poached in spiced tomato sauce.", "piece"},
				{"sandwiches", "Macarona Bechamel", "FL-006", "6220100000067", 1400, 700, 25, "photo-1572442388796-11668a67e53d", "Oven-baked pasta with white sauce.", "piece"},
				{"sandwiches", "Koshari Plate", "FL-007", "6220100000074", 1000, 500, 40, "photo-1572442388796-11668a67e53d", "Rice, lentils, pasta, crispy onions.", "piece"},
				{"sides", "Fries 200g", "FL-008", "6220100000081", 400, 150, 60, "photo-1572442388796-11668a67e53d", "Golden fries with amba.", "piece"},
				{"sides", "Hummus Bowl", "FL-009", "6220100000098", 650, 300, 35, "photo-1572442388796-11668a67e53d", "Smooth chickpea dip with olive oil.", "piece"},
				{"sides", "Salad Bowl", "FL-010", "6220100000104", 350, 140, 55, "photo-1572442388796-11668a67e53d", "Fresh chopped salad.", "piece"},
			},
		},
		"pharmacy": {
			categories: [][2]string{{"health", "Health"}, {"care", "Care"}, {"baby", "Baby"}},
			products: []demoProduct{
				{"health", "Panadol Tablets 24", "PH-001", "6220110000019", 4500, 3900, 40, "photo-1572442388796-11668a67e53d", "Pain relief tablets, 24 blister.", "piece"},
				{"health", "Aspirin 500mg 20", "PH-002", "6220110000026", 2500, 2100, 35, "photo-1572442388796-11668a67e53d", "Low-dose aspirin, 20 tablets.", "piece"},
				{"health", "Vitamin C Effervescent 20", "PH-003", "6220110000033", 3200, 2700, 30, "photo-1572442388796-11668a67e53d", "Orange-flavored vitamin C tablets.", "piece"},
				{"health", "Cough Syrup 150ml", "PH-004", "6220110000040", 5500, 4700, 25, "photo-1572442388796-11668a67e53d", "Soothing cough mixture.", "piece"},
				{"health", "Paracetamol Syrup 120ml", "PH-005", "6220110000057", 3500, 3000, 28, "photo-1572442388796-11668a67e53d", "Children's paracetamol suspension.", "piece"},
				{"care", "Sterile Gauze Roll", "PH-006", "6220110000064", 850, 600, 50, "photo-1572442388796-11668a67e53d", "Absorbent gauze for dressing.", "piece"},
				{"care", "Alcohol 500ml", "PH-007", "6220110000071", 1900, 1500, 40, "photo-1572442388796-11668a67e53d", "Isopropyl rubbing alcohol.", "piece"},
				{"care", "Gloves (50 pcs)", "PH-008", "6220110000088", 1600, 1200, 45, "photo-1572442388796-11668a67e53d", "Powder-free nitrile gloves.", "box"},
				{"baby", "Baby Wet Wipes 80", "PH-009", "6220110000095", 3800, 3200, 38, "photo-1572442388796-11668a67e53d", "Gentle fragrance-free wipes.", "pack"},
				{"baby", "Diapers Size M (30)", "PH-010", "6220110000101", 14500, 13000, 20, "photo-1572442388796-11668a67e53d", "Medium diapers, pack of 30.", "piece"},
			},
		},
		"butcher": {
			categories: [][2]string{{"beef", "Beef"}, {"chicken", "Chicken"}, {"cuts", "Cuts"}},
			products: []demoProduct{
				{"beef", "Lean Beef 1kg", "BT-001", "6220120000016", 45000, 41000, 25, "photo-1572442388796-11668a67e53d", "Fresh local beef cuts.", "kg"},
				{"beef", "Boneless Veal 1kg", "BT-002", "6220120000023", 49000, 45000, 20, "photo-1572442388796-11668a67e53d", "Tender veal, boneless.", "kg"},
				{"beef", "Minced Beef 1kg", "BT-003", "6220120000030", 42000, 38000, 22, "photo-1572442388796-11668a67e53d", "Freshly ground lean beef.", "kg"},
				{"beef", "Beef Sirloin 1kg", "BT-004", "6220120000047", 56000, 51000, 15, "photo-1572442388796-11668a67e53d", "Prime sirloin steak.", "kg"},
				{"cuts", "Lamb Shoulder 1kg", "BT-005", "6220120000054", 52000, 48000, 12, "photo-1572442388796-11668a67e53d", "Succulent lamb shoulder.", "kg"},
				{"chicken", "Whole Chicken 1kg", "BT-006", "6220120000061", 10500, 9000, 40, "photo-1572442388796-11668a67e53d", "Fresh whole chicken.", "kg"},
				{"chicken", "Chicken Breast 1kg", "BT-007", "6220120000078", 13500, 12000, 35, "photo-1572442388796-11668a67e53d", "Boneless skinless breast fillet.", "kg"},
				{"chicken", "Chicken Liver 1kg", "BT-008", "6220120000085", 12500, 11000, 20, "photo-1572442388796-11668a67e53d", "Clean trimmed chicken livers.", "kg"},
				{"cuts", "Beef Sausage 1kg", "BT-009", "6220120000092", 28000, 25000, 18, "photo-1572442388796-11668a67e53d", "Spiced Egyptian beef sausage.", "kg"},
				{"cuts", "Shish Tawook 1kg", "BT-010", "6220120000108", 32000, 28000, 16, "photo-1572442388796-11668a67e53d", "Marinated chicken cubes, ready to grill.", "kg"},
			},
		},
		"fruits_veg": {
			categories: [][2]string{{"vegetables", "Vegetables"}, {"fruits", "Fruits"}, {"herbs", "Herbs"}},
			products: []demoProduct{
				{"vegetables", "Tomatoes 1kg", "FV-001", "6220130000013", 1600, 1100, 60, "photo-1572442388796-11668a67e53d", "Vine-ripened plum tomatoes.", "kg"},
				{"vegetables", "Potatoes 1kg", "FV-002", "6220130000020", 1300, 900, 80, "photo-1572442388796-11668a67e53d", "Farm-fresh field potatoes.", "kg"},
				{"vegetables", "Onions 1kg", "FV-003", "6220130000037", 1800, 1300, 70, "photo-1572442388796-11668a67e53d", "Medium yellow onions.", "kg"},
				{"vegetables", "Cucumbers 1kg", "FV-004", "6220130000044", 1400, 950, 55, "photo-1572442388796-11668a67e53d", "Crisp salad cucumbers.", "kg"},
				{"fruits", "Bananas 1kg", "FV-005", "6220130000051", 2800, 2100, 50, "photo-1572442388796-11668a67e53d", "Sweet ripe bananas.", "kg"},
				{"fruits", "Oranges 1kg", "FV-006", "6220130000068", 1500, 1000, 65, "photo-1572442388796-11668a67e53d", "Juicy local oranges.", "kg"},
				{"fruits", "Strawberries 1kg", "FV-007", "6220130000075", 3500, 2600, 25, "photo-1572442388796-11668a67e53d", "Fresh Egyptian strawberries.", "kg"},
				{"fruits", "Lemons 1kg", "FV-008", "6220130000082", 1200, 800, 45, "photo-1572442388796-11668a67e53d", "Sour lemons for juice and cooking.", "kg"},
				{"herbs", "Fresh Coriander Bunch", "FV-009", "6220130000099", 500, 250, 40, "photo-1572442388796-11668a67e53d", "Tied fresh coriander bunches.", "piece"},
				{"herbs", "Mint Bunch", "FV-010", "6220130000105", 600, 300, 35, "photo-1572442388796-11668a67e53d", "Fresh spearmint bunches.", "piece"},
			},
		},
		"clothing": {
			categories: [][2]string{{"men", "Men"}, {"women", "Women"}, {"kids", "Kids"}},
			products: []demoProduct{
				{"men", "Cotton T-Shirt", "CL-001", "6220140000010", 30000, 21000, 30, "photo-1572442388796-11668a67e53d", "Premium soft cotton tee.", "piece"},
				{"men", "Men's Jeans", "CL-002", "6220140000027", 65000, 45000, 18, "photo-1572442388796-11668a67e53d", "Classic straight-fit denim.", "piece"},
				{"men", "Men's Polo Shirt", "CL-003", "6220140000034", 38000, 27000, 22, "photo-1572442388796-11668a67e53d", "Breathable pique polo.", "piece"},
				{"women", "Women's Dress", "CL-004", "6220140000041", 85000, 58000, 15, "photo-1572442388796-11668a67e53d", "Elegant summer dress.", "piece"},
				{"women", "Hijab 3-Pack", "CL-005", "6220140000058", 25000, 16000, 25, "photo-1572442388796-11668a67e53d", "Soft viscose hijabs.", "pack"},
				{"women", "Women's Blouse", "CL-006", "6220140000065", 45000, 31000, 20, "photo-1572442388796-11668a67e53d", "Smart casual blouse.", "piece"},
				{"kids", "Kids Set (Shirt + Pants)", "CL-007", "6220140000072", 32000, 22000, 25, "photo-1572442388796-11668a67e53d", "Cotton set for kids.", "piece"},
				{"kids", "Baby Onesie", "CL-008", "6220140000089", 28000, 19000, 28, "photo-1572442388796-11668a67e53d", "Soft newborn onesie.", "piece"},
				{"men", "Winter Jacket", "CL-009", "6220140000096", 120000, 85000, 8, "photo-1572442388796-11668a67e53d", "Insulated winter jacket.", "piece"},
				{"women", "Sports Leggings", "CL-010", "6220140000102", 40000, 28000, 20, "photo-1572442388796-11668a67e53d", "High-waist active leggings.", "piece"},
			},
		},
		"sweets": {
			categories: [][2]string{{"oriental", "Oriental"}, {"chocolates", "Chocolates"}, {"gifts", "Gifts"}},
			products: []demoProduct{
				{"oriental", "Konafa 500g", "SW-001", "6220150000017", 4500, 2500, 20, "photo-1572442388796-11668a67e53d", "Golden syrup-soaked pastry.", "box"},
				{"oriental", "Basbousa 500g", "SW-002", "6220150000024", 3200, 1800, 25, "photo-1572442388796-11668a67e53d", "Semolina cake with syrup.", "box"},
				{"oriental", "Baklava 500g", "SW-003", "6220150000031", 5500, 3200, 18, "photo-1572442388796-11668a67e53d", "Nut-filled layers with honey.", "box"},
				{"oriental", "Ghraybeh Cookies 500g", "SW-004", "6220150000048", 2800, 1500, 22, "photo-1572442388796-11668a67e53d", "Butter shortbread cookies.", "box"},
				{"oriental", "Qatayef 6 pcs", "SW-005", "6220150000055", 2500, 1300, 26, "photo-1572442388796-11668a67e53d", "Stuffed pancake foldovers.", "box"},
				{"chocolates", "Chocolate Cake 1kg", "SW-006", "6220150000062", 9000, 5200, 10, "photo-1572442388796-11668a67e53d", "Layer chocolate gateau.", "box"},
				{"oriental", "Lokum (Turkish Delight) 500g", "SW-007", "6220150000079", 3500, 2000, 20, "photo-1572442388796-11668a67e53d", "Rose lokum with nuts.", "box"},
				{"gifts", "Roasted Nuts Mix 500g", "SW-008", "6220150000086", 6500, 4000, 15, "photo-1572442388796-11668a67e53d", "Cashew, almond and pistachio mix.", "box"},
				{"gifts", "Sweet House Mix 1kg", "SW-009", "6220150000093", 7500, 4500, 12, "photo-1572442388796-11668a67e53d", "Assorted oriental sweets box.", "box"},
				{"gifts", "Sambousa (50 pcs)", "SW-010", "6220150000109", 2000, 1100, 30, "photo-1572442388796-11668a67e53d", "Cheese-filled fried pastries.", "piece"},
			},
		},
		"jewelry": {
			categories: [][2]string{{"gold", "Gold"}, {"silver", "Silver"}, {"accessories", "Accessories"}},
			products: []demoProduct{
				{"gold", "21K Gold (per gram)", "JW-001", "6220160000014", 92000, 88000, 50, "photo-1572442388796-11668a67e53d", "21-karat Egyptian gold.", "g"},
				{"gold", "24K Gold (per gram)", "JW-002", "6220160000021", 105000, 101000, 40, "photo-1572442388796-11668a67e53d", "Pure 24-karat gold.", "g"},
				{"gold", "18K Gold (per gram)", "JW-003", "6220160000038", 79000, 75000, 35, "photo-1572442388796-11668a67e53d", "18-karat gold alloy.", "g"},
				{"silver", "925 Silver (per gram)", "JW-004", "6220160000045", 4200, 3800, 80, "photo-1572442388796-11668a67e53d", "Sterling silver 925.", "g"},
				{"gold", "Custom Gold Ring", "JW-005", "6220160000052", 250000, 230000, 6, "photo-1572442388796-11668a67e53d", "Made-to-order gold ring.", "piece"},
				{"gold", "Gold Chain", "JW-006", "6220160000069", 480000, 440000, 4, "photo-1572442388796-11668a67e53d", "Handcrafted gold chain.", "piece"},
				{"silver", "Silver Bracelet", "JW-007", "6220160000076", 15000, 12000, 15, "photo-1572442388796-11668a67e53d", "Sterling silver bangle.", "piece"},
				{"accessories", "Pearl Earrings", "JW-008", "6220160000083", 9000, 7000, 12, "photo-1572442388796-11668a67e53d", "Freshwater pearl drop earrings.", "piece"},
				{"accessories", "Gold Pendant", "JW-009", "6220160000090", 65000, 58000, 8, "photo-1572442388796-11668a67e53d", "Gold pendant with engraving.", "piece"},
				{"accessories", "Gift Box", "JW-010", "6220160000106", 1000, 400, 40, "photo-1572442388796-11668a67e53d", "Elegant jewelry gift box.", "box"},
			},
		},
		"hardware": {
			categories: [][2]string{{"tools", "Tools"}, {"electrical", "Electrical"}, {"paint", "Paint"}},
			products: []demoProduct{
				{"tools", "Hammer 500g", "HW-001", "6220170000011", 1800, 1000, 30, "photo-1572442388796-11668a67e53d", "Steel-head claw hammer.", "piece"},
				{"tools", "Screwdriver Set 6pc", "HW-002", "6220170000028", 4200, 2600, 20, "photo-1572442388796-11668a67e53d", "Insulated handle set.", "box"},
				{"tools", "Adjustable Wrench", "HW-003", "6220170000035", 3500, 2000, 22, "photo-1572442388796-11668a67e53d", "Chrome-vanadium wrench.", "piece"},
				{"tools", "Tape Measure 5m", "HW-004", "6220170000042", 1200, 600, 40, "photo-1572442388796-11668a67e53d", "Locking retractable tape.", "piece"},
				{"tools", "Drill Machine 13mm", "HW-005", "6220170000059", 45000, 32000, 8, "photo-1572442388796-11668a67e53d", "Variable-speed hammer drill.", "piece"},
				{"electrical", "Extension Cord 10m", "HW-006", "6220170000066", 2200, 1300, 25, "photo-1572442388796-11668a67e53d", "3-gang extension lead.", "piece"},
				{"electrical", "LED Bulb 12W", "HW-007", "6220170000073", 900, 450, 100, "photo-1572442388796-11668a67e53d", "Bright white LED lamp.", "piece"},
				{"paint", "Paint Brush Set", "HW-008", "6220170000080", 1500, 800, 35, "photo-1572442388796-11668a67e53d", "Bristle brush assortment.", "box"},
				{"paint", "Wall Paint 1L", "HW-009", "6220170000097", 7000, 4800, 15, "photo-1572442388796-11668a67e53d", "Interior emulsion paint.", "liter"},
				{"paint", "PVC Pipe 1m", "HW-010", "6220170000103", 800, 400, 60, "photo-1572442388796-11668a67e53d", "Schedule-40 PVC pipe.", "m"},
				{"tools", "Door Hinge Set", "HW-011", "6220170000110", 1100, 550, 26, "photo-1572442388796-11668a67e53d", "Brass hinges with screws.", "box"},
				{"tools", "Safety Gloves (12 pairs)", "HW-012", "6220170000127", 900, 450, 30, "photo-1572442388796-11668a67e53d", "Cut-resistant work gloves.", "pack"},
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
			INSERT INTO categories (tenant_id, name, slug, name_ar)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, slug) DO UPDATE SET name = EXCLUDED.name,
				name_ar = COALESCE(EXCLUDED.name_ar, categories.name_ar)
			RETURNING id`, tenantID, cat[1], cat[0], arabicCategoryNames[cat[0]]).Scan(&id)
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
		unit := p.unit
		if unit == "" {
			unit = "piece"
		}
		ar := arabicProducts[p.barcode]
		if _, err := tx.Exec(ctx, `
			INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity, image_url, description, unit, name_ar, description_ar)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'EGP', $8, $9, $10, $11, $12, $13)`,
			tenantID, categoryID, p.name, p.sku, p.barcode, p.priceMinor, p.costMinor, p.stock,
			"https://images.unsplash.com/"+p.imageURL+"?auto=format&fit=crop&w=400&q=60", p.description, unit,
			emptyString(ar[0]), emptyString(ar[1])); err != nil {
			return err
		}
	}
	return nil
}

func emptyString(s string) *string {
	t := s
	if t == "" {
		return nil
	}
	return &t
}
