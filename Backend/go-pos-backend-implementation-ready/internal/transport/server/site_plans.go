package server

// planTranslations localizes the plan catalog names + descriptions by plan code.
var planTranslations = map[string]map[string]struct{ Name, Desc string }{
	"en": {
		"trial":      {"Trial", "Trial entitlement granted at signup"},
		"starter":    {"Starter", "Solo store getting started"},
		"business":   {"Business", "Growing multi-cashier store"},
		"enterprise": {"Enterprise", "Unlimited stores and verticals"},
	},
	"ar": {
		"trial":      {"تجريبية", "اشتراك تجريبي يُمنح عند التسجيل"},
		"starter":    {"المبتدئ", "متجر منفرد للبداية"},
		"business":   {"الأعمال", "متجر متنامٍ بأكثر من كاشير"},
		"enterprise": {"المؤسسات", "متاجر وقطاعات غير محدودة"},
	},
}

// featureTranslations localizes plan feature keys into display labels.
//
// This table is deliberately kept to features that actually exist. An earlier
// draft also carried pos.advanced, e_invoice, self_order, refunds,
// discount_policy and billing: no seeded plan grants them and no code reads
// them, so they were a list of promises with nothing behind them, which is
// the failure this table is most likely to cause. The reverse mistake was
// just as real — pos.subdomain is granted to every plan by migration 041 and
// had no entry, so the live /pricing page rendered the raw key to customers in
// five places. A key that is missing here degrades to the raw string, so the
// table must cover every key the plans actually carry.
var featureTranslations = map[string]map[string]string{
	"en": {
		"pos.basic":          "Core point of sale",
		"pos.subdomain":      "Your own store subdomain",
		"inventory.basic":    "Basic inventory",
		"inventory.advanced": "Advanced inventory",
		"dashboard.basic":    "Basic dashboard",
		"dashboard.advanced": "Advanced dashboard & analytics",
		"restaurant":         "Restaurants (tables & split bills)",
		"loyalty":            "Customer loyalty",
		"pharmacy":           "Pharmacies (lots & expiry)",
		"textile":            "Textile (sizes & colours)",
		"sync.multi_device":  "Multi-device sync",
		"ocr_capture":        "Invoice capture (OCR)",
	},
	"ar": {
		"pos.basic":          "نقاط بيع أساسية",
		"pos.subdomain":      "نطاق فرعي خاص بمتجرك",
		"inventory.basic":    "مخزون أساسي",
		"inventory.advanced": "مخزون متقدم",
		"dashboard.basic":    "لوحة تحكم أساسية",
		"dashboard.advanced": "لوحة تحكم وتقارير متقدمة",
		"restaurant":         "مطاعم (طاولات وتقسيم فواتير)",
		"loyalty":            "ولاء العملاء",
		"pharmacy":           "صيدليات (دفعات وانتهاء صلاحية)",
		"textile":            "أقمشة وأزياء (مقاسات وألوان)",
		"sync.multi_device":  "مزامنة متعددة الأجهزة",
		"ocr_capture":        "التقاط الفواتير (OCR)",
	},
}
