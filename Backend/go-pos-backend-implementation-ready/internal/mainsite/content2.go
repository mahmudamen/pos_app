package mainsite

// This file holds the product, marketing and legal pages plus the page registry.

// posPage returns /odoo-pos-egypt.
func posPage() page {
	return page{
		Slug:    "/odoo-pos-egypt",
		Nav:     "pos",
		Title:   more("Odoo POS Egypt — offline, multi-store, integrated point of sale", "نقاط بيع Odoo في مصر — تعمل دون إنترنت ومتعددة الفروع ومتكاملة"),
		Meta:    more("Odoo POS in Egypt: works offline, multi-store, receipt printing and full ERP integration. XamlTech implements and customizes Odoo Point of Sale.", "نقاط بيع Odoo في مصر: تعمل دون إنترنت، متعددة الفروع، طباعة إيصالات وتكامل كامل مع ERP. زامل تك تنفّذ وتخصّص نقطة بيع Odoo."),
		Eyebrow: more("Odoo POS", "نقاط بيع أودو"),
		H1:      more("The point of sale that never stops selling.", "نقطة بيع لا تتوقف عن البيع."),
		Lead:    more("Odoo POS runs on any device, keeps working when the internet drops, and stays fully integrated with your ERP — inventory, accounting and customers.", "تعمل نقطة بيع Odoo على أي جهاز، وتستمر دون إنترنت، وتبقى متكاملة مع نظام ERP — المخزون والمحاسبة والعملاء."),
		HeroCTAs: []link{
			{Label: more("Try Live Demo", "جرّب العرض الحي"), Href: demoURL, Kind: "primary", Ext: true},
			{Label: more("Get a Quote", "اطلب عرض سعر"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secFeatures,
				Features: []feature{
					{Icon: "📴", Title: more("Works Offline", "يعمل دون إنترنت"), Body: more("Keep selling when the connection drops — transactions sync automatically when you're back online.", "استمر في البيع عند انقطاع الاتصال — وتتزامن العمليات تلقائيًا عند العودة.")},
					{Icon: "🏬", Title: more("Multi-Store", "متعدد الفروع"), Body: more("Central catalog and pricing across every branch, with per-register sessions.", "كتالوج مركزي وأسعار موحّدة لكل الفروع مع جلسات لكل نقطة بيع.")},
					{Icon: "🧾", Title: more("Receipt Printing", "طباعة الإيصالات"), Body: more("Thermal ESC/POS receipts with QR codes, Arabic support and split payments.", "إيصالات ESC/POS حرارية مع رموز QR ودعم العربية والدفع المقسّم.")},
					{Icon: "📱", Title: more("Any Device", "أي جهاز"), Body: more("Tablets, laptops and desktops — one interface that adapts.", "أجهزة لوحية ولابتوبات ومكتبية — واجهة واحدة تتكيف.")},
					{Icon: "🔗", Title: more("Fully Integrated", "متكامل بالكامل"), Body: more("Sales, stock, accounting, loyalty and customers update in real time.", "المبيعات والمخزون والمحاسبة والولاء والعملاء تُحدّث لحظيًا.")},
					{Icon: "🔐", Title: more("Security & Control", "الأمن والتحكم"), Body: more("Manager PIN, discount limits, register sessions and role-based access.", "رقم PIN للمدير وحدود الخصم وجلسات الصندوق وصلاحيات حسب الدور.")},
				},
			},
			{
				Kind:  secCards,
				Label: more("POS Addons", "إضافات نقاط البيع"),
				Title: more("Extend Odoo POS with our modules", "طوّر نقطة بيع أودو بإضافاتنا"),
				Cards: []card{
					{Icon: "✂️", Badge: more("Best seller", "الأكثر مبيعًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Cutting Orders", "أوامر القص"), Body: more("Fabric cutting managed from the POS.", "إدارة قص الأقمشة من نقطة البيع."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.9 / 5", "4.9 / 5")}, Price: more("€149", "€149"), Href: "/addons-store"},
					{Icon: "📦", Badge: more("Top rated", "الأعلى تقييمًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Stock Real-Time", "المخزون الفوري"), Body: more("Live stock sync across sessions.", "مزامنة المخزون الفورية عبر الجلسات."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.8 / 5", "4.8 / 5")}, Price: more("€129", "€129"), Href: "/addons-store"},
					{Icon: "🔐", Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Security Framework", "إطار أمان نقاط البيع"), Body: more("Manager PIN and discount limits.", "رقم PIN للمدير وحدود الخصم."), Meta: []bi{more("v19", "v19"), more("4.7 / 5", "4.7 / 5")}, Price: more("€99", "€99"), Href: "/addons-store"},
				},
			},
		},
		Final: finalBand(more("Run Odoo POS on any device.", "شغّل نقاط بيع أودو على أي جهاز."),
			more("Book a demonstration with our team.", "احجز عرضًا توضيحيًا مع فريقنا."),
			link{Label: more("Request a Demo", "اطلب تجربة"), Href: "/request-demo", Kind: "primary"},
			link{Label: more("Browse Addons", "تصفّح الإضافات"), Href: "/addons-store", Kind: "ghost"}),
	}
}

// agriPage returns /erp-agriculture-egypt.
func agriPage() page {
	return page{
		Slug:    "/erp-agriculture-egypt",
		Nav:     "agri",
		Title:   more("Agriculture ERP for Egypt — farm management, crops & cost analytics", "ERP للزراعة في مصر — إدارة المزارع والمحاصيل وتحليل التكاليف"),
		Meta:    more("Agriculture ERP for Egyptian farms: crop lifecycle, farm hierarchy, work orders, harvest tracking and full accounting — built on Odoo.", "ERP زراعي للمزارع المصرية: دورة المحصول، هيكل المزرعة، أوامر العمل، تتبع الحصاد والمحاسبة الكاملة — مبني على أودو."),
		Eyebrow: more("Agriculture ERP", "ERP الزراعة"),
		H1:      more("Farm management built for Upper Egypt.", "إدارة مزارع مصممة لصعيد مصر."),
		Lead:    more("Track every field, crop, work order and harvest cost — and connect it to full accounting, inventory and payroll.", "تابع كل حقل ومحصول وأمر عمل وتكلفة حصاد — واربطها بالمحاسبة والمخزون والرواتب."),
		HeroCTAs: []link{
			{Label: more("Download Free Addon", "حمّل الإضافة المجانية"), Href: odooAppsURL, Kind: "primary", Ext: true},
			{Label: more("Talk to Us", "تحدّث معنا"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secFeatures,
				Features: []feature{
					{Icon: "🌱", Title: more("Crop Lifecycle", "دورة حياة المحصول"), Body: more("Plan seasons, track planting, growth stages, treatments and yield per crop.", "خطط للمواسم وتابع الزراعة ومراحل النمو والمعالجات والمحصول لكل محصول.")},
					{Icon: "🗺️", Title: more("Farm Hierarchy", "هيكل المزرعة"), Body: more("Organize farms, plots, fields and sub-fields with full geographic detail.", "نظّم المزارع والقطع والحقول والحقول الفرعية بتفصيل جغرافي كامل.")},
					{Icon: "🧑‍🌾", Title: more("Work Orders", "أوامر العمل"), Body: more("Assign labor, equipment and materials to field operations and track progress.", "وزّع العمالة والمعدات والمواد على العمليات الميدانية وتابع التقدم.")},
					{Icon: "💰", Title: more("Cost Analytics", "تحليل التكاليف"), Body: more("Cost per field, crop and operation — know your true profitability.", "التكلفة لكل حقل ومحصول وعملية — اعرف ربحيتك الحقيقية.")},
					{Icon: "📦", Title: more("Harvest Tracking", "تتبع الحصاد"), Body: more("Record harvest quantities, quality and storage into inventory.", "سجّل كميات الحصاد والجودة والتخزين في المخزون.")},
					{Icon: "🧮", Title: more("Full Accounting", "محاسبة كاملة"), Body: more("Native Odoo accounting — invoices, expenses, assets and reporting.", "محاسبة أودو الأصلية — فواتير ومصروفات وأصول وتقارير.")},
				},
			},
			{
				Kind:  secCards,
				Label: more("Crops", "المحاصيل"),
				Title: more("Supported across Egyptian agriculture", "مدعومة عبر الزراعة المصرية"),
				Cards: []card{
					{Icon: "🌾", Title: more("Field Crops", "المحاصيل الحقلية")},
					{Icon: "🌴", Title: more("Palm & Dates", "النخيل والتمور")},
					{Icon: "🍊", Title: more("Citrus & Fruit", "الحمضيات والفاكهة")},
					{Icon: "🥬", Title: more("Vegetables", "الخضروات")},
					{Icon: "🧵", Title: more("Cotton", "القطن")},
					{Icon: "🐄", Title: more("Livestock", "الثروة الحيوانية")},
				},
			},
		},
		Final: finalBand(more("Modernize your farm management.", "رقّن إدارة مزرعتك."),
			more("Start with the free addon or book a consultation.", "ابدأ بالإضافة المجانية أو احجز استشارة."),
			link{Label: more("Download Addon", "حمّل الإضافة"), Href: odooAppsURL, Kind: "primary", Ext: true},
			waLink()),
	}
}

// aiPage returns /ai-solutions.
func aiPage() page {
	return page{
		Slug:    "/ai-solutions",
		Nav:     "ai",
		Title:   more("AI Copilots for Odoo 19 — marketing and POS, self-hosted", "مساعدون أذكياء لأودو 19 — للتسويق ونقاط البيع، مستضافون ذاتيًا"),
		Meta:    more("AI Marketing Copilot and AI POS Copilot for Odoo 19 — multi-provider, self-hosted, privacy-first. Get more sales without a data team.", "مساعد التسويق الذكي ومساعد نقاط البيع الذكي لأودو 19 — متعدد المزودين ومستضاف ذاتيًا ويحفظ الخصوصية. زد مبيعاتك دون فريق بيانات."),
		Eyebrow: more("AI Solutions", "حلول الذكاء الاصطناعي"),
		H1:      more("AI copilots that work inside your Odoo.", "مساعدون أذكياء يعملون داخل أودو."),
		Lead:    more("Two copilots that turn your Odoo data into action: AI Marketing Copilot to grow demand, and AI POS Copilot to close the sale — both self-hosted, multi-provider and privacy-first.", "مساعدان يحوّلان بيانات أودو إلى أفعال: مساعد التسويق لزيادة الطلب، ومساعد نقاط البيع لإتمام البيع — كلاهما مستضاف ذاتيًا ومتعدد المزودين ويحفظ الخصوصية."),
		HeroCTAs: []link{
			{Label: more("Browse AI Addons", "تصفّح إضافات الذكاء"), Href: "/addons-store", Kind: "primary"},
			{Label: more("Talk to Us", "تحدّث معنا"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secCards,
				Cards: []card{
					{Icon: "📣", Eyebrow: more("Marketing · Odoo 19", "التسويق · أودو 19"), Title: more("AI Marketing Copilot", "مساعد التسويق الذكي"), Body: more("Product descriptions, SEO keywords, campaign copy and CRM insights — generated from your own catalog.", "أوصاف المنتجات وكلمات SEO ونصوص الحملات ورؤى CRM — مولّدة من كتالوجك."), Meta: []bi{more("Multi-provider", "متعدد المزودين"), more("Self-hosted", "مستضاف ذاتيًا"), more("From €79", "من €79")}, Href: "/insights/ai-marketing-copilot", CTA: more("Read the guide", "اقرأ الدليل")},
					{Icon: "🛒", Eyebrow: more("Point of Sale · Odoo 19", "نقاط البيع · أودو 19"), Title: more("AI POS Copilot", "مساعد نقاط البيع الذكي"), Body: more("Real-time upsell and cross-sell suggestions at the till, using the basket and stock on hand.", "اقتراحات بيع وبيع متقاطع فورية عند الصندوق، بناءً على السلة والمخزون."), Meta: []bi{more("Offline-aware", "يدعم دون إنترنت"), more("Cache-first", "ذاكرة أولًا"), more("From €59", "من €59")}, Href: "/insights/ai-pos-copilot", CTA: more("Read the guide", "اقرأ الدليل")},
				},
			},
			{
				Kind:  secFeatures,
				Label: more("Why our AI", "لماذا ذكاؤنا الاصطناعي؟"),
				Title: more("Built for Egyptian businesses.", "مصمّم للأعمال المصرية."),
				Features: []feature{
					{Icon: "🔐", Title: more("Privacy-first", "الخصوصية أولًا"), Body: more("Self-hosted — your data never leaves your server.", "مستضاف ذاتيًا — بياناتك لا تغادر سيرفرك.")},
					{Icon: "🔀", Title: more("Multi-provider", "متعدد المزودين"), Body: more("Switch between AI providers without changing your workflow.", "بدّل بين مزودي الذكاء الاصطناعي دون تغيير سير عملك.")},
					{Icon: "🌐", Title: more("Arabic-aware", "يدعم العربية"), Body: more("Copy generated in Arabic and English, ready for the Egyptian market.", "محتوى مولّد بالعربية والإنجليزية جاهز للسوق المصري.")},
					{Icon: "📈", Title: more("ROI focused", "يركّز على العائد"), Body: more("Measured against real sales, not vanity metrics.", "يُقاس بمبيعات حقيقية لا بمؤشرات شكلية.")},
				},
			},
		},
		Final: closing(),
	}
}

// addonsPage returns /addons-store.
func addonsPage() page {
	return page{
		Slug:    "/addons-store",
		Nav:     "addons",
		Title:   more("XamlTech Addons Store — premium Odoo 19 modules", "متجر إضافات زامل تك — موديولات Odoo 19 متميزة"),
		Meta:    more("Production-tested Odoo 19 addons: POS security, cutting orders, real-time stock, revenue management, agriculture and AI copilots. Published on the official Odoo Apps Store.", "إضافات Odoo 19 مُختبرة: أمان نقاط البيع، أوامر القص، المخزون الفوري، إدارة الإيرادات، الزراعة والمساعدون الأذكياء. منشورة على متجر Odoo الرسمي."),
		Eyebrow: more("Addons Store", "متجر الإضافات"),
		H1:      more("Premium Odoo addons, production-tested.", "إضافات Odoo متميزة، مُختبرة في الإنتاج."),
		Lead:    more("Eight modules published and battle-tested, with automated test suites and Arabic support.", "ثمانية موديولات منشورة ومجرّبة، باختبارات آلية ودعم للعربية."),
		HeroCTAs: []link{
			{Label: more("Open Odoo Apps Store", "افتح متجر Odoo"), Href: odooAppsURL, Kind: "primary", Ext: true},
			{Label: more("Request a Custom Addon", "اطلب إضافة مخصصة"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind:  secCards,
				Title: more("Featured addons", "إضافات مميزة"),
				Lead:  more("Point of Sale · v19 / v18", "نقاط البيع · v19 / v18"),
				Cards: []card{
					{Icon: "🔐", Badge: more("Essential", "أساسي"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Security Framework", "إطار أمان نقاط البيع"), Body: more("Manager PIN, discount limits and backend-enforced session security.", "رقم PIN للمدير وحدود الخصم وأمان الجلسات على مستوى الخادم."), Meta: []bi{more("v19 · v18 · v17", "v19 · v18 · v17"), more("4.7 / 5", "4.7 / 5")}, Price: more("€99", "€99"), Href: odooAppsURL},
					{Icon: "🎨", Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Customization Kit", "طقم تخصيص نقاط البيع"), Body: more("A complete theme builder: dark/light modes, RTL and layout presets.", "منشئ قوالب كامل: أوضاع داكن/فاتح، RTL وقوالب تخطيط."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.8 / 5", "4.8 / 5")}, Price: more("€89", "€89"), Href: odooAppsURL},
					{Icon: "✂️", Badge: more("Best seller", "الأكثر مبيعًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Cutting Orders", "أوامر القص"), Body: more("Fabric cutting orders handled right at the point of sale.", "أوامر قص الأقمشة مباشرة عند نقطة البيع."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.9 / 5", "4.9 / 5")}, Price: more("€149", "€149"), Href: odooAppsURL},
					{Icon: "📦", Badge: more("Top rated", "الأعلى تقييمًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Stock Real-Time", "المخزون الفوري"), Body: more("Reactive stock badges and batch loading across every session.", "شارات مخزون تفاعلية وتحميل دفعات عبر الجلسات."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.8 / 5", "4.8 / 5")}, Price: more("€129", "€129"), Href: odooAppsURL},
					{Icon: "🏛️", Eyebrow: more("Revenue Management", "إدارة الإيرادات"), Title: more("HDA Collection", "مجموعة HDA للتحصيل"), Body: more("Revenue collection for museums and tourism, with ticketing and reporting.", "تحصيل الإيرادات للمتاحف والسياحة، مع التذاكر والتقارير."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.6 / 5", "4.6 / 5")}, Price: more("€249", "€249"), Href: odooAppsURL},
					{Icon: "🌾", Badge: more("Free", "مجاني"), Eyebrow: more("Agriculture", "الزراعة"), Title: more("Advanced Agricultural Management", "الإدارة الزراعية المتقدمة"), Body: more("Farm-to-warehouse tracking: crops, fields, work orders and harvest.", "تتبع من المزرعة إلى المخزن: محاصيل وحقول وأوامر عمل وحصاد."), Meta: []bi{more("v19 · v18", "v19 · v18"), more("4.9 / 5", "4.9 / 5")}, Price: more("Free", "مجاني"), Href: odooAppsURL},
					{Icon: "📣", Eyebrow: more("AI · Odoo 19", "ذكاء اصطناعي · أودو 19"), Title: more("AI Marketing Copilot", "مساعد التسويق الذكي"), Body: more("Product copy, SEO keywords and CRM insights, multi-provider.", "نصوص المنتجات وكلمات SEO ورؤى CRM، متعدد المزودين."), Meta: []bi{more("v19", "v19"), more("New", "جديد")}, Price: more("€79", "€79"), Href: odooAppsURL},
					{Icon: "🛒", Eyebrow: more("AI · Odoo 19", "ذكاء اصطناعي · أودو 19"), Title: more("AI POS Copilot", "مساعد نقاط البيع الذكي"), Body: more("Real-time upsell suggestions at the till, cache-first.", "اقتراحات بيع فورية عند الصندوق، ذاكرة أولًا."), Meta: []bi{more("v19", "v19"), more("New", "جديد")}, Price: more("€59", "€59"), Href: odooAppsURL},
				},
			},
			{
				Kind:  secFeatures,
				Label: more("Why our addons", "لماذا إضافاتنا؟"),
				Title: more("Built like software, not demos.", "مبنية كبرمجيات حقيقية، لا عروض."),
				Features: []feature{
					{Icon: "🧪", Title: more("Automated tests", "اختبارات آلية"), Body: more("Every module ships with a test suite — 300+ tests across the stack.", "كل موديول يأتي بطقم اختبار — أكثر من ٣٠٠ اختبار.")},
					{Icon: "🌍", Title: more("Arabic & RTL", "العربية و RTL"), Body: more("Native Arabic interfaces and reports, not machine translation.", "واجهات وتقارير عربية أصلية، لا ترجمة آلية.")},
					{Icon: "⬆️", Title: more("Upgrade-safe", "آمنة للترقية"), Body: more("Clean inheritance, no core patches, forward-only migrations.", "وراثة نظيفة، بلا تعديل للنواة، وترقيات أحادية الاتجاه.")},
					{Icon: "📚", Title: more("Documentation", "الوثائق"), Body: more("Install guides, configuration docs and changelogs.", "أدلة التركيب ووثائق الإعداد وسجل التغييرات.")},
				},
			},
		},
		Final: finalBand(more("Need a module that doesn't exist yet?", "هل تحتاج موديول غير موجود حتى الآن؟"),
			more("We build custom Odoo addons to spec.", "نبني إضافات Odoo مخصصة حسب طلبك."),
			link{Label: more("Request a Custom Addon", "اطلب إضافة مخصصة"), Href: "/contact-us", Kind: "primary"},
			link{Label: more("View on Odoo Apps", "عرض على Odoo Apps"), Href: odooAppsURL, Kind: "ghost", Ext: true}),
	}
}

// pricingPage returns /pricing.
func pricingPage() page {
	return page{
		Slug:    "/pricing",
		Nav:     "pricing",
		Title:   more("Addon Pricing — one-time purchase, lifetime updates", "أسعار الإضافات — شراء لمرة واحدة وتحديثات مدى الحياة"),
		Meta:    more("XamlTech Odoo addon pricing: POS Security Framework, Customization Kit, Cutting Orders, Stock Real-Time, HDA Collection and AI copilots. One-time purchase with lifetime updates.", "أسعار إضافات Odoo من زامل تك: إطار أمان نقاط البيع، طقم التخصيص، أوامر القص، المخزون الفوري، مجموعة HDA والمساعدون الأذكياء. شراء لمرة واحدة مع تحديثات مدى الحياة."),
		Eyebrow: more("Pricing", "الأسعار"),
		H1:      more("One-time purchase. Lifetime updates.", "شراء لمرة واحدة. تحديثات مدى الحياة."),
		Lead:    more("Transparent per-module pricing in USD and EUR. No subscriptions, no per-seat fees — buy once and keep the module.", "أسعار واضحة لكل موديول بالدولار واليورو. لا اشتراكات ولا رسوم لكل مستخدم — اشترِ مرة وامتلك الموديول."),
		HeroCTAs: []link{
			{Label: more("Open Odoo Apps Store", "افتح متجر Odoo"), Href: odooAppsURL, Kind: "primary", Ext: true},
			{Label: more("Need a Bundle?", "تحتاج باقة؟"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind:  secPlans,
				Title: more("Point of Sale addons", "إضافات نقاط البيع"),
				Plans: []plan{
					{Name: more("POS Security Framework", "إطار أمان نقاط البيع"), Desc: more("Manager PIN, discount limits, register session security.", "رقم PIN للمدير وحدود الخصم وأمان جلسات الصندوق."), Price: more("$99", "$99"), Currency: "USD", Features: []bi{more("v19 · v18 · v17", "v19 · v18 · v17"), more("Automated tests", "اختبارات آلية"), more("Arabic ready", "جاهز للعربية")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
					{Name: more("POS Customization Kit", "طقم تخصيص نقاط البيع"), Desc: more("Theme builder, dark/light, RTL, layout presets.", "منشئ قوالب، داكن/فاتح، RTL، قوالب تخطيط."), Price: more("$89", "$89"), Currency: "USD", Features: []bi{more("v19 · v18", "v19 · v18"), more("Dark & light", "داكن وفاتح"), more("RTL built-in", "دعم RTL مدمج")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
					{Name: more("POS Cutting Orders", "أوامر القص"), Desc: more("Fabric cutting at the point of sale.", "قص الأقمشة عند نقطة البيع."), Price: more("$149", "$149"), Currency: "USD", Featured: true, Badge: more("Best seller", "الأكثر مبيعًا"), Features: []bi{more("v19 · v18", "v19 · v18"), more("Textile vertical", "قطاع المنسوجات"), more("4.9 / 5", "4.9 / 5")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
					{Name: more("POS Stock Real-Time", "المخزون الفوري"), Desc: more("Reactive stock badges and batch loading.", "شارات مخزون تفاعلية وتحميل دفعات."), Price: more("$129", "$129"), Currency: "USD", Badge: more("Top rated", "الأعلى تقييمًا"), Features: []bi{more("v19 · v18", "v19 · v18"), more("Multi-store", "متعدد الفروع"), more("4.8 / 5", "4.8 / 5")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
				},
			},
			{
				Kind:  secPlans,
				Title: more("Revenue, agriculture & AI", "الإيرادات والزراعة والذكاء الاصطناعي"),
				Plans: []plan{
					{Name: more("HDA Collection", "مجموعة HDA للتحصيل"), Desc: more("Revenue management for museums and tourism.", "إدارة الإيرادات للمتاحف والسياحة."), Price: more("$249", "$249"), Currency: "USD", Features: []bi{more("Ticketing", "تذاكر"), more("Reporting", "تقارير"), more("v19 · v18", "v19 · v18")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
					{Name: more("Advanced Agriculture", "الإدارة الزراعية المتقدمة"), Desc: more("Farm-to-warehouse crop and harvest management.", "إدارة المحاصيل والحصاد من المزرعة للمخزن."), Price: more("Free", "مجاني"), Currency: "", Badge: more("Free", "مجاني"), Features: []bi{more("Crop lifecycle", "دورة المحصول"), more("Work orders", "أوامر العمل"), more("Harvest tracking", "تتبع الحصاد")}, CTA: more("Buy Now", "احصل عليها"), Href: odooAppsURL},
					{Name: more("AI Marketing Copilot", "مساعد التسويق الذكي"), Desc: more("Product copy, SEO and CRM insights.", "نصوص المنتجات وSEO ورؤى CRM."), Price: more("$79", "$79"), Currency: "USD", Badge: more("New", "جديد"), Features: []bi{more("Multi-provider", "متعدد المزودين"), more("Self-hosted", "مستضاف ذاتيًا"), more("Odoo 19", "أودو 19")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
					{Name: more("AI POS Copilot", "مساعد نقاط البيع الذكي"), Desc: more("Real-time upsell at the till.", "اقتراحات بيع فورية عند الصندوق."), Price: more("$59", "$59"), Currency: "USD", Badge: more("New", "جديد"), Features: []bi{more("Cache-first", "ذاكرة أولًا"), more("Offline-aware", "يدعم دون إنترنت"), more("Odoo 19", "أودو 19")}, CTA: more("Buy Now", "اشترِ الآن"), Href: odooAppsURL},
				},
			},
			{
				Kind: secFAQ,
				FAQ: []faq{
					{more("Which Odoo versions are supported?", "ما إصدارات أودو المدعومة؟"), more("Our current modules target Odoo 19 and 18, with several still supporting 17. Each product page lists its exact versions.", "تستهدف موديولاتنا الحالية Odoo 19 و18، مع دعم بعضها للإصدار 17. كل صفحة منتج توضح الإصدارات.")},
					{more("Is support included?", "هل الدعم مشمول؟"), more("Every purchase includes 12 months of bug-fix support. Extended and priority support is available on request.", "كل عملية شراء تشمل ١٢ شهرًا من إصلاح الأخطاء. الدعم الموسّع والأولوية متاح عند الطلب.")},
					{more("Do you offer custom development?", "هل تقدّمون تطويرًا مخصصًا؟"), more("Yes — we build custom Odoo addons to specification. Contact us with your requirements.", "نعم — نبني إضافات Odoo مخصصة حسب الطلب. تواصل معنا بمتطلباتك.")},
					{more("How do I pay?", "كيف أدفع؟"), more("Purchases are handled securely through the official Odoo Apps Store, in USD or EUR.", "تتم عمليات الشراء بأمان عبر متجر Odoo الرسمي، بالدولار أو اليورو.")},
				},
			},
		},
		Final: finalBand(more("Not sure which module fits?", "لست متأكدًا أي موديول يناسبك؟"),
			more("Tell us your workflow and we'll recommend the right addons.", "أخبرنا بسير عملك وسنرشّح لك الإضافات المناسبة."),
			waLink(),
			link{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"}),
	}
}

// portfolioPage returns /portfolio.
func portfolioPage() page {
	return page{
		Slug:    "/portfolio",
		Nav:     "portfolio",
		Title:   more("Our Work — ERP, POS, healthcare & textile projects", "أعمالنا — مشاريع ERP ونقاط بيع وصحة ومنسوجات"),
		Meta:    more("Selected XamlTech projects: surgical centers, hospitals, textile POS, ZATCA e-invoicing and farm management — real deployments in Egypt.", "مشاريع مختارة من زامل تك: مراكز جراحة، مستشفيات، نقاط بيع منسوجات، فاتورة إلكترونية ZATCA وإدارة مزارع — تنفيذات حقيقية في مصر."),
		Eyebrow: more("Portfolio", "أعمالنا"),
		H1:      more("Real systems, running in the field.", "أنظمة حقيقية تعمل على أرض الواقع."),
		Lead:    more("A selection of deployments across healthcare, retail, textile and agriculture.", "مختارات من مشاريعنا في الصحة والتجزئة والمنسوجات والزراعة."),
		HeroCTAs: []link{
			{Label: more("Start Your Project", "ابدأ مشروعك"), Href: "/contact-us", Kind: "primary"},
			{Label: more("Testimonials", "آراء العملاء"), Href: "/testimonials", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secCards,
				Cards: []card{
					{Icon: "🏥", Eyebrow: more("Healthcare · Odoo", "الرعاية الصحية · أودو"), Title: more("Furkan Surgical Center", "مركز فرقان للجراحة"), Body: more("Full Odoo deployment: scheduling, billing, inventory and reporting for a surgical center.", "نشر Odoo كاملًا: مواعيد وفواتير ومخزون وتقارير لمركز جراحة."), Meta: []bi{more("Odoo ERP", "أودو ERP"), more("Healthcare", "صحي")}},
					{Icon: "🏥", Eyebrow: more("Healthcare · Odoo", "الرعاية الصحية · أودو"), Title: more("Al Helal Hospital", "مستشفى الهلال"), Body: more("Hospital management on Odoo — departments, patients, pharmacy and accounting.", "إدارة المستشفى على أودو — أقسام ومرضى وصيدلية ومحاسبة."), Meta: []bi{more("Odoo ERP", "أودو ERP"), more("Healthcare", "صحي")}},
					{Icon: "🧵", Eyebrow: more("Textile · POS", "المنسوجات · نقاط البيع"), Title: more("Al-Ghazl Al-Arabi Textile POS", "نقاط بيع الغزل العربي للمنسوجات"), Body: more("Odoo POS with cutting orders for a fabric retailer.", "نقاط بيع Odoo مع أوامر قص لمتجر أقمشة."), Meta: []bi{more("Odoo POS", "نقاط بيع أودو"), more("Cutting orders", "أوامر القص")}},
					{Icon: "🧾", Eyebrow: more("Finance · Compliance", "المالية · الامتثال"), Title: more("ZATCA E-Invoice Integration", "تكامل الفاتورة الإلكترونية ZATCA"), Body: more("E-invoicing integration built to Saudi tax-authority requirements.", "تكامل الفاتورة الإلكترونية وفقًا لمتطلبات هيئة الزكاة والضريبة."), Meta: []bi{more("Integration", "تكامل"), more("Compliance", "امتثال")}},
					{Icon: "🌾", Eyebrow: more("Agriculture · Odoo", "الزراعة · أودو"), Title: more("Farm Management System", "نظام إدارة المزارع"), Body: more("Crop lifecycle, field hierarchy and cost analytics for an Upper Egypt farm.", "دورة المحصول وهيكل الحقول وتحليل التكاليف لمزرعة في صعيد مصر."), Meta: []bi{more("Agriculture ERP", "ERP زراعي"), more("Cost analytics", "تحليل التكاليف")}},
					{Icon: "🛒", Eyebrow: more("Retail · POS", "التجزئة · نقاط البيع"), Title: more("Multi-Store Retail POS", "نقاط بيع تجزئة متعددة الفروع"), Body: more("Central catalog and real-time stock across branches with offline resilience.", "كتالوج مركزي ومخزون لحظي عبر الفروع مع العمل دون إنترنت."), Meta: []bi{more("Odoo POS", "نقاط بيع أودو"), more("Multi-store", "متعدد الفروع")}},
				},
			},
		},
		Final: closing(),
	}
}

// testimonialsPage returns /testimonials.
func testimonialsPage() page {
	return page{
		Slug:    "/testimonials",
		Nav:     "testimonials",
		Title:   more("Testimonials — what our clients say", "آراء العملاء — ماذا يقول عملاؤنا"),
		Meta:    more("XamlTech rated 4.8/5 by 40+ organizations across Egypt — healthcare, retail, textile and agriculture.", "زامل تك بتقييم ٤.٨/٥ من أكثر من ٤٠ منظمة في مصر — الصحة والتجزئة والمنسوجات والزراعة."),
		Eyebrow: more("Testimonials", "آراء العملاء"),
		H1:      more("Rated 4.8 / 5 by 40+ organizations.", "تقييم ٤.٨ / ٥ من أكثر من ٤٠ منظمة."),
		Lead:    more("Real feedback from the businesses we serve.", "آراء حقيقية من الأعمال التي نخدمها."),
		Sections: []section{
			{
				Kind: secStats,
				Stats: []stat{
					{Value: more("4.8 / 5", "٤.٨ / ٥"), Label: more("Average rating", "متوسط التقييم")},
					{Value: more("40+", "٤٠+"), Label: more("Organizations", "منظمة")},
					{Value: more("98%", "٩٨٪"), Label: more("Would recommend", "يوصون بنا")},
					{Value: more("24/7", "٢٤/٧"), Label: more("Support", "دعم")},
				},
			},
			{
				Kind:  secCards,
				Label: more("What clients say", "ماذا يقول العملاء"),
				Cards: []card{
					{Icon: "⚕️", Title: more("Dr. Ahmed Hassan", "د. أحمد حسن"), Body: more("“XamlTech transformed how we run the clinic. Odoo covers appointments, billing and inventory — and their support is genuinely responsive.”", "«غيّرت زامل تك طريقة إدارتنا للعيادة. يغطّي أودو المواعيد والفواتير والمخزون — ودعمهم سريع فعلًا.»"), Eyebrow: more("Clinic Director", "مدير عيادة"), Meta: []bi{more("★★★★★", "★★★★★")}},
					{Icon: "🛍️", Title: more("Marwa Sayed", "مروة سيد"), Body: more("“The POS addons saved us hours every week. Cutting orders and stock now just work, even when the internet goes down.”", "«وفّرت إضافات نقاط البيع ساعات كل أسبوع. أوامر القص والمخزون تعمل بسلاسة حتى عند انقطاع الإنترنت.»"), Eyebrow: more("Retail Owner", "صاحبة متجر"), Meta: []bi{more("★★★★★", "★★★★★")}},
					{Icon: "🏥", Title: more("Al Helal Hospital", "مستشفى الهلال"), Body: more("“A hospital-grade Odoo deployment, delivered on time and supported long after go-live.”", "«نشر أودو بمستوى المستشفيات، سُلّم في الوقت ومُدعم بعد الإطلاق بفترة طويلة.»"), Eyebrow: more("Operations", "العمليات"), Meta: []bi{more("★★★★★", "★★★★★")}},
				},
			},
		},
		Final: finalBand(more("Join 40+ organizations.", "انضم إلى أكثر من ٤٠ منظمة."),
			more("Let's build something you'll be glad to recommend.", "لنبنِ شيئًا تفتخر بتوصيته."),
			link{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"},
			link{Label: more("See Our Work", "شاهد أعمالنا"), Href: "/portfolio", Kind: "ghost"}),
	}
}

// contactPage returns /contact-us.
func contactPage() page {
	return page{
		Slug:    "/contact-us",
		Nav:     "contact",
		Title:   more("Contact XamlTech — Aswan, Egypt", "تواصل مع زامل تك — أسوان، مصر"),
		Meta:    more("Talk to XamlTech about Odoo ERP, POS addons, IT infrastructure, web development or AI. Call, WhatsApp or email us from Aswan, Egypt.", "تواصل مع زامل تك بشأن Odoo ERP أو إضافات نقاط البيع أو البنية التحتية أو تطوير المواقع أو الذكاء الاصطناعي. اتصل أو راسلنا من أسوان، مصر."),
		Eyebrow: more("Contact Us", "تواصل معنا"),
		H1:      more("Let's talk about your project.", "لنتحدّث عن مشروعك."),
		Lead:    more("Tell us what you need. We usually reply within one business day.", "أخبرنا بما تحتاجه. عادةً نرد خلال يوم عمل واحد."),
		HeroCTAs: []link{
			{Label: more("WhatsApp", "واتساب"), Href: whatsapp, Kind: "wa", Ext: true},
			{Label: more("Call "+phoneA, "اتصل "+phoneA), Href: "tel:" + urlEncode(phoneA), Kind: "primary"},
		},
		Sections: []section{
			{
				Kind:  secFeatures,
				Label: more("Reach us", "تواصل معنا"),
				Title: more("Contact details", "بيانات التواصل"),
				Features: []feature{
					{Icon: "📞", Title: more(phoneA, phoneA), Body: more(phoneB, phoneB)},
					{Icon: "✉️", Title: more(emailMain, emailMain), Body: more("Email us anytime.", "راسلنا في أي وقت.")},
					{Icon: "📍", Title: more(addrEn, addrAr), Body: more("Sun–Thu 9AM–6PM", "الأحد–الخميس ٩ص–٦م")},
					{Icon: "💬", Title: more("WhatsApp", "واتساب"), Body: more("Fastest way to reach us.", "أسرع طريق للوصول إلينا.")},
				},
				Links: []link{
					{Label: more("Email us", "راسلنا"), Href: mailto("Website enquiry"), Kind: "primary"},
					waLink(),
				},
			},
			{
				Kind:  secProse,
				Title: more("Before you write", "قبل أن تراسلنا"),
				Blocks: []block{
					{Body: []bi{
						more("To help us reply faster, tell us your business type, the problem you want solved, and your preferred Odoo version if you know it.", "لمساعدتنا على الرد أسرع، أخبرنا بنوع عملك والمشكلة التي تريد حلها وإصدار أودو المفضل إن كنت تعرفه."),
					}},
				},
			},
		},
		Final: finalBand(more("Prefer a live demo?", "تفضّل عرضًا حيًّا؟"),
			more("Explore a running Odoo 18 instance with real data.", "استكشف نسخة Odoo 18 تعمل ببيانات حقيقية."),
			link{Label: more("Request Demo", "اطلب تجربة"), Href: "/request-demo", Kind: "primary"},
			link{Label: more("Download Apps", "حمّل التطبيقات"), Href: "/apk-download", Kind: "ghost"}),
	}
}

// demoPage returns /request-demo.
func demoPage() page {
	return page{
		Slug:    "/request-demo",
		Nav:     "demo",
		Title:   more("Request a Free Odoo Demo", "اطلب عرضًا تجريبيًا مجانيًا لأودو"),
		Meta:    more("Explore a live Odoo 18 instance with real business data. No registration, no credit card — instant demo access from XamlTech.", "استكشف نسخة Odoo 18 حية ببيانات أعمال حقيقية. بلا تسجيل وبلا بطاقة — وصول فوري للعرض من زامل تك."),
		Eyebrow: more("Free Demo", "تجربة مجانية"),
		H1:      more("Explore a live Odoo, no strings attached.", "استكشف Odoo مباشرة، بلا التزامات."),
		Lead:    more("Get instant access to a running Odoo 18 instance complete with real business data — inventory, POS, accounting and more.", "احصل على وصول فوري لنسخة Odoo 18 حية ببيانات أعمال حقيقية — المخزون ونقاط البيع والمحاسبة وغيرها."),
		HeroCTAs: []link{
			{Label: more("Open the Demo Now", "افتح العرض الآن"), Href: "https://" + demoURL, Kind: "primary", Ext: true},
			{Label: more("Talk to Us", "تحدّث معنا"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind:  secProse,
				Label: more("Demo credentials", "بيانات الدخول"),
				Title: more("Your login details", "تفاصيل الدخول"),
				Blocks: []block{
					{Body: []bi{
						more("Use the credentials below to sign in. The instance is shared for evaluation, so please don't enter real customer data.", "استخدم البيانات أدناه لتسجيل الدخول. النسخة مخصصة للتجربة، لذا يُرجى عدم إدخال بيانات عملاء حقيقية."),
					}, List: []bi{
						more("URL: "+demoURL, "الرابط: "+demoURL),
						more("Username: "+demoUser, "اسم المستخدم: "+demoUser),
						more("Password: "+demoPass, "كلمة المرور: "+demoPass),
					}},
				},
				Links: []link{{Label: more("Open Demo", "افتح العرض"), Href: "https://" + demoURL, Kind: "primary", Ext: true}},
			},
			{
				Kind:  secFeatures,
				Title: more("What's included", "ما المشمول"),
				Features: []feature{
					{Icon: "📦", Title: more("Real data", "بيانات حقيقية"), Body: more("Products, customers and transactions pre-loaded.", "منتجات وعملاء ومعاملات جاهزة.")},
					{Icon: "🛒", Title: more("Odoo POS", "نقاط بيع أودو"), Body: more("Try the point of sale end to end.", "جرّب نقطة البيع من البداية للنهاية.")},
					{Icon: "🧮", Title: more("Accounting", "المحاسبة"), Body: more("Invoices, payments and reports live.", "فواتير ومدفوعات وتقارير مباشرة.")},
					{Icon: "🔐", Title: more("Safe to explore", "آمن للاستكشاف"), Body: more("A sandbox reset regularly for evaluations.", "بيئة معزولة تُعاد ضبطها بانتظام للتجربة.")},
				},
			},
		},
		Final: finalBand(more("Want a guided walkthrough?", "تريد جولة موجّهة؟"),
			more("Our team can demo Odoo against your own workflow.", "فريقنا قد يعرض أودو وفقًا لسير عملك."),
			link{Label: more("Book a Guided Demo", "احجز عرضًا موجّهًا"), Href: "/contact-us", Kind: "primary"},
			waLink()),
	}
}

// apkPage returns /apk-download.
func apkPage() page {
	return page{
		Slug:    "/apk-download",
		Nav:     "apk",
		Title:   more("Download our Android apps", "حمّل تطبيقات أندرويد"),
		Meta:    more("Secure APK downloads for XamlTech Android apps — POS Go and Network Copilot. Links are generated per request and stay valid for 30 days.", "تحميل آمن لتطبيقات أندرويد من زامل تك — POS Go و Network Copilot. تُنشأ الروابط لكل طلب وتبقى صالحة ٣٠ يومًا."),
		Eyebrow: more("Download", "التحميل"),
		H1:      more("Our Android apps.", "تطبيقاتنا لأندرويد."),
		Lead:    more("POS Go is available on Google Play. For direct installs, request a secure link — it stays valid for 30 days and is tied to your device.", "تطبيق POS Go متاح على Google Play. للتركيب المباشر، اطلب رابطًا آمنًا صالحًا ٣٠ يومًا ومرتبطًا بجهازك."),
		HeroCTAs: []link{
			{Label: more("Get POS Go on Play", "حمّل POS Go من Play"), Href: playStoreURL, Kind: "primary", Ext: true},
			{Label: more("Network Copilot", "Network Copilot"), Href: "/networktool", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secCards,
				Cards: []card{
					{Icon: "🛒", Eyebrow: more("Android · POS", "أندرويد · نقاط البيع"), Title: more("POS Go", "POS Go"), Body: more("The companion app for your Odoo POS — offline sales, receipts and sessions.", "التطبيق المرافق لنقاط بيع أودو — مبيعات دون إنترنت وإيصالات وجلسات."), Meta: []bi{more("Android 10+", "أندرويد ١٠+"), more("Google Play", "Google Play")}, Href: playStoreURL, CTA: more("Get on Play", "حمّل من Play")},
					{Icon: "🛠️", Eyebrow: more("Android · Utilities", "أندرويد · أدوات"), Title: more("Network Copilot", "Network Copilot"), Body: more("40+ network tools and an AI assistant in one Android app.", "أكثر من ٤٠ أداة شبكات ومساعد ذكي في تطبيق واحد."), Meta: []bi{more("Android 10+", "أندرويد ١٠+"), more("Coming soon", "قريبًا")}, Href: "/networktool", CTA: more("Learn more", "اعرف المزيد")},
				},
			},
			{
				Kind:  secProse,
				Label: more("Direct install", "التركيب المباشر"),
				Title: more("Request a secure APK link", "اطلب رابط APK آمن"),
				Blocks: []block{
					{Body: []bi{
						more("Prefer a direct install? Fill out the request via WhatsApp or email and we'll send a signed APK link valid for 30 days.", "تفضّل التركيب المباشر؟ أرسل طلبك عبر واتساب أو البريد وسنرسل رابط APK موقّعًا صالحًا ٣٠ يومًا."),
					}},
				},
				Links: []link{
					{Label: more("Request link on WhatsApp", "اطلب الرابط على واتساب"), Href: whatsappWith("APK download request"), Kind: "wa", Ext: true},
					{Label: more("Email a request", "أرسل طلبًا بالبريد"), Href: mailto("APK download request"), Kind: "ghost"},
				},
			},
		},
		Final: closing(),
	}
}

// networktoolPage returns /networktool.
func networktoolPage() page {
	return page{
		Slug:    "/networktool",
		Nav:     "networktool",
		Title:   more("Network Copilot — 40+ network tools & AI assistant for Android", "Network Copilot — أكثر من ٤٠ أداة شبكات ومساعد ذكي لأندرويد"),
		Meta:    more("Network Copilot is an Android app with 40+ networking tools, 8 AI providers and 318 automated tests — a pocket toolkit for network engineers.", "Network Copilot تطبيق أندرويد يضم أكثر من ٤٠ أداة شبكات و٨ مزودي ذكاء اصطناعي و٣١٨ اختبارًا آليًا — صندوق أدوات لمهندسي الشبكات."),
		Eyebrow: more("Network Copilot", "Network Copilot"),
		H1:      more("A network toolkit in your pocket.", "صندوق أدوات شبكات في جيبك."),
		Lead:    more("Ping, DNS, ports, WHOIS, subnetting and more — plus an AI copilot that explains what you're looking at, across 8 providers.", "Ping و DNS والمنافذ و WHOIS وتقسيم الشبكات وغيرها — إضافة إلى مساعد ذكي يشرح ما تراه، عبر ٨ مزودين."),
		HeroCTAs: []link{
			{Label: more("Join the Beta", "انضم للتجربة"), Href: "/contact-us", Kind: "primary"},
			{Label: more("See Our Apps", "تصفّح تطبيقاتنا"), Href: "/apk-download", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secStats,
				Stats: []stat{
					{Value: more("40+", "٤٠+"), Label: more("Network tools", "أداة شبكات")},
					{Value: more("8", "٨"), Label: more("AI providers", "مزود ذكاء")},
					{Value: more("318", "٣١٨"), Label: more("Automated tests", "اختبار آلي")},
					{Value: more("Android 10+", "أندرويد ١٠+"), Label: more("Minimum version", "أقل إصدار")},
				},
			},
			{
				Kind:  secFeatures,
				Label: more("Tools", "الأدوات"),
				Title: more("Everything a network engineer reaches for.", "كل ما يحتاجه مهندس الشبكات."),
				Features: []feature{
					{Icon: "📡", Title: more("Reachability", "الاتصال"), Body: more("Ping, traceroute, port scanning and latency checks.", "Ping و traceroute وفحص المنافذ وقياس زمن الاستجابة.")},
					{Icon: "🔎", Title: more("DNS & WHOIS", "DNS و WHOIS"), Body: more("DNS lookups, record inspection and domain whois.", "استعلامات DNS وفحص السجلات و whois للنطاق.")},
					{Icon: "🧮", Title: more("Subnetting", "تقسيم الشبكات"), Body: more("CIDR calculators, subnet planning and address ranges.", "حاسبات CIDR وتخطيط الشبكات ونطاقات العناوين.")},
					{Icon: "🔐", Title: more("Security checks", "فحوصات الأمان"), Body: more("TLS inspection, header checks and open-port discovery.", "فحص TLS والترويسات واكتشاف المنافذ المفتوحة.")},
					{Icon: "🤖", Title: more("AI assistant", "المساعد الذكي"), Body: more("8 providers, explain-this-output and guided troubleshooting.", "٨ مزودين وشرح المخرجات وحل المشكلات الموجّه.")},
					{Icon: "🧪", Title: more("Reliable", "موثوق"), Body: more("318 automated tests across the toolchain.", "٣١٨ اختبارًا آليًا عبر سلسلة الأدوات.")},
				},
			},
		},
		Final: finalBand(more("Want early access?", "تريد الوصول المبكر؟"),
			more("Join the Network Copilot beta.", "انضم إلى تجربة Network Copilot."),
			link{Label: more("Join the Beta", "انضم للتجربة"), Href: "/contact-us", Kind: "primary"},
			waLink()),
	}
}

// insightsPage returns /insights (blog index).
func insightsPage() page {
	return page{
		Slug:    "/insights",
		Nav:     "insights",
		Title:   more("Insights — AI, Odoo and POS guides", "المدونة — أدلة الذكاء الاصطناعي وأودو ونقاط البيع"),
		Meta:    more("Practical guides on AI copilots for Odoo, POS best practices and digital transformation from the XamlTech team.", "أدلة عملية عن المساعدين الأذكياء لأودو وأفضل ممارسات نقاط البيع والتحول الرقمي من فريق زامل تك."),
		Eyebrow: more("Insights", "المدونة"),
		H1:      more("Guides from the field.", "أدلة من الميدان."),
		Lead:    more("Practical writing on AI, Odoo and running a modern point of sale.", "كتابات عملية عن الذكاء الاصطناعي وأودو وإدارة نقطة بيع حديثة."),
		Sections: []section{
			{
				Kind:  secCards,
				Title: more("Featured guides", "أدلة مميزة"),
				Cards: []card{
					{Icon: "📣", Eyebrow: more("AI · 8 min read", "ذكاء اصطناعي · قراءة ٨ دقائق"), Title: more("AI Marketing Copilot for Odoo 19", "مساعد التسويق الذكي لأودو 19"), Body: more("How to turn your Odoo catalog into product descriptions, SEO keywords and campaign copy — without a data team.", "كيف تحوّل كتالوج أودو إلى أوصاف منتجات وكلمات SEO ونصوص حملات — دون فريق بيانات."), Href: "/insights/ai-marketing-copilot", CTA: more("Read guide", "اقرأ الدليل")},
					{Icon: "🛒", Eyebrow: more("AI · 7 min read", "ذكاء اصطناعي · قراءة ٧ دقائق"), Title: more("AI POS Copilot for Odoo 19", "مساعد نقاط البيع الذكي لأودو 19"), Body: more("Real-time upsell at the till — how it reads the basket, the stock and the customer to suggest the next item.", "اقتراحات بيع فورية عند الصندوق — كيف يقرأ السلة والمخزون والعميل ليقترح العنصر التالي."), Href: "/insights/ai-pos-copilot", CTA: more("Read guide", "اقرأ الدليل")},
				},
			},
			{
				Kind:  secCards,
				Label: more("More reading", "قراءات أخرى"),
				Cards: []card{
					{Icon: "✂️", Eyebrow: more("POS", "نقاط البيع"), Title: more("Odoo POS for Textile & Cutting Orders", "نقاط بيع أودو للمنسوجات وأوامر القص"), Body: more("Building the ma_pos_cutting module and migrating it to OWL v3.", "بناء موديول ma_pos_cutting وترقيته إلى OWL v3."), Href: "/knowledge", CTA: more("Read", "اقرأ")},
					{Icon: "🧾", Eyebrow: more("POS", "نقاط البيع"), Title: more("Running Odoo POS Offline", "تشغيل نقاط بيع أودو دون إنترنت"), Body: more("Designing for network drops without losing a sale.", "التصميم لانقطاع الشبكة دون خسارة بيع."), Href: "/odoo-pos-egypt", CTA: more("Read", "اقرأ")},
				},
			},
		},
		Final: closing(),
	}
}

// guideMarketingPage returns /insights/ai-marketing-copilot.
func guideMarketingPage() page {
	return page{
		Slug:    "/insights/ai-marketing-copilot",
		Nav:     "insights",
		Title:   more("AI Marketing Copilot for Odoo 19 — a practical guide", "مساعد التسويق الذكي لأودو 19 — دليل عملي"),
		Meta:    more("A practical guide to the AI Marketing Copilot for Odoo 19: product descriptions, SEO keywords and CRM insights, multi-provider and self-hosted.", "دليل عملي لمساعد التسويق الذكي لأودو 19: أوصاف المنتجات وكلمات SEO ورؤى CRM، متعدد المزودين ومستضاف ذاتيًا."),
		Eyebrow: more("Guide · 8 min read", "دليل · قراءة ٨ دقائق"),
		H1:      more("AI Marketing Copilot for Odoo 19.", "مساعد التسويق الذكي لأودو 19."),
		Lead:    more("Turn the data you already have in Odoo into marketing that runs itself.", "حوّل البيانات الموجودة لديك في أودو إلى تسويق يعمل تلقائيًا."),
		Sections: []section{
			{Kind: secProse, Blocks: []block{
				{Heading: more("The problem", "المشكلة"), Body: []bi{
					more("Most small businesses have a rich product catalog in Odoo and almost no marketing copy. Writing descriptions, finding keywords and building campaigns is slow, and outsourcing it is expensive.", "لدى معظم الأنشطة التجارية الصغيرة كتالوج غني في أودو ومحتوى تسويقي شبه معدوم. كتابة الأوصاف وإيجاد الكلمات وبناء الحملات بطيء، وتوكيله مكلف."),
				}},
				{Heading: more("What the copilot does", "ما الذي يفعله المساعد"), List: []bi{
					more("Generates product descriptions and SEO keywords from each product's attributes.", "يولّد أوصاف المنتجات وكلمات SEO من سمات كل منتج."),
					more("Drafts campaign and email copy aligned to your catalog and audience.", "يكتب مسودات الحملات والبريد بما يتوافق مع كتالوجك وجمهورك."),
					more("Surfaces CRM insights — which segments respond, what to promote next.", "يقدّم رؤى CRM — أي الشرائح تستجيب وما التالي للترويج."),
				}},
				{Heading: more("Why self-hosted matters", "لماذا الاستضافة الذاتية مهمة"), Body: []bi{
					more("The copilot supports multiple AI providers and runs on your own server, so customer and product data never leave your infrastructure — a requirement for many Egyptian businesses.", "يدعم المساعد عدة مزودي ذكاء اصطناعي ويعمل على سيرفرك، لذا لا تغادر بيانات العملاء والمنتجات بنيتك التحتية — وهو شرط لكثير من الشركات المصرية."),
				}},
			}},
		},
		Final: finalBand(more("Try the AI Marketing Copilot", "جرّب مساعد التسويق الذكي"),
			more("Available on the Odoo Apps Store for Odoo 19.", "متاح على متجر Odoo الرسمي لأودو 19."),
			link{Label: more("View Addon", "عرض الإضافة"), Href: odooAppsURL, Kind: "primary", Ext: true},
			link{Label: more("Talk to Us", "تحدّث معنا"), Href: "/contact-us", Kind: "ghost"}),
	}
}

// guidePOSPage returns /insights/ai-pos-copilot.
func guidePOSPage() page {
	return page{
		Slug:    "/insights/ai-pos-copilot",
		Nav:     "insights",
		Title:   more("AI POS Copilot for Odoo 19 — a practical guide", "مساعد نقاط البيع الذكي لأودو 19 — دليل عملي"),
		Meta:    more("How the AI POS Copilot suggests upsells at the till in real time, using the basket and stock on hand, cache-first and offline-aware.", "كيف يقترح مساعد نقاط البيع الذكي عناصر إضافية عند الصندوق لحظيًا، بناءً على السلة والمخزون، ذاكرة أولًا ويدعم العمل دون إنترنت."),
		Eyebrow: more("Guide · 7 min read", "دليل · قراءة ٧ دقائق"),
		H1:      more("AI POS Copilot for Odoo 19.", "مساعد نقاط البيع الذكي لأودو 19."),
		Lead:    more("The right suggestion at the right moment — at the till, not in a quarterly report.", "الاقتراح الصحيح في اللحظة الصحيحة — عند الصندوق، لا في تقرير ربع سنوي."),
		Sections: []section{
			{Kind: secProse, Blocks: []block{
				{Heading: more("The opportunity", "الفرصة"), Body: []bi{
					more("Upsell dies at the till because cashiers are busy and customers are impatient. The AI POS Copilot watches the basket and stock and proposes one relevant add-on, instantly.", "يفوت البيع الإضافي عند الصندوق لأن الكاشير مشغول والعميل مستعجل. يراقب مساعد نقاط البيع الذكي السلة والمخزون ويقترح إضافة واحدة مناسبة، فورًا."),
				}},
				{Heading: more("How it works", "كيف يعمل"), List: []bi{
					more("Reads the current basket and the products' on-hand stock.", "يقرأ السلة الحالية والمخزون المتاح للمنتجات."),
					more("Ranks suggestions by margin, availability and basket affinity.", "يرتّب الاقتراحات حسب الهامش والتوفر والملاءمة مع السلة."),
					more("Caches results so it keeps working when the connection drops.", "يخزّن النتائج مؤقتًا ليستمر في العمل عند انقطاع الاتصال."),
				}},
				{Heading: more("Designed for the store", "مصمّم للمتجر"), Body: []bi{
					more("Cache-first and offline-aware, so it never slows the till down or blocks a sale when the network is unstable — a common reality in Egyptian retail.", "ذاكرة أولًا ويدعم العمل دون إنترنت، فلا يبطئ الصندوق ولا يمنع بيعًا عند اضطراب الشبكة — واقع شائع في التجزئة المصرية."),
				}},
			}},
		},
		Final: finalBand(more("Try the AI POS Copilot", "جرّب مساعد نقاط البيع الذكي"),
			more("Available on the Odoo Apps Store for Odoo 19.", "متاح على متجر Odoo الرسمي لأودو 19."),
			link{Label: more("View Addon", "عرض الإضافة"), Href: odooAppsURL, Kind: "primary", Ext: true},
			link{Label: more("Talk to Us", "تحدّث معنا"), Href: "/contact-us", Kind: "ghost"}),
	}
}

// knowledgePage returns /knowledge.
func knowledgePage() page {
	return page{
		Slug:    "/knowledge",
		Nav:     "knowledge",
		Title:   more("Knowledge — Odoo POS development notes", "المعرفة — ملاحظات تطوير نقاط بيع أودو"),
		Meta:    more("Technical notes on building Odoo POS modules — the textile cutting module, OWL v3 migration and offline POS design, from the XamlTech team.", "ملاحظات تقنية حول بناء موديولات نقاط بيع أودو — موديول قص المنسوجات والترقية إلى OWL v3 وتصميم نقاط البيع دون إنترنت، من فريق زامل تك."),
		Eyebrow: more("Knowledge Base", "قاعدة المعرفة"),
		H1:      more("Notes from building real Odoo modules.", "ملاحظات من بناء موديولات أودو حقيقية."),
		Lead:    more("Technical write-ups for developers and partners.", "كتابات تقنية للمطورين والشركاء."),
		Sections: []section{
			{Kind: secProse, Blocks: []block{
				{Heading: more("Building ma_pos_cutting for textile retailers", "بناء ma_pos_cutting لتجار المنسوجات"), Body: []bi{
					more("Textile retail has a problem generic POS software ignores: customers buy fabric by the metre and the sale often triggers a cutting order for a workshop. We built ma_pos_cutting so the cutting job is opened from the POS line itself, tracked through production and linked back to the sale and inventory.", "لدى تجارة المنسوجات مشكلة تتجاهلها برامج نقاط البيع العامة: يشتري العملاء القماش بالمتر وغالبًا ما يولّد البيع أمر قص للورشة. بنينا ma_pos_cutting بحيث يُفتح أمر القص من سطر نقطة البيع نفسه، ويُتابع عبر الإنتاج ويُربط بالبيع والمخزون."),
				}},
				{Heading: more("Migrating to OWL v3", "الترقية إلى OWL v3"), Body: []bi{
					more("Odoo 17+ moved the frontend to OWL v3. The migration meant rewriting screen components, adopting the new reactivity model and re-testing the POS flows end to end — worth it, because the screen is faster and the code is cleaner.", "انتقلت أودو 17+ إلى OWL v3 في الواجهة. تطلّبت الترقية إعادة كتابة مكوّنات الشاشة وتبنّي نموذج التفاعل الجديد وإعادة اختبار مسارات نقاط البيع من البداية للنهاية — وهو جدير بالجهد، لأن الشاشة أسرع والكود أنظف."),
				}},
				{Heading: more("Designing for offline", "التصميم للعمل دون إنترنت"), List: []bi{
					more("Queue writes locally and replay them idempotently when back online.", "ضع الكتابات في قائمة محلية وأعد تشغيلها بأمان عند العودة."),
					more("Never block a sale on a network call — degrade, don't fail.", "لا تحجز عملية بيع على نداء شبكة — قلّل الأداء ولا تفشل."),
					more("Give each client command a stable idempotency key.", "امنح كل أمر عميل مفتاح تكرار ثابتًا."),
				}},
			}},
		},
		Final: finalBand(more("Building on Odoo?", "تتبنى أودو؟"),
			more("We partner with integrators and in-house teams.", "نتعاون مع المندوبين والفرق الداخلية."),
			link{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"},
			link{Label: more("Read Insights", "اقرأ المدونة"), Href: "/insights", Kind: "ghost"}),
	}
}

// privacyPage returns /privacy-policy.
func privacyPage() page {
	return page{
		Slug:    "/privacy-policy",
		Nav:     "",
		Title:   more("Privacy Policy", "سياسة الخصوصية"),
		Meta:    more("How XamlTech collects, uses and protects your data.", "كيف تجمع زامل تك بياناتك وتستخدمها وتحميها."),
		Eyebrow: more("Legal", "قانوني"),
		H1:      more("Privacy Policy", "سياسة الخصوصية"),
		Lead:    more("Your privacy matters to us.", "خصوصيتك تهمّنا."),
		Robots:  "noindex,follow",
		Sections: []section{
			{Kind: secProse, Blocks: []block{
				{Heading: more("Data we collect", "البيانات التي نجمعها"), Body: []bi{
					more("We collect only what you provide to us directly: contact details you submit through forms, email or WhatsApp, and basic technical information needed to serve this website.", "نجمع فقط ما تقدّمه لنا مباشرة: بيانات التواصل التي ترسلها عبر النماذج أو البريد أو واتساب، والمعلومات التقنية الأساسية اللازمة لتقديم هذا الموقع."),
				}},
				{Heading: more("How we use it", "كيف نستخدمها"), List: []bi{
					more("To reply to your enquiries and provide the services you request.", "للرد على استفساراتك وتقديم الخدمات التي تطلبها."),
					more("To improve this website and our services.", "لتحسين هذا الموقع وخدماتنا."),
					more("To meet legal or accounting obligations.", "للوفاء بالالتزامات القانونية أو المحاسبية."),
				}},
				{Heading: more("Sharing", "المشاركة"), Body: []bi{
					more("We do not sell your personal data. We share it only with service providers who help us operate, and only as needed.", "لا نبيع بياناتك الشخصية. نشاركها فقط مع مزودي الخدمات الذين يساعدوننا في التشغيل، وبالحد الضروري."),
				}},
				{Heading: more("Your choices", "خياراتك"), Body: []bi{
					more("You can ask us to access, correct or delete your data at any time by emailing "+emailMain+".", "يمكنك أن تطلب منا الوصول لبياناتك أو تصحيحها أو حذفها في أي وقت عبر مراسلة "+emailMain+"."),
				}},
			}},
		},
		Final: finalBand(more("Questions about privacy?", "أسئلة حول الخصوصية؟"),
			more("Contact us and we'll be glad to help.", "تواصل معنا وسنُسعد بمساعدتك."),
			link{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"}),
	}
}

// termsPage returns /terms.
func termsPage() page {
	return page{
		Slug:    "/terms",
		Nav:     "",
		Title:   more("Terms of Service", "شروط الخدمة"),
		Meta:    more("Terms of service for using the XamlTech website, demos and addons.", "شروط استخدام موقع زامل تك والعروض التجريبية والإضافات."),
		Eyebrow: more("Legal", "قانوني"),
		H1:      more("Terms of Service", "شروط الخدمة"),
		Lead:    more("The terms that govern use of our website and services.", "الشروط التي تحكم استخدام موقعنا وخدماتنا."),
		Robots:  "noindex,follow",
		Sections: []section{
			{Kind: secProse, Blocks: []block{
				{Heading: more("Use of the website", "استخدام الموقع"), Body: []bi{
					more("By using this website you agree to use it lawfully and not to interfere with its operation or security.", "باستخدامك هذا الموقع فإنك توافق على استخدامه بشكل قانوني وعدم التعرض لتشغيله أو أمنه."),
				}},
				{Heading: more("Demos", "العروض التجريبية"), Body: []bi{
					more("Our Odoo demo is provided for evaluation only, as-is and without warranty, and is reset periodically. Do not enter real or sensitive data.", "يُقدَّم عرض أودو التجريبي للتقييم فقط، كما هو ودون ضمان، ويُعاد ضبطه دوريًا. لا تُدخل بيانات حقيقية أو حساسة."),
				}},
				{Heading: more("Addon licensing", "ترخيص الإضافات"), Body: []bi{
					more("Odoo addons are licensed under their terms on the Odoo Apps Store. One-time purchases include the stated support window.", "تُرخَّص إضافات أودو وفق شروطها على متجر Odoo. تشمل عمليات الشراء لمرة واحدة مدة الدعم المذكورة."),
				}},
				{Heading: more("Liability", "المسؤولية"), Body: []bi{
					more("Services are provided on a best-effort basis. To the extent permitted by law, XamlTech is not liable for indirect or consequential losses.", "تُقدَّم الخدمات على أساس بذل أقصى جهد. وبالقدر الذي يسمح به القانون، لا تتحمل زامل تك مسؤولية الخسائر غير المباشرة أو التبعية."),
				}},
			}},
		},
		Final: finalBand(more("Questions about our terms?", "أسئلة حول شروطنا؟"),
			more("Get in touch.", "تواصل معنا."),
			link{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"}),
	}
}

// registry maps a language-neutral path to a page builder. Paths never carry
// the /ar prefix — the router strips it and passes the lang separately.
func registry() map[string]func() page {
	return map[string]func() page{
		"/":                              home,
		"/about":                         about,
		"/solutions":                     solutionsHub,
		"/solutions/erp":                 erpPage,
		"/solutions/it-infrastructure":   itPage,
		"/solutions/web":                 webPage,
		"/odoo-pos-egypt":                posPage,
		"/erp-agriculture-egypt":         agriPage,
		"/ai-solutions":                  aiPage,
		"/addons-store":                  addonsPage,
		"/pricing":                       pricingPage,
		"/portfolio":                     portfolioPage,
		"/testimonials":                  testimonialsPage,
		"/insights":                      insightsPage,
		"/insights/ai-marketing-copilot": guideMarketingPage,
		"/insights/ai-pos-copilot":       guidePOSPage,
		"/knowledge":                     knowledgePage,
		"/networktool":                   networktoolPage,
		"/contact-us":                    contactPage,
		"/request-demo":                  demoPage,
		"/apk-download":                  apkPage,
		"/privacy-policy":                privacyPage,
		"/terms":                         termsPage,
	}
}
