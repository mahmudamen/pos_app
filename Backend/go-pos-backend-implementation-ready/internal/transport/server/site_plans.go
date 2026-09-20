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
var featureTranslations = map[string]map[string]string{
	"en": {
		"pos.basic":          "Core point of sale",
		"pos.advanced":       "Advanced point of sale",
		"inventory.basic":    "Basic inventory",
		"inventory.advanced": "Advanced inventory",
		"dashboard.basic":    "Basic dashboard",
		"dashboard.advanced": "Advanced dashboard & analytics",
		"restaurant":         "Restaurants (tables & split bills)",
		"loyalty":            "Customer loyalty",
		"pharmacy":           "Pharmacies (lots & expiry)",
		"textile":            "Textile (sizes & colours)",
		"sync.multi_device":  "Multi-device sync",
		"e_invoice":          "E-invoice ready",
		"ocr_capture":        "OCR purchase capture",
		"self_order":         "QR self-ordering",
		"refunds":            "Refunds & returns",
		"discount_policy":    "Discount policy & manager PIN",
		"billing":            "Monthly billing in EGP",
	},
	"ar": {
		"pos.basic":          "نقاط بيع أساسية",
		"pos.advanced":       "نقاط بيع متقدمة",
		"inventory.basic":    "مخزون أساسي",
		"inventory.advanced": "مخزون متقدم",
		"dashboard.basic":    "لوحة تحكم أساسية",
		"dashboard.advanced": "لوحة تحكم وتقارير متقدمة",
		"restaurant":         "مطاعم (طاولات وتقسيم فواتير)",
		"loyalty":            "ولاء العملاء",
		"pharmacy":           "صيدليات (دفعات وانتهاء صلاحية)",
		"textile":            "أقمشة وأزياء (مقاسات وألوان)",
		"sync.multi_device":  "مزامنة متعددة الأجهزة",
		"e_invoice":          "جاهز للفاتورة الإلكترونية",
		"ocr_capture":        "التقاط الفواتير بـ OCR",
		"self_order":         "طلب ذاتي عبر QR",
		"refunds":            "مرتجعات ومرتجعات",
		"discount_policy":    "سياسة الخصم ورقم PIN للمدير",
		"billing":            "فوترة شهرية بالجنيه المصري",
	},
}
