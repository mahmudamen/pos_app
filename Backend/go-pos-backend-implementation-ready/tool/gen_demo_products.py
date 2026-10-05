#!/usr/bin/env python3
"""Generate a rich, realistic demo catalogue for the 5 demo tenants.

Real products at real Egyptian-market prices (EGP), bilingual EN/AR names,
valid EAN-13 barcodes, categories, cost prices and stock. Idempotent: every
row is upserted on its natural key, so it is safe to re-run -- which is what
the server cron job does daily to keep the demo shops stocked.

Products:  (sku_suffix, name, name_ar, category, price_egp, cost_egp, stock, unit)
Categories: (slug, name, name_ar)

Output (under .secrets/, gitignored):
  .secrets/testing/demo_products_seed.sql

Usage:
  python3 tool/gen_demo_products.py                  # write the SQL
  python3 tool/gen_demo_products.py --print          # also print a summary
  psql "$DATABASE_URL" -f .secrets/testing/demo_products_seed.sql
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

WORKSPACE = Path(__file__).resolve().parents[3]
OUT = WORKSPACE / ".secrets" / "testing" / "demo_products_seed.sql"
CURRENCY = "EGP"

# products.unit is a FK into the `units` reference table, which only carries
# these codes. Catalogue units are mapped onto the closest valid code and
# validated at generation time so a bad value fails here, not mid-INSERT.
VALID_UNITS = {"box", "dozen", "g", "kg", "liter", "m", "ml", "pack", "piece", "qm"}
UNIT_MAP = {
    "piece": "piece", "pack": "pack", "kg": "kg", "box": "box",
    "cup": "piece", "plate": "piece", "slice": "piece", "bowl": "piece",
    "copy": "piece",
    "glass": "piece", "scoop": "piece", "set": "pack",
}

# EAN-13: prefix 20 is the in-store/internal-use range, so demo barcodes never
# collide with a real manufacturer GTIN. The body must be exactly 12 digits --
# the 13th is the check digit.
BARCODE_PREFIX = 200_000_000_000


def ean13(seed: int) -> str:
    """Build a valid EAN-13 from a 12-digit seed (adds the check digit)."""
    body = str(seed)
    if len(body) != 12:
        raise ValueError(f"EAN seed must be 12 digits, got {len(body)}: {seed}")
    digits = [int(c) for c in body]
    # EAN-13 check digit: sum of odd positions (1-indexed) *1 + even *3
    total = sum(digits[0:12:2]) + 3 * sum(digits[1:12:2])
    return body + str((10 - total % 10) % 10)


# --------------------------------------------------------------------------
# 1. Restaurant -- Egyptian cafe + fast food
# --------------------------------------------------------------------------
RESTAURANT = {
    "categories": [
        ("hot-drinks", "Hot Drinks", "مشروبات ساخنة"),
        ("cold-drinks", "Cold Drinks", "مشروبات باردة"),
        ("breakfast", "Breakfast", "فطار"),
        ("mains", "Main Dishes", "أطباق رئيسية"),
        ("sandwiches", "Sandwiches & Rolls", "ساندوتشات و رول"),
        ("sides", "Sides & Appetizers", "أطباق جانبية"),
        ("desserts", "Desserts", "حلويات"),
    ],
    "products": [
        ("ESP-01", "Espresso", "إسبريسو", "hot-drinks", 55, 18, 120, "cup"),
        ("LAT-01", "Latte", "لاتيه", "hot-drinks", 75, 26, 100, "cup"),
        ("CAP-01", "Cappuccino", "كابتشينو", "hot-drinks", 75, 26, 100, "cup"),
        ("FLW-01", "Flat White", "فلات وايت", "hot-drinks", 80, 28, 80, "cup"),
        ("MOC-01", "Mocha", "موكا", "hot-drinks", 85, 31, 70, "cup"),
        ("AMR-01", "Americano", "أمريكانو", "hot-drinks", 60, 19, 110, "cup"),
        ("TUR-01", "Turkish Coffee", "قهوة تركية", "hot-drinks", 40, 12, 140, "cup"),
        ("CRD-01", "Cardamom Coffee", "قهوة هيل", "hot-drinks", 65, 21, 90, "cup"),
        ("TEA-01", "Black Tea", "شاي", "hot-drinks", 25, 6, 200, "cup"),
        ("MNT-01", "Mint Tea", "شاي بالنعناع", "hot-drinks", 35, 9, 180, "cup"),
        ("CHO-01", "Hot Chocolate", "شوكولاتة ساخنة", "hot-drinks", 70, 24, 60, "cup"),
        ("ICA-01", "Iced Americano", "أمريكانو مثلج", "cold-drinks", 65, 21, 90, "cup"),
        ("ILT-01", "Iced Latte", "لاتيه مثلج", "cold-drinks", 85, 30, 85, "cup"),
        ("ICP-01", "Ice Cappuccino", "كابتشينو مثلج", "cold-drinks", 85, 30, 80, "cup"),
        ("ORJ-01", "Fresh Orange Juice", "عصير برتقال طازج", "cold-drinks", 60, 22, 70, "glass"),
        ("LMN-01", "Lemon & Mint", "ليمون بالنعناع", "cold-drinks", 55, 18, 75, "glass"),
        ("MNG-01", "Mango Lassi", "لاتس مانجو", "cold-drinks", 70, 25, 60, "glass"),
        ("ITA-01", "Iced Tea", "شاي مثلج", "cold-drinks", 35, 10, 120, "glass"),
        ("MSH-01", "Chocolate Milkshake", " milkshake شوكولاتة", "cold-drinks", 90, 33, 50, "glass"),
        ("FRP-01", "Fried Potatoes", "بطاطس مقلي", "breakfast", 45, 14, 150, "plate"),
        ("OME-01", "Cheese Omelette", "أومليت جبن", "breakfast", 75, 27, 100, "plate"),
        ("FTR-01", "Feteer Meshaltet", "فتير مشطبت", "breakfast", 60, 20, 90, "piece"),
        ("EGG-01", "Eggs with Cheese", "بيض بالجبن", "breakfast", 85, 30, 80, "plate"),
        ("SHA-01", "Shakshuka", "شكشوكة", "breakfast", 95, 35, 60, "plate"),
        ("FUL-01", "Ful Medames", "فول مدمس", "breakfast", 40, 11, 200, "plate"),
        ("HAL-01", "Halloumi Plate", "جبنة حلومي", "breakfast", 110, 42, 50, "plate"),
        ("KSH-01", "Koshari", "كشري", "mains", 90, 30, 120, "plate"),
        ("FOL-01", "Foul Medames", "فول مدمس", "mains", 45, 13, 160, "plate"),
        ("MLK-01", "Molokhia with Chicken", "ملوخية بالدجاج", "mains", 110, 40, 70, "plate"),
        ("MKR-01", "Pasta with Tomato Sauce", "مكرونة بصوص الطماطم", "mains", 85, 28, 90, "plate"),
        ("KOF-01", "Grilled Kofta", "كفتة مشوية", "mains", 165, 62, 60, "plate"),
        ("MGR-01", "Mixed Grill Platter", "مشويات مشكلة", "mains", 320, 130, 40, "plate"),
        ("TAW-01", "Shish Tawook", "شيش طاووق", "mains", 190, 74, 50, "plate"),
        ("SAL-01", "Grilled Salmon", "سلمون مشوي", "mains", 340, 145, 25, "plate"),
        ("FCH-01", "Feteer with Cheese", "فتير بجبن", "sandwiches", 45, 14, 120, "piece"),
        ("HLW-01", "Hallouweese", "حلويز", "sandwiches", 65, 22, 100, "piece"),
        ("TAM-01", "Taameya", "طعمية", "sandwiches", 40, 12, 130, "piece"),
        ("SAG-01", "Sausage Roll", "رول سجق", "sandwiches", 55, 18, 80, "piece"),
        ("GCH-01", "Grilled Cheese", "جبنة مشوية", "sandwiches", 50, 16, 90, "piece"),
        ("FFR-01", "French Fries", "بطاطس محمر", "sides", 55, 17, 150, "plate"),
        ("RIC-01", "Egyptian Rice", "رز مصري", "sides", 40, 12, 170, "plate"),
        ("SAL-02", "Egyptian Salad", "سلطة مصرية", "sides", 45, 13, 120, "plate"),
        ("HUM-01", "Hummus", "حمص", "sides", 50, 15, 110, "plate"),
        ("MOB-01", "Moutabal", "متبل", "sides", 55, 17, 100, "plate"),
        ("UMA-01", "Umm Ali", "أم علي", "desserts", 90, 33, 60, "bowl"),
        ("RPD-01", "Rice Pudding", "رز باللبن", "desserts", 70, 24, 70, "bowl"),
        ("BAS-01", "Basbousa", "بسبوسة", "desserts", 65, 22, 80, "piece"),
        ("CHK-01", "New York Cheesecake", "تشيز كيك", "desserts", 95, 34, 50, "slice"),
        ("TIR-01", "Tiramisu", "تيراميسو", "desserts", 85, 30, 55, "slice"),
        ("ICE-01", "Ice Cream Scoop", "بوظة", "desserts", 40, 13, 90, "scoop"),
    ],
}

# --------------------------------------------------------------------------
# 2. Book store -- Arabic + English titles, stationery
# --------------------------------------------------------------------------
BOOK_STORE = {
    "categories": [
        ("arabic-fiction", "Arabic Fiction", "رواية عربية"),
        ("world-fiction", "World Fiction", "أدب عالمي"),
        ("business", "Business & Self Help", "أعمال وتطوير ذاتي"),
        ("programming", "Programming", "برمجة"),
        ("kids", "Kids & Young Readers", "أطفال"),
        ("science", "Science", "علوم"),
        ("stationery", "Stationery", "قرطاسية"),
    ],
    "products": [
        ("AR-01", "One Hundred Years of Solitude", "مئة عام من العزلة", "arabic-fiction", 285, 155, 24, "copy"),
        ("AR-02", "Bamboo Stalk", "ساق البامبو", "arabic-fiction", 240, 130, 18, "copy"),
        ("AR-03", "Granada Trilogy", "ثلاثية غرناطة", "arabic-fiction", 320, 175, 15, "copy"),
        ("AR-04", "Azazeel", "عزازيل", "arabic-fiction", 180, 95, 30, "copy"),
        ("AR-05", "The Open Door", "الباب المفتوح", "arabic-fiction", 210, 110, 22, "copy"),
        ("AR-06", "The Diwan of Ahmed Shawqi", "ديوان أحمد شوقي", "arabic-fiction", 260, 140, 12, "copy"),
        ("EN-01", "1984", "ألف وأربعمائة وأربعة وثمانين", "world-fiction", 220, 118, 35, "copy"),
        ("EN-02", "The Stranger", "الغريب", "world-fiction", 175, 92, 40, "copy"),
        ("EN-03", "The Prophet", "النبي", "world-fiction", 190, 100, 28, "copy"),
        ("EN-04", "The Old Man and the Sea", "العجوز والبحر", "world-fiction", 165, 86, 32, "copy"),
        ("BS-01", "Atomic Habits", "العادات الذرية", "business", 340, 185, 26, "copy"),
        ("BS-02", "Zero to One", "من صفر إلى واحد", "business", 365, 200, 20, "copy"),
        ("BS-03", "The Lean Startup", "الشركة الرشيقة", "business", 350, 190, 19, "copy"),
        ("BS-04", "Thinking, Fast and Slow", "التفكير السريع والبطيء", "business", 395, 215, 14, "copy"),
        ("BS-05", "Deep Work", "العمل العميق", "business", 310, 168, 23, "copy"),
        ("BS-06", "Rich Dad Poor Dad", "الأب الغني والأب الفقير", "business", 270, 145, 30, "copy"),
        ("PR-01", "Clean Code", "كود نظيف", "programming", 620, 340, 16, "copy"),
        ("PR-02", "Refactoring", "إعادة الهيكلة", "programming", 650, 360, 12, "copy"),
        ("PR-03", "Design Patterns", "أنماط التصميم", "programming", 690, 385, 10, "copy"),
        ("PR-04", "The Pragmatic Programmer", "المبرمج العملي", "programming", 580, 320, 14, "copy"),
        ("PR-05", "Learning SQL", "تعلم SQL", "programming", 540, 295, 18, "copy"),
        ("PR-06", "Fluent Python", "بايثون بطلاقة", "programming", 610, 335, 13, "copy"),
        ("PR-07", "JavaScript: The Definitive Guide", "دليل جافاسكربت الشامل", "programming", 670, 370, 11, "copy"),
        ("KD-01", "Harry Potter and the Philosopher's Stone", "هاري بوتر وحجر الفلاسفة", "kids", 295, 158, 25, "copy"),
        ("KD-02", "Diary of a Wimpy Kid", "مذكرات مراهق", "kids", 260, 138, 27, "copy"),
        ("KD-03", "Percy Jackson and the Lightning Thief", "بيرسي جاكسون", "kids", 285, 152, 21, "copy"),
        ("KD-04", "The Very Hungry Caterpillar", "اليرقة الجائعة", "kids", 150, 78, 40, "copy"),
        ("SC-01", "A Brief History of Time", "موجز تاريخ الزمن", "science", 330, 178, 17, "copy"),
        ("SC-02", "Cosmos", "كونفس", "science", 345, 185, 15, "copy"),
        ("SC-03", "Silent Spring", "الربيع الصامت", "science", 310, 165, 16, "copy"),
        ("SC-04", "The Selfish Gene", "الجين الأناني", "science", 355, 192, 13, "copy"),
        ("ST-01", "A5 Ruled Notebook", "كراسة A5 مسطرة", "stationery", 85, 38, 120, "piece"),
        ("ST-02", "Ballpoint Pens (Pack of 12)", "أقلام جاف (12)", "stationery", 60, 25, 150, "pack"),
        ("ST-03", "Highlighter Set (6 Colours)", "هايلايتر (6 ألوان)", "stationery", 110, 48, 90, "set"),
        ("ST-04", "Sticky Notes Assorted", "ورق ملاحظ لاصق", "stationery", 55, 22, 130, "pack"),
        ("ST-05", "Whiteboard Markers (4)", "ماركر سبورة (4)", "stationery", 95, 40, 85, "pack"),
        ("ST-06", "Pencil Case", "مقلمة", "stationery", 75, 32, 100, "piece"),
    ],
}

# --------------------------------------------------------------------------
# 3. Mobile shop -- real handsets + accessories
# --------------------------------------------------------------------------
MOBILE_SHOP = {
    "categories": [
        ("samsung", "Samsung", "سامسونج"),
        ("xiaomi", "Xiaomi / Redmi / POCO", "شاومي"),
        ("oppo", "OPPO", "أوبو"),
        ("tecno", "TECNO", "تكنو"),
        ("accessories", "Accessories", "اكسسوارات"),
    ],
    "products": [
        ("SAM-A55", "Samsung Galaxy A55 5G 8/128GB", "سامسونج جالكسي A55", "samsung", 18500, 16800, 12, "piece"),
        ("SAM-A35", "Samsung Galaxy A35 5G 6/128GB", "سامسونج جالكسي A35", "samsung", 14200, 12900, 18, "piece"),
        ("SAM-M14", "Samsung Galaxy M14 4/128GB", "سامسونج جالكسي M14", "samsung", 8900, 7900, 25, "piece"),
        ("SAM-S24", "Samsung Galaxy S24 8/256GB", "سامسونج جالكسي S24", "samsung", 46500, 42000, 5, "piece"),
        ("SAM-S23", "Samsung Galaxy S23 8/256GB", "سامسونج جالكسي S23", "samsung", 38000, 34000, 7, "piece"),
        ("SAM-ZF5", "Samsung Galaxy Z Flip5 8/256GB", "سامسونج جالكسي Z فليب", "samsung", 43000, 39000, 4, "piece"),
        ("XIA-RN13", "Xiaomi Redmi Note 13 8/256GB", "شاومي ريدمي نوت 13", "xiaomi", 12500, 11200, 22, "piece"),
        ("XIA-R13C", "Xiaomi Redmi 13C 6/128GB", "شاومي ريدمي 13C", "xiaomi", 6900, 6100, 30, "piece"),
        ("XIA-PX5", "POCO X5 Pro 8/256GB", "بوكو X5 برو", "xiaomi", 16800, 15200, 14, "piece"),
        ("XIA-RN12", "Xiaomi Redmi Note 12 8/256GB", "شاومي ريدمي نوت 12", "xiaomi", 11000, 9800, 20, "piece"),
        ("OPP-A78", "OPPO A78 8/256GB", "أوبو A78", "oppo", 13200, 11800, 19, "piece"),
        ("OPP-A59", "OPPO A59 6/128GB", "أوبو A59", "oppo", 9500, 8500, 24, "piece"),
        ("OPP-R10", "OPPO Reno10 8/256GB", "أوبو رينو 10", "oppo", 21500, 19500, 9, "piece"),
        ("TEC-SP20", "TECNO Spark 20 8/256GB", "تكنو سبارك 20", "tecno", 8900, 7900, 26, "piece"),
        ("TEC-CM30", "TECNO Camon 30 8/256GB", "تكنو كامون 30", "tecno", 14500, 13100, 15, "piece"),
        ("TEC-PVF", "TECNO Phantom V Fold 12/256GB", "تكنو فانتوم V", "tecno", 39000, 35500, 3, "piece"),
        ("ACC-CHG", "33W Fast Charger", "شاحن سريع 33 وات", "accessories", 850, 420, 60, "piece"),
        ("ACC-EAR", "Wired Earphones 3.5mm", "سماعة سلكية", "accessories", 350, 160, 90, "piece"),
        ("ACC-CAS", "Silicone Phone Case", "كفر سيليكون", "accessories", 450, 210, 75, "piece"),
        ("ACC-SCR", "Tempered Glass Screen Protector", "اسكرينة حماية", "accessories", 400, 180, 80, "piece"),
        ("ACC-PWB", "Power Bank 20000mAh", "باور بانك 20000 مللي", "accessories", 1650, 1050, 35, "piece"),
        ("ACC-CBL", "USB-C Fast Cable 1m", "كابل USB-C سريع", "accessories", 300, 130, 100, "piece"),
        ("ACC-CAR", "Car Charger 12V", "شاحن سيارة", "accessories", 500, 230, 55, "piece"),
        ("ACC-MEM", "Memory Card 128GB", "كارت ذاكرة 128 جيجا", "accessories", 950, 520, 45, "piece"),
    ],
}

# --------------------------------------------------------------------------
# 4. Computer shop -- laptops, components, peripherals
# --------------------------------------------------------------------------
COMPUTER_SHOP = {
    "categories": [
        ("laptops", "Laptops", "لاب توب"),
        ("desktops", "Desktops", "كمبيوتر مكتبي"),
        ("components", "Components", "قطع كمبيوتر"),
        ("peripherals", "Peripherals", "ملحقات"),
        ("accessories", "Accessories", "اكسسوارات"),
    ],
    "products": [
        ("LAP-HP15", "HP Pavilion 15 i5 / 8GB / 512GB", "HP بافليون 15", "laptops", 27500, 24800, 8, "piece"),
        ("LAP-DL15", "Dell Inspiron 15 i5 / 8GB / 512GB", "ديل إنسبايرون 15", "laptops", 26200, 23600, 9, "piece"),
        ("LAP-LN3", "Lenovo IdeaPad 3 Ryzen 5 / 8GB / 512GB", "لينوفو ايديا باد 3", "laptops", 21800, 19600, 11, "piece"),
        ("LAP-AS5", "ASUS Vivobook 15 i5 / 8GB / 512GB", "أسوس فيفوبوك 15", "laptops", 24900, 22400, 7, "piece"),
        ("LAP-AC5", "Acer Aspire 5 Ryzen 7 / 16GB / 512GB", "أسر اسباير 5", "laptops", 28900, 26000, 6, "piece"),
        ("LAP-MBA", "MacBook Air M2 8GB / 256GB", "ماك بوك إير M2", "laptops", 62000, 57000, 2, "piece"),
        ("PC-HPD", "HP Desktop i5 / 8GB / 1TB", "HP مكتبي", "desktops", 19500, 17300, 8, "piece"),
        ("PC-DOP", "Dell OptiPlex i5 / 16GB / 512GB", "ديل أوبتي بلكس", "desktops", 22800, 20400, 6, "piece"),
        ("PC-GAM", "Custom Gaming PC RTX 4060 / i5 / 16GB", "كمبيوتر جيمنق", "desktops", 47500, 42500, 3, "piece"),
        ("CMP-SSD1", "Samsung SSD 1TB NVMe", "هارد سامسونج SSD 1 تيرا", "components", 4200, 3200, 24, "piece"),
        ("CMP-HDD2", "WD HDD 2TB", "هارد HDD 2 تيرا", "components", 3600, 2750, 18, "piece"),
        ("CMP-RAM16", "Corsair RAM 16GB DDR4 3200MHz", "رام كورسير 16 جيجا", "components", 4300, 3300, 20, "piece"),
        ("CMP-RTX40", "NVIDIA GeForce RTX 4060 8GB", "كارت شاشة انفيديا RTX 4060", "components", 18500, 16200, 7, "piece"),
        ("CMP-I5", "Intel Core i5-12400", "معالج انتل i5-12400", "components", 6200, 4700, 15, "piece"),
        ("CMP-R5", "AMD Ryzen 5 5600", "معالج ريزن 5 5600", "components", 5800, 4400, 14, "piece"),
        ("CMP-PSU", "PSU 650W 80+ Gold", "باور سبلاي 650 وات", "components", 3200, 2400, 16, "piece"),
        ("CMP-MB", "Motherboard B550M", "لوحة أم B550M", "components", 4900, 3700, 11, "piece"),
        ("PER-KB", "Logitech Keyboard MK270", "كيبورد لوجيتك", "peripherals", 950, 620, 35, "piece"),
        ("PER-MS", "Logitech Mouse G102", "ماوس لوجيتك G102", "peripherals", 750, 490, 40, "piece"),
        ("PER-PR", "HP Laser Printer M15w", "طابعة ليزر HP", "peripherals", 8500, 6800, 9, "piece"),
        ("PER-MN", "Samsung Monitor 24\" IPS", "شاشة سامسونج 24 بوصة", "peripherals", 6200, 4900, 12, "piece"),
        ("PER-KBM", "Mechanical Keyboard RGB", "كيبورد ميكانيكي RGB", "peripherals", 2800, 2000, 14, "piece"),
        ("PER-CAM", "HD Webcam 1080p", "كاميرا 1080p", "peripherals", 1900, 1350, 17, "piece"),
        ("ACC-HUB", "USB-C Hub 7-in-1", "موزع USB-C سبعة منافذ", "accessories", 1400, 950, 26, "piece"),
        ("ACC-HDMI", "HDMI Cable 2m", "كابل HDMI 2 متر", "accessories", 350, 160, 60, "piece"),
        ("ACC-UPS", "UPS 650VA", "مزود طاقة UPS", "accessories", 3800, 2800, 12, "piece"),
        ("ACC-BAG", "Laptop Backpack 15\"", "شنطة لاب توب 15 بوصة", "accessories", 1100, 700, 28, "piece"),
        ("ACC-COOL", "Laptop Cooling Pad", "قاعدة تبريد لاب توب", "accessories", 900, 560, 22, "piece"),
    ],
}

# --------------------------------------------------------------------------
# 5. Grocery -- everyday Egyptian supermarket
# --------------------------------------------------------------------------
GROCERY = {
    "categories": [
        ("dairy", "Dairy & Cheese", "ألبان وأجبان"),
        ("bakery", "Bakery", "مخبوزات"),
        ("pantry", "Pantry Staples", "بقالة"),
        ("beverages", "Beverages", "مشروبات"),
        ("snacks", "Snacks & Sweets", "وجبات خفيفة وحلويات"),
        ("household", "Household & Cleaning", "منظفات"),
        ("produce", "Fresh Produce", "خضار وفاكهة"),
    ],
    "products": [
        ("DRY-DOM", "Domti White Cheese 500g", "جبنة دومتي بيضاء 500 جم", "dairy", 145, 118, 60, "piece"),
        ("DRY-DYM", "Domti Yellow Cheese 200g", "جبنة دومتي صفراء 200 جم", "dairy", 110, 88, 55, "piece"),
        ("DRY-DAN", "Danane Milk Full Cream 1L", "لبن داني 1 لتر", "dairy", 95, 76, 90, "piece"),
        ("DRY-SAR", "Saree Milk 1L", "لبن سري 1 لتر", "dairy", 88, 70, 85, "piece"),
        ("DRY-LAC", "Lactel Yogurt 105g", "زبادي لاكتل", "dairy", 42, 30, 100, "piece"),
        ("DRY-CRM", "Cooking Cream 200ml", "قشدة طبخ 200 مل", "dairy", 75, 55, 70, "piece"),
        ("BAK-SRD", "Sourdough Loaf", "عيش سن", "bakery", 70, 45, 40, "piece"),
        ("BAK-PIT", "Pita Bread (5 Pack)", "عيش فيتي (5)", "bakery", 25, 14, 150, "pack"),
        ("BAK-BAG", "French Baguette", "باتو فرنسي", "bakery", 40, 22, 60, "piece"),
        ("BAK-CRO", "Butter Croissant", "كرواسون بالزبدة", "bakery", 65, 38, 45, "piece"),
        ("BAK-TST", "Toast Bread", "عيش توست", "bakery", 55, 33, 50, "piece"),
        ("PAN-RIC", "Egyptian Rice 1kg", "رز مصري 1 كيلو", "pantry", 65, 48, 120, "kg"),
        ("PAN-MAC", "Macaroni 400g", "مكرونة 400 جم", "pantry", 35, 22, 140, "piece"),
        ("PAN-SUG", "Sugar 1kg", "سكر 1 كيلو", "pantry", 60, 45, 100, "kg"),
        ("PAN-FLR", "Flour 1kg", "دقيق 1 كيلو", "pantry", 48, 33, 90, "kg"),
        ("PAN-TOM", "Tomato Paste 400g", "صلصة طماطم 400 جم", "pantry", 55, 38, 110, "piece"),
        ("PAN-LEN", "Lentils 500g", "عدس 500 جم", "pantry", 75, 52, 80, "piece"),
        ("PAN-OIL", "Corn Oil 800ml", "زيت ذرة 800 مل", "pantry", 130, 96, 70, "piece"),
        ("PAN-VIN", "White Vinegar 500ml", "خل أبيض", "pantry", 40, 26, 85, "piece"),
        ("BEV-NES", "Nescafe Classic 200g", "نسكافيه كلاسيك", "beverages", 385, 320, 45, "piece"),
        ("BEV-JAC", "Jacobs Coffee 250g", "جاكوبس قهوة", "beverages", 340, 275, 40, "piece"),
        ("BEV-LIP", "Lipton Yellow Tea 100 bags", "شاي ليبتون", "beverages", 260, 200, 55, "piece"),
        ("BEV-PEP", "Pepsi 1L", "بيبسي 1 لتر", "beverages", 45, 33, 120, "piece"),
        ("BEV-WAT", "Water 1.5L", "مياه 1.5 لتر", "beverages", 20, 12, 200, "piece"),
        ("BEV-MIR", "Miracle Orange Juice 1L", "عصير أورانج معجزة", "beverages", 85, 60, 60, "piece"),
        ("BEV-MIR2", "Miracle Guava Juice 1L", "عصير جوافة معجزة", "beverages", 85, 60, 55, "piece"),
        ("SNK-CHP", "Lay's Chips 40g", "شيبسي بطاطس", "snacks", 30, 19, 160, "piece"),
        ("SNK-ORE", "Oreo 133g", "أوريو", "snacks", 55, 38, 100, "piece"),
        ("SNK-CHD", "Chedra Chips 60g", "شيبسي شedra", "snacks", 45, 29, 95, "piece"),
        ("SNK-GAL", "Galaxy Chocolate 40g", "شوكولاتة جالكسي", "snacks", 60, 40, 90, "piece"),
        ("SNK-BIS", "Biscuits Assorted 200g", "بسكويت مشكلة", "snacks", 40, 24, 110, "piece"),
        ("SNK-NUT", "Mixed Nuts 200g", "مكسرات مشكلة", "snacks", 210, 150, 35, "piece"),
        ("HOU-LUX", "Lux Soap 125g", "صابون لوكس", "household", 32, 20, 150, "piece"),
        ("HOU-FAR", "Fairy Dish Liquid 500ml", "سائل جلي فيري", "household", 95, 68, 80, "piece"),
        ("HOU-ARI", "Ariel Detergent 2kg", "مسحوق أرييل 2 كيلو", "household", 385, 310, 40, "piece"),
        ("HOU-VIM", "Vim Cleaning Gel 500ml", "جيل تنظيف فيم", "household", 78, 55, 70, "piece"),
        ("HOU-TIS", "Fine Toilet Paper (4 Rolls)", "مناديل فاين", "household", 75, 45, 90, "pack"),
        ("HOU-SUN", "Sunsilk Shampoo 400ml", "شامبو صن سيلك", "household", 210, 145, 45, "piece"),
        ("PRD-TOM", "Tomatoes 1kg", "طماطم 1 كيلو", "produce", 35, 20, 80, "kg"),
        ("PRD-POT", "Potatoes 1kg", "بطاطس 1 كيلو", "produce", 30, 17, 100, "kg"),
        ("PRD-ONI", "Onions 1kg", "بصل 1 كيلو", "produce", 32, 18, 90, "kg"),
        ("PRD-BAN", "Bananas 1kg", "موز 1 كيلو", "produce", 65, 42, 60, "kg"),
        ("PRD-APP", "Red Apples 1kg", "تفاح أحمر 1 كيلو", "produce", 110, 75, 45, "kg"),
        ("PRD-LEM", "Lemons 1kg", "ليمون 1 كيلو", "produce", 55, 32, 50, "kg"),
        ("PRD-EGG", "Eggs (12 Pack)", "بيض 12 بيضة", "produce", 120, 92, 40, "pack"),
    ],
}

CATALOGUES = [
    ("demo-restaurant", "RST", RESTAURANT),
    ("demo-book-store", "BKS", BOOK_STORE),
    ("demo-mobile-shop", "MOB", MOBILE_SHOP),
    ("demo-computer-shop", "CMP", COMPUTER_SHOP),
    ("demo-grocery", "GRC", GROCERY),
]


def q(value: str) -> str:
    return "'" + str(value).replace("'", "''") + "'"


def minor(egp: float) -> int:
    return int(round(float(egp) * 100))


def unit_code(unit: str) -> str:
    """Map a catalogue unit onto a code that exists in the `units` table."""
    try:
        return UNIT_MAP[unit]
    except KeyError as exc:
        raise SystemExit(
            f"unknown unit {unit!r}; add it to UNIT_MAP (valid: {sorted(VALID_UNITS)})"
        ) from exc


def build_tenant_sql(slug: str, prefix: str, data: dict) -> str:
    barcode_seed = BARCODE_PREFIX + CATALOGUES.index((slug, prefix, data)) * 1000
    category_rows = ",\n        ".join(
        "(tid, {name}, {name_ar}, {slug})".format(name=q(name), name_ar=q(name_ar), slug=q(cslug))
        for cslug, name, name_ar in data["categories"]
    )
    product_rows = []
    for index, (sku_suffix, name, name_ar, category, price, cost, stock, unit) in enumerate(
        data["products"]
    ):
        sku = f"{prefix}-{sku_suffix}"
        barcode = ean13(barcode_seed + index + 1)
        product_rows.append(
            "(tid,\n         (SELECT id FROM categories WHERE tenant_id = tid AND slug = {cat}),\n"
            "         {name}, {name_ar},\n"
            "         {desc}, {desc_ar},\n"
            "         {sku}, {barcode}, {price}, {cost},\n"
            "         {currency}, {stock}, {unit}, TRUE, TRUE)".format(
                cat=q(category),
                name=q(name),
                name_ar=q(name_ar),
                desc=q(f"{name} - demo catalogue item"),
                desc_ar=q(f"{name_ar} - صنف تجريبي"),
                sku=q(sku),
                barcode=q(barcode),
                price=minor(price),
                cost=minor(cost),
                currency=q(CURRENCY),
                stock=stock,
                unit=q(unit_code(unit)),
            )
        )
    products = ",\n      ".join(product_rows)
    return f"""  -- {slug}: {len(data['categories'])} categories, {len(data['products'])} products
  SELECT id INTO tid FROM tenants WHERE slug = {q(slug)};
  IF tid IS NULL THEN
    RAISE NOTICE 'skip {slug} (tenant missing)';
  ELSE
    PERFORM set_config('app.current_tenant', tid::text, true);

    INSERT INTO categories (tenant_id, name, name_ar, slug) VALUES
        {category_rows}
    ON CONFLICT (tenant_id, slug) DO UPDATE SET
      name = EXCLUDED.name,
      name_ar = COALESCE(EXCLUDED.name_ar, categories.name_ar),
      is_active = TRUE,
      updated_at = now();

    INSERT INTO products (tenant_id, category_id, name, name_ar, description, description_ar,
                          sku, barcode, price_minor, cost_minor, currency, stock_quantity,
                          unit, is_active, selforder_enabled)
    SELECT * FROM (VALUES
      {products}
    ) AS v(tenant_id, category_id, name, name_ar, description, description_ar,
           sku, barcode, price_minor, cost_minor, currency, stock_quantity, unit,
           is_active, selforder_enabled)
    ON CONFLICT (tenant_id, sku) DO UPDATE SET
      category_id    = EXCLUDED.category_id,
      name           = EXCLUDED.name,
      name_ar        = EXCLUDED.name_ar,
      description    = EXCLUDED.description,
      description_ar = EXCLUDED.description_ar,
      barcode        = EXCLUDED.barcode,
      price_minor    = EXCLUDED.price_minor,
      cost_minor     = EXCLUDED.cost_minor,
      currency       = EXCLUDED.currency,
      -- restock to the catalogue level so the cron keeps demo shops sellable
      stock_quantity = EXCLUDED.stock_quantity,
      unit           = EXCLUDED.unit,
      is_active      = TRUE,
      updated_at     = now();
    RAISE NOTICE '{slug}: % categories / % products', {len(data['categories'])}, {len(data['products'])};
  END IF;"""


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--print", action="store_true", help="print a per-tenant summary")
    parser.add_argument("--out", default=str(OUT), help="output SQL path")
    args = parser.parse_args()

    blocks = [build_tenant_sql(*entry) for entry in CATALOGUES]
    total = sum(len(entry[2]["products"]) for entry in CATALOGUES)
    sql = f"""-- Rich demo catalogue: 5 demo tenants, {total} products with real
-- Egyptian-market names, prices, EAN-13 barcodes and stock.
-- Generated by tool/gen_demo_products.py -- safe to re-apply (idempotent upsert),
-- which is how /opt/pos/seed_demo_products.sh keeps the demo shops stocked.
-- Run as the database owner:  psql "$DATABASE_URL" -f {Path(args.out).name}

DO $$
DECLARE
  tid UUID;
BEGIN
{chr(10).join(blocks)}
END $$;
"""
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(sql, encoding="utf-8")
    out.chmod(0o600)

    print(f"wrote {out} ({total} products across {len(CATALOGUES)} tenants)")
    if args.print:
        for slug, prefix, data in CATALOGUES:
            print(f"  {slug:22} {len(data['categories'])} categories  "
                  f"{len(data['products'])} products  prefix {prefix}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())