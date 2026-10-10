package mainsite

// This file holds shared content helpers and the first half of the page set.

// closing is the "Ready to transform your business?" band every page ends with.
func closing() section {
	return section{
		Kind:  secCTA,
		Title: more("Ready to transform your business?", "مستعد لتحويل أعمالك رقميًا؟"),
		Lead:  more("Get started with XamlTech solutions today.", "ابدأ مع حلول زامل تك اليوم."),
		Links: []link{
			{Label: more("Get in Touch", "تواصل معنا"), Href: "/contact-us", Kind: "primary"},
			{Label: more("Request Demo", "اطلب تجربة"), Href: "/request-demo", Kind: "ghost"},
			{Label: more("WhatsApp", "واتساب"), Href: whatsapp, Kind: "wa", Ext: true},
		},
	}
}

// finalBand builds a labelled closing CTA with custom copy.
func finalBand(title, lead bi, links ...link) section {
	return section{Kind: secCTA, Title: title, Lead: lead, Links: links}
}

// quoteLink is the "Get a Quote" CTA used across the solution pages.
func quoteLink() link {
	return link{Label: more("Get a Quote", "اطلب عرض سعر"), Href: "/contact-us", Kind: "primary"}
}

func waLink() link {
	return link{Label: more("WhatsApp", "واتساب"), Href: whatsapp, Kind: "wa", Ext: true}
}

// home returns the landing page.
func home() page {
	return page{
		Slug:    "/",
		Nav:     "home",
		Title:   more("Odoo ERP, POS & IT Solutions in Egypt", "أنظمة Odoo ERP ونقاط البيع وحلول IT في مصر"),
		Meta:    more("Odoo ERP specialists, POS addons, network infrastructure, CCTV, web development, cloud solutions and digital transformation. Published modules on the Odoo Apps Store. Based in Aswan, Egypt.", "متخصصون في أنظمة Odoo ERP وإضافات نقاط البيع والبنية التحتية والكاميرات وتطوير المواقع والحلول السحابية والتحول الرقمي. موديولات منشورة على متجر Odoo الرسمي — أسوان، مصر."),
		Eyebrow: more("XamlTech · Aswan, Egypt · Since 2024", "زامل تك · أسوان، مصر · منذ ٢٠٢٤"),
		H1:      more("More than technology — we build digital growth.", "أكثر من تقنية — نبني نموًا رقميًا حقيقيًا."),
		Lead:    more("Odoo ERP specialists, network infrastructure, CCTV, web development, cloud solutions and digital transformation — with published modules on the official Odoo Apps Store.", "متخصصون في أنظمة Odoo ERP، الشبكات، كاميرات المراقبة، تطوير المواقع، الحلول السحابية والتحول الرقمي — بموديولات منشورة على متجر Odoo الرسمي."),
		HeroCTAs: []link{
			{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "primary"},
			{Label: more("Free Demo", "تجربة مجانية"), Href: "/request-demo", Kind: "ghost"},
		},
		HeroTiles: []feature{
			{Icon: "🎯", Title: more("Smart Strategy", "استراتيجية ذكية")},
			{Icon: "⚙️", Title: more("Reliable Systems", "أنظمة موثوقة")},
			{Icon: "📈", Title: more("Better Performance", "أداء أفضل")},
			{Icon: "🖥️", Title: more("Scalable Infrastructure", "بنية قابلة للتوسع")},
			{Icon: "🌐", Title: more("Stronger Digital Presence", "حضور رقمي أقوى")},
		},
		Sections: []section{
			{
				Kind: secStats,
				Stats: []stat{
					{Value: more("8+", "٨+"), Label: more("Odoo addons published", "إضافة Odoo منشورة")},
					{Value: more("40+", "٤٠+"), Label: more("organizations served", "منظمة نخدمها")},
					{Value: more("8", "٨"), Label: more("years Odoo (v12–v19)", "سنوات خبرة Odoo")},
					{Value: more("Aswan", "أسوان"), Label: more("based · Upper Egypt", "مقرنا · صعيد مصر")},
				},
			},
			{
				Kind:  secCards,
				Label: more("Focus Areas", "مجالات التخصص"),
				Title: more("Everything a growing business needs.", "كل ما تحتاجه الأعمال النامية."),
				Lead:  more("Six practices, one team — from the ERP that runs the business to the network that carries it.", "ستة مجالات، فريق واحد — من نظام ERP الذي يدير العمل إلى الشبكة التي تحمله."),
				Cards: []card{
					{Icon: "📊", Title: more("Odoo ERP", "أنظمة أودو ERP"), Body: more("ERP, POS, accounting, HR — implementation, customization and integration.", "ERP، نقاط بيع، محاسبة، موارد بشرية — تنفيذ وتخصيص وتكامل."), Href: "/solutions/erp"},
					{Icon: "🔌", Title: more("Networks & CCTV", "الشبكات والمراقبة"), Body: more("LAN/WAN, servers, VPN, IP cameras and 24/7 monitoring.", "شبكات، سيرفرات، VPN، كاميرات IP ومراقبة على مدار الساعة."), Href: "/solutions/it-infrastructure"},
					{Icon: "🌐", Title: more("Web Development", "تطوير المواقع"), Body: more("Corporate sites, e-commerce, web apps and SEO.", "مواقع الشركات، المتاجر، تطبيقات الويب وتحسين محركات البحث."), Href: "/solutions/web"},
					{Icon: "✨", Title: more("AI Copilots", "مساعدو الذكاء الاصطناعي"), Body: more("AI marketing optimization and real-time POS upsell, inside Odoo 19.", "تحسين التسويق بالذكاء الاصطناعي واقتراحات بيع فورية داخل Odoo 19."), Href: "/ai-solutions"},
					{Icon: "🌾", Title: more("Agriculture ERP", "ERP للزراعة"), Body: more("Farm hierarchy, crops, work orders, harvest and cost analytics.", "هيكل المزارع، المحاصيل، أوامر العمل، الحصاد وتحليل التكاليف."), Href: "/erp-agriculture-egypt"},
					{Icon: "🛒", Title: more("Addons Store", "متجر الإضافات"), Body: more("Premium Odoo 19 modules, published and production-tested.", "موديولات Odoo 19 متميزة، منشورة ومُختبرة في الإنتاج."), Href: "/addons-store"},
				},
			},
			{
				Kind:  secFeatures,
				Label: more("Why XamlTech", "لماذا زامل تك؟"),
				Title: more("Why choose XamlTech?", "لماذا تختار زامل تك؟"),
				Features: []feature{
					{Icon: "✅", Title: more("Integrated Solutions", "حلول متكاملة"), Body: more("One team for software, network and hardware — no vendor juggling.", "فريق واحد للبرمجيات والشبكات والأجهزة — دون تنسيق بين موردين.")},
					{Icon: "💎", Title: more("Premium Quality", "جودة عالية"), Body: more("Production-tested modules with automated tests, not demos.", "موديولات مُختبرة في الإنتاج باختبارات آلية، لا مجرد عروض.")},
					{Icon: "🎯", Title: more("Custom Solutions", "حلول مخصصة"), Body: more("Tailored to your workflow — cutting orders, agriculture, revenue collection.", "مصممة لسير عملك — أوامر القص، الزراعة، تحصيل الإيرادات.")},
					{Icon: "🕐", Title: more("24/7 Support", "دعم ٢٤/٧"), Body: more("Reachable on WhatsApp and email, with on-site visits in Upper Egypt.", "متاحون على واتساب والبريد، مع زيارات ميدانية في صعيد مصر.")},
				},
			},
			{
				Kind:  secSteps,
				Label: more("Methodology", "منهجية العمل"),
				Title: more("Implementation process", "منهجية التنفيذ"),
				Steps: []step{
					{Title: more("Requirements analysis", "تحليل المتطلبات"), Body: more("We map your real workflow before writing a line of code.", "نفهم سير عملك الفعلي قبل كتابة أي كود.")},
					{Title: more("Design & setup", "التصميم والإعداد"), Body: more("Modules, data model, roles, receipts and hardware configured.", "إعداد الموديولات وبنية البيانات والصلاحيات والفواتير والأجهزة.")},
					{Title: more("Development & integration", "التطوير والتكامل"), Body: more("Custom modules and third-party integrations built and connected.", "بناء الموديولات المخصصة والتكامل مع الأنظمة الخارجية.")},
					{Title: more("UAT & acceptance", "الاختبار والقبول"), Body: more("User acceptance testing and performance validation.", "اختبار القبول من المستخدمين والتحقق من الأداء.")},
					{Title: more("Training & support", "التدريب والدعم"), Body: more("Hands-on training, documentation and ongoing support.", "تدريب عملي ووثائق ودعم مستمر.")},
				},
			},
			{
				Kind:  secCards,
				Label: more("Addons Store", "متجر الإضافات"),
				Title: more("Our published addons", "إضافاتنا المنشورة"),
				Lead:  more("Production-tested Odoo 19 modules, available on the official Odoo Apps Store.", "موديولات Odoo 19 مُختبرة، متاحة على متجر Odoo الرسمي."),
				Cards: []card{
					{Icon: "✂️", Badge: more("Best seller", "الأكثر مبيعًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Cutting Orders", "أوامر القص"), Body: more("Fabric cutting managed directly from POS.", "إدارة قص الأقمشة مباشرة من نقطة البيع."), Price: more("€149", "€149"), Rating: "4.9", Href: "/addons-store"},
					{Icon: "📦", Badge: more("Top rated", "الأعلى تقييمًا"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Stock Real-Time", "المخزون الفوري"), Body: more("Live stock sync across every POS session.", "مزامنة المخزون الفورية عبر جميع الجلسات."), Price: more("€129", "€129"), Rating: "4.8", Href: "/addons-store"},
					{Icon: "🔐", Badge: more("Essential", "أساسي"), Eyebrow: more("Point of Sale", "نقاط البيع"), Title: more("POS Security Framework", "إطار أمان نقاط البيع"), Body: more("Manager PIN and discount limits.", "رقم PIN للمدير وحدود الخصم."), Price: more("€99", "€99"), Rating: "4.7", Href: "/addons-store"},
				},
				Links: []link{{Label: more("View all addons", "عرض كل الإضافات"), Href: "/addons-store", Kind: "ghost"}},
			},
		},
		Final: closing(),
	}
}

// about returns the studio/about page.
func about() page {
	return page{
		Slug:    "/about",
		Nav:     "about",
		Title:   more("About XamlTech — an Egyptian software studio", "عن زامل تك — شركة برمجيات مصرية"),
		Meta:    more("XamlTech is an Egyptian software studio building Odoo ERP systems, POS addons, AI copilots and IT infrastructure from Aswan, Upper Egypt.", "زامل تك شركة برمجيات مصرية تبني أنظمة Odoo ERP وإضافات نقاط البيع والمساعدين الأذكياء والبنية التحتية من أسوان في صعيد مصر."),
		Eyebrow: more("About us", "من نحن"),
		H1:      more("Software, systems and infrastructure — built here in Upper Egypt.", "برمجيات وأنظمة وبنية تحتية — صُنعت هنا في صعيد مصر."),
		Lead:    more("We are an Odoo-focused studio in Aswan. We implement ERP, build custom POS modules, design network and CCTV infrastructure, and ship AI copilots — all in Arabic and English, and all production-tested.", "نحن شركة متخصصة في Odoo بأسوان. ننفّذ أنظمة ERP ونبني موديولات نقاط بيع مخصصة ونصمم شبكات وكاميرات ونطلق مساعدين أذكياء — بالعربية والإنجليزية ومُختبرة في الإنتاج."),
		HeroCTAs: []link{
			{Label: more("See our work", "شاهد أعمالنا"), Href: "/portfolio", Kind: "primary"},
			{Label: more("Contact us", "تواصل معنا"), Href: "/contact-us", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind:  secProse,
				Title: more("What we do", "ما نقوم به"),
				Blocks: []block{
					{Body: []bi{
						more("XamlTech works where software meets the physical world. We implement Odoo ERP for hospitals, retailers, textile traders and farms; we build custom modules published on the official Odoo Apps Store; we design and install the networks, servers and CCTV systems those businesses run on.", "تعمل زامل تك في نقطة التقاء البرمجيات بالعالم المادي. ننفّذ أنظمة Odoo ERP للمستشفيات والمتاجر وتجار الأقمشة والمزارع، ونبني موديولات مخصصة منشورة على متجر Odoo الرسمي، ونصمم ونثبّت الشبكات والسيرفرات وأنظمة المراقبة التي تعتمد عليها هذه الأعمال."),
						more("We are a small, senior team. That means the person who designs your system also builds and supports it — no hand-offs, no account-manager telephone game.", "نحن فريق صغير من ذوي الخبرة. يعني ذلك أن من يصمم نظامك هو نفسه من يبنيه ويدعمه — بلا تسليم لأطراف أخرى ولا وسطاء."),
					}},
					{Heading: more("Our principles", "مبادئنا"), List: []bi{
						more("Real workflows over demos — everything we ship is used in production.", "سير عمل حقيقي لا عروض — كل ما نطلقه مستخدم في الإنتاج."),
						more("Arabic-first — RTL, Arabic receipts and interfaces, not an afterthought.", "العربية أولًا — واجهات وفواتير بالعربية، وليست إضافة لاحقة."),
						more("Own the whole stack — software, network and hardware from one team.", "نمثل المنظومة كاملة — برمجيات وشبكات وأجهزة من فريق واحد."),
						more("Support that stays reachable — WhatsApp, email and on-site visits.", "دعم يمكن الوصول إليه — واتساب وبريد وزيارات ميدانية."),
					}},
				},
			},
			{
				Kind: secStats,
				Stats: []stat{
					{Value: more("180+", "١٨٠+"), Label: more("public repositories", "مستودع عام")},
					{Value: more("8", "٨"), Label: more("Odoo addons published", "إضافة Odoo منشورة")},
					{Value: more("v12–v19", "v12–v19"), Label: more("Odoo versions built", "إصدارات أودو")},
					{Value: more("40+", "٤٠+"), Label: more("organizations served", "منظمة نخدمها")},
				},
			},
			{
				Kind:  secCards,
				Label: more("Team", "الفريق"),
				Title: more("The people behind the work", "الأشخاص خلف العمل"),
				Cards: []card{
					{Icon: "👨‍💻", Title: more(founderName, "محمود أمين"), Body: more("Founder & Lead Developer — Odoo, full-stack and infrastructure.", "المؤسس والمطوّر الرئيسي — أودو وتطوير متكامل وبنية تحتية."), Href: mailto(""), CTA: more("Email", "راسلني")},
					{Icon: "📞", Title: more("Osama M Salah", "أسامة صلاح"), Body: more(phoneB, phoneB), Href: "tel:" + urlEncode(phoneB), CTA: more("Call", "اتصل")},
				},
			},
		},
		Final: closing(),
	}
}

// solutionsHub returns the /solutions overview.
func solutionsHub() page {
	return page{
		Slug:     "/solutions",
		Nav:      "solutions",
		Title:    more("Solutions — ERP, IT infrastructure, web & AI", "الحلول — ERP والبنية التحتية والمواقع والذكاء الاصطناعي"),
		Meta:     more("XamlTech solutions: Odoo ERP implementation, IT infrastructure and CCTV, professional web development, AI copilots and agriculture ERP.", "حلول زامل تك: تنفيذ Odoo ERP، البنية التحتية والكاميرات، تطوير المواقع، المساعدون الأذكياء و ERP للزراعة."),
		Eyebrow:  more("Solutions", "الحلول"),
		H1:       more("One partner for the software, the network and everything between.", "شريك واحد للبرمجيات والشبكات وكل ما بينهما."),
		Lead:     more("Five practices that cover the technology a real business runs on.", "خمسة مجالات تغطي التقنية التي تعتمد عليها الأعمال الحقيقية."),
		HeroCTAs: []link{quoteLink(), waLink()},
		Sections: []section{
			{
				Kind: secCards,
				Cards: []card{
					{Icon: "📊", Eyebrow: more("ERP & Odoo", "أنظمة أودو"), Title: more("Complete ERP for your business", "نظام ERP متكامل لأعمالك"), Body: more("Implementation, customization, integration and BI — sales to accounting, inventory to HR.", "تنفيذ وتخصيص وتكامل وذكاء أعمال — من البيع للمحاسبة ومن المخزون للموارد البشرية."), Href: "/solutions/erp", CTA: more("Explore", "اكتشف")},
					{Icon: "🖧", Eyebrow: more("IT Infrastructure", "البنية التحتية"), Title: more("Secure & reliable infrastructure", "بنية تحتية آمنة وموثوقة"), Body: more("LAN/WAN, wireless, firewall, servers, CCTV and monitoring.", "شبكات، لاسلكي، جدار ناري، سيرفرات، كاميرات ومراقبة."), Href: "/solutions/it-infrastructure", CTA: more("Explore", "اكتشف")},
					{Icon: "🌐", Eyebrow: more("Web Development", "تطوير المواقع"), Title: more("A professional website that delivers", "موقع احترافي يحقق نتائج"), Body: more("Corporate sites, e-commerce, web apps and SEO.", "مواقع شركات، متاجر إلكترونية، تطبيقات ويب وتحسين محركات البحث."), Href: "/solutions/web", CTA: more("Explore", "اكتشف")},
					{Icon: "✨", Eyebrow: more("AI Copilots", "المساعدون الأذكياء"), Title: more("AI copilots for Odoo 19", "مساعدون أذكياء لأودو 19"), Body: more("Marketing optimization and real-time POS upsell, self-hosted.", "تحسين التسويق واقتراحات بيع فورية داخل أودو، مستضاف ذاتيًا."), Href: "/ai-solutions", CTA: more("Explore", "اكتشف")},
					{Icon: "🌾", Eyebrow: more("Agriculture ERP", "ERP للزراعة"), Title: more("Farm management for Upper Egypt", "إدارة المزارع لصعيد مصر"), Body: more("Farm hierarchy, crops, work orders, harvest and cost analytics.", "هيكل المزارع، المحاصيل، أوامر العمل، الحصاد وتحليل التكاليف."), Href: "/erp-agriculture-egypt", CTA: more("Explore", "اكتشف")},
					{Icon: "🛒", Eyebrow: more("Addons Store", "متجر الإضافات"), Title: more("Premium Odoo addons", "إضافات Odoo متميزة"), Body: more("Published, production-tested modules for POS, AI, agriculture and more.", "موديولات منشورة ومُختبرة لنقاط البيع والذكاء الاصطناعي والزراعة وغيرها."), Href: "/addons-store", CTA: more("Browse", "تصفّح")},
				},
			},
		},
		Final: closing(),
	}
}

// erpPage returns /solutions/erp.
func erpPage() page {
	return page{
		Slug:    "/solutions/erp",
		Nav:     "solutions",
		Title:   more("Odoo ERP Development & Custom Module Implementation in Egypt", "تطوير وتخصيص أنظمة Odoo ERP في مصر"),
		Meta:    more("Odoo ERP implementation, customization, integration, BI, training and support in Egypt — sales, accounting, inventory and HR in one unified system.", "تنفيذ وتخصيص وتكامل وذكاء أعمال وتدريب ودعم Odoo ERP في مصر — البيع والمحاسبة والمخزون والموارد البشرية في نظام واحد."),
		Eyebrow: more("ERP Solutions", "حلول ERP"),
		H1:      more("Complete ERP for your business.", "نظام ERP متكامل لأعمالك."),
		Lead:    more("Are your operations disorganized? We implement Odoo ERP to manage everything — from sales to accounting, inventory to HR — in one unified system.", "هل عملياتك غير منظمة؟ ننفّذ Odoo ERP لإدارة كل شيء — من البيع للمحاسبة ومن المخزون للموارد البشرية — في نظام موحّد."),
		HeroCTAs: []link{
			{Label: more("Try Live Demo", "جرّب العرض الحي"), Href: "/request-demo", Kind: "primary"},
			quoteLink(),
		},
		Sections: []section{
			{
				Kind:  secFeatures,
				Label: more("Our ERP Services", "خدماتنا في ERP"),
				Title: more("From analysis to ongoing support.", "من التحليل حتى الدعم المستمر."),
				Features: []feature{
					{Icon: "🔧", Title: more("Implementation", "التنفيذ"), Body: more("Full Odoo deployment — requirements analysis, configuration, data migration, testing and go-live support.", "نشر Odoo كاملًا — تحليل المتطلبات والإعداد وترحيل البيانات والاختبار ودعم الإطلاق.")},
					{Icon: "🧩", Title: more("Customization", "التخصيص"), Body: more("Custom module development, workflow automation, report builder and UI/UX tailoring.", "تطوير موديولات مخصصة وأتمتة سير العمل ومُنشئ التقارير وتخصيص الواجهات.")},
					{Icon: "🔗", Title: more("Integration", "التكامل"), Body: more("Payment gateways, shipping, ZATCA e-invoicing, APIs and data sync.", "بوابات الدفع والشحن والفاتورة الإلكترونية ZATCA وواجهات البرمجة ومزامنة البيانات.")},
					{Icon: "📊", Title: more("Business Intelligence", "ذكاء الأعمال"), Body: more("Custom dashboards, KPI tracking, financial reports and analytics.", "لوحات معلومات مخصصة وتتبع مؤشرات الأداء وتقارير مالية وتحليلات.")},
					{Icon: "🎓", Title: more("Training", "التدريب"), Body: more("User and admin training, video tutorials and documentation in English and Arabic.", "تدريب المستخدمين والمشرفين ودروس مصوّرة ووثائق بالعربية والإنجليزية.")},
					{Icon: "🛠️", Title: more("Support", "الدعم"), Body: more("24/7 support, bug fixes, performance optimization, security patches and upgrades.", "دعم على مدار الساعة وإصلاحات وتحسين أداء وتحديثات أمنية وترقيات.")},
				},
			},
			{
				Kind:  secSteps,
				Label: more("Methodology", "المنهجية"),
				Title: more("Implementation process", "منهجية التنفيذ"),
				Steps: []step{
					{Title: more("REQ Analysis", "تحليل المتطلبات"), Body: more("Analyze business needs and determine the optimal strategy.", "تحليل احتياجات العمل وتحديد الاستراتيجية المثلى.")},
					{Title: more("Design", "التصميم"), Body: more("Custom configuration of modules and features.", "إعداد مخصص للموديولات والميزات.")},
					{Title: more("Development", "التطوير"), Body: more("Module development and third-party integration.", "تطوير الموديولات والتكامل مع الأنظمة الخارجية.")},
					{Title: more("UAT", "اختبار القبول"), Body: more("User acceptance testing and performance validation.", "اختبار القبول من المستخدمين والتحقق من الأداء.")},
					{Title: more("Training", "التدريب"), Body: more("Comprehensive training and ongoing support.", "تدريب شامل ودعم مستمر.")},
				},
			},
			{
				Kind:  secCards,
				Label: more("Industries", "القطاعات"),
				Title: more("Industries we serve with Odoo", "القطاعات التي نخدمها بأودو"),
				Cards: []card{
					{Icon: "🏭", Title: more("Manufacturing", "التصنيع")},
					{Icon: "🏪", Title: more("Retail & POS", "التجزئة ونقاط البيع")},
					{Icon: "🏥", Title: more("Healthcare", "الرعاية الصحية")},
					{Icon: "🧵", Title: more("Textile", "المنسوجات")},
					{Icon: "🌾", Title: more("Agriculture", "الزراعة")},
					{Icon: "💰", Title: more("Finance", "المالية")},
				},
			},
			{
				Kind:  secPlans,
				Label: more("Packages", "الباقات"),
				Title: more("Flexible pricing for every business size", "أسعار مرنة لكل حجم أعمال"),
				Plans: []plan{
					{Name: more("Starter", "المبتدئ"), Desc: more("Basic Odoo setup, 3 modules, training, 3 months support.", "إعداد أودو أساسي، ٣ موديولات، تدريب، دعم ٣ شهور."), Features: []bi{more("3 modules", "٣ موديولات"), more("Training", "تدريب"), more("3 months support", "دعم ٣ شهور")}, CTA: more("Get Quote", "اطلب عرض سعر"), Href: "/contact-us"},
					{Name: more("Business", "الأعمال"), Desc: more("Full implementation, 8+ modules, customization, integration, 6 months support.", "تنفيذ كامل، ٨+ موديولات، تخصيص، تكامل، دعم ٦ شهور."), Featured: true, Badge: more("Popular", "الأكثر طلبًا"), Features: []bi{more("8+ modules", "٨+ موديولات"), more("Customization", "تخصيص"), more("Integration", "تكامل"), more("6 months support", "دعم ٦ شهور")}, CTA: more("Get Quote", "اطلب عرض سعر"), Href: "/contact-us"},
					{Name: more("Enterprise", "المؤسسات"), Desc: more("Unlimited modules, custom dev, multi-company, 12 months support.", "موديولات غير محدودة، تطوير مخصص، شركات متعددة، دعم ١٢ شهرًا."), Features: []bi{more("Unlimited modules", "موديولات غير محدودة"), more("Custom development", "تطوير مخصص"), more("Multi-company", "شركات متعددة"), more("12 months support", "دعم ١٢ شهرًا")}, CTA: more("Get Quote", "اطلب عرض سعر"), Href: "/contact-us"},
				},
			},
		},
		Final: closing(),
	}
}

// itPage returns /solutions/it-infrastructure.
func itPage() page {
	return page{
		Slug:    "/solutions/it-infrastructure",
		Nav:     "solutions",
		Title:   more("Network Infrastructure & CCTV Surveillance Installation in Aswan, Egypt", "تصميم وتركيب الشبكات وكاميرات المراقبة في أسوان، مصر"),
		Meta:    more("Professional network infrastructure, servers, firewall/VPN and CCTV surveillance installation across Aswan and Egypt — secure, fast and reliable.", "تصميم وتركيب البنية التحتية للشبكات والسيرفرات والجدار الناري وكاميرات المراقبة في أسوان ومصر — آمنة وسريعة وموثوقة."),
		Eyebrow: more("IT Infrastructure", "البنية التحتية"),
		H1:      more("Secure & reliable infrastructure.", "بنية تحتية آمنة وموثوقة."),
		Lead:    more("Slow network? Weak security? We design and implement professional network infrastructure, server systems and surveillance solutions for your business.", "شبكة بطيئة؟ أمان ضعيف؟ نصمم وننفّذ بنية تحتية احترافية وشبكات وسيرفرات وأنظمة مراقبة لأعمالك."),
		HeroCTAs: []link{
			{Label: more("Book a Site Visit", "احجز زيارة ميدانية"), Href: "/contact-us", Kind: "primary"},
			waLink(),
		},
		Sections: []section{
			{
				Kind: secFeatures,
				Features: []feature{
					{Icon: "🔌", Title: more("LAN / WAN", "شبكات LAN / WAN"), Body: more("Structured cabling, VLANs, routing and switching for offices and data centers.", "كابلات منظمة و VLANs وتوجيه وتبديل للمكاتب ومراكز البيانات.")},
					{Icon: "📡", Title: more("Wireless", "الشبكات اللاسلكية"), Body: more("Enterprise Wi-Fi — seamless roaming, load balancing, captive portals, central management.", "واي فاي للمؤسسات — تجوال سلس وتوزيع أحمال وبوابات وصول وإدارة مركزية.")},
					{Icon: "🔒", Title: more("Firewall & VPN", "الجدار الناري و VPN"), Body: more("Advanced firewall, site-to-site VPN, remote access and intrusion detection.", "جدار ناري متقدم و VPN بين المواقع ووصول عن بُعد وكشف التسلل.")},
					{Icon: "🖥️", Title: more("Servers", "السيرفرات"), Body: more("Linux/Windows servers, virtualization (Proxmox, VMware), Docker and HA clustering.", "سيرفرات لينكس وويندوز وافتراضية (Proxmox، VMware) وحاويات وتجميع عالي الإتاحة.")},
					{Icon: "📹", Title: more("CCTV & Security", "الكاميرات والأمن"), Body: more("IP cameras (2–8MP), DVR/NVR, remote monitoring, mobile apps, up to 128 channels.", "كاميرات IP (٢–٨ ميجابكسل) وأنظمة تسجيل ومراقبة عن بُعد وتطبيقات جوال حتى ١٢٨ قناة.")},
					{Icon: "📊", Title: more("Monitoring", "المراقبة"), Body: more("24/7 monitoring, uptime tracking, alerting, bandwidth analysis and optimization.", "مراقبة على مدار الساعة وتتبّع الجاهزية والتنبيهات وتحليل النطاق الترددي.")},
				},
			},
			{
				Kind:  secFeatures,
				Label: more("Why our infrastructure?", "لماذا بنيتنا التحتية؟"),
				Title: more("Built to stay up.", "مصممة لتبقى تعمل."),
				Features: []feature{
					{Icon: "🛡️", Title: more("Security", "الأمان"), Body: more("Enterprise-grade cybersecurity and data protection.", "أمن سيبراني بمستوى المؤسسات وحماية للبيانات.")},
					{Icon: "⚡", Title: more("Speed", "السرعة"), Body: more("Optimized for maximum throughput and low latency.", "محسّنة لأقصى إنتاجية وأقل زمن استجابة.")},
					{Icon: "🔄", Title: more("Stability", "الاستقرار"), Body: more("Redundant systems with a 99.9% uptime target.", "أنظمة احتياطية بهدف جاهزية ٩٩.٩٪.")},
					{Icon: "📈", Title: more("Scalable", "قابلة للتوسع"), Body: more("Grows with your business without major overhauls.", "تنمو مع أعمالك دون إعادة هيكلة كبيرة.")},
				},
			},
		},
		Final: finalBand(more("Need a network or CCTV quote?", "تحتاج عرضًا للشبكات أو كاميرات المراقبة؟"),
			more("Free site surveys across Aswan and Egypt.", "معاينات مجانية في أسوان ومصر."),
			link{Label: more("Request a Quote", "اطلب عرض سعر"), Href: "/contact-us", Kind: "primary"},
			waLink()),
	}
}

// webPage returns /solutions/web.
func webPage() page {
	return page{
		Slug:    "/solutions/web",
		Nav:     "solutions",
		Title:   more("Professional Web Development & Design Services in Egypt", "خدمات تطوير وتصميم المواقع الاحترافية في مصر"),
		Meta:    more("Modern, SEO-optimized websites, e-commerce stores and web applications built in Egypt — corporate sites, UI/UX, custom apps and maintenance.", "مواقع حديثة محسّنة لمحركات البحث ومتاجر إلكترونية وتطبيقات ويب في مصر — مواقع شركات وتصميم واجهات وتطبيقات مخصصة وصيانة."),
		Eyebrow: more("Web Development", "تطوير المواقع"),
		H1:      more("A professional website that delivers results.", "موقع احترافي يحقق نتائج."),
		Lead:    more("Your website isn't generating results? We build modern, SEO-optimized websites that convert visitors into customers.", "موقعك لا يحقق نتائج؟ نبني مواقع حديثة محسّنة لمحركات البحث تحوّل الزوار إلى عملاء."),
		HeroCTAs: []link{
			{Label: more("Get Your Website", "اطلب موقعك"), Href: "/contact-us", Kind: "primary"},
			{Label: more("View Portfolio", "شاهد أعمالنا"), Href: "/portfolio", Kind: "ghost"},
		},
		Sections: []section{
			{
				Kind: secFeatures,
				Features: []feature{
					{Icon: "🏢", Title: more("Corporate Sites", "مواقع الشركات"), Body: more("Professional company websites with modern design, responsive layout and clear brand messaging.", "مواقع شركات احترافية بتصميم حديث وتصميم متجاوب ورسالة علامة واضحة.")},
					{Icon: "🛒", Title: more("E-Commerce", "التجارة الإلكترونية"), Body: more("Online stores with payment integration, inventory sync, catalogs and order management.", "متاجر إلكترونية مع دمج الدفع ومزامنة المخزون والكتالوجات وإدارة الطلبات.")},
					{Icon: "🎨", Title: more("UI/UX Design", "تصميم الواجهات"), Body: more("Beautiful, intuitive interfaces designed for engagement, accessibility and conversion.", "واجهات جميلة وبديهية مصممة للتفاعل وسهولة الوصول والتحويل.")},
					{Icon: "📱", Title: more("Web Applications", "تطبيقات الويب"), Body: more("Custom web apps, dashboards, portals and SaaS platforms.", "تطبيقات ويب مخصصة ولوحات معلومات وبوابات ومنصات SaaS.")},
					{Icon: "🔍", Title: more("SEO", "تحسين محركات البحث"), Body: more("Search optimization, content strategy, meta tags, schema markup and performance.", "تحسين البحث واستراتيجية المحتوى والوسوم وبيانات المخطط والأداء.")},
					{Icon: "🔧", Title: more("Maintenance", "الصيانة"), Body: more("Ongoing updates, security patches, content management, hosting and monitoring.", "تحديثات مستمرة ورقع أمنية وإدارة محتوى واستضافة ومراقبة.")},
				},
			},
		},
		Final: finalBand(more("Ready for a website that works?", "جاهز لموقع يحقق نتائج؟"),
			more("Start your project today.", "ابدأ مشروعك اليوم."),
			link{Label: more("Start Your Project", "ابدأ مشروعك"), Href: "/contact-us", Kind: "primary"},
			waLink()),
	}
}
