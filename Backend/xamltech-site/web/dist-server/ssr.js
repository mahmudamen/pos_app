import { renderToString } from "react-dom/server";
import { useState } from "react";
import { Fragment, jsx, jsxs } from "react/jsx-runtime";
//#region \0rolldown/runtime.js
var __defProp = Object.defineProperty;
var __exportAll = (all, no_symbols) => {
	let target = {};
	for (var name in all) __defProp(target, name, {
		get: all[name],
		enumerable: true
	});
	if (!no_symbols) __defProp(target, Symbol.toStringTag, { value: "Module" });
	return target;
};
//#endregion
//#region src/content/data/pages/addons-store.json
var addons_store_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$22,
	default: () => addons_store_default,
	en: () => en$22
});
var en$22 = {
	"meta": {
		"title": "Premium Odoo 19 POS Addons — Published on the Odoo Apps Store — XamlTech",
		"description": "XamlTech's published Odoo addons: POS Security Framework, Customization Kit, Cutting Orders, Stock Real-Time, HDA Collection, Agriculture ERP and AI Copilots.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🛒 Odoo Apps Store",
				"title": "Premium Odoo addons",
				"lead": "Production-tested modules built from real client deployments and published on the official Odoo Apps Store. One-time purchase, lifetime updates.",
				"ctas": [{
					"label": "Browse on Odoo Apps",
					"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech",
					"style": "primary",
					"big": true,
					"external": true
				}, {
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "ghost",
					"big": true
				}],
				"stats": [
					{
						"v": "8",
						"l": "Published addons"
					},
					{
						"v": "19",
						"l": "Odoo 19 ready"
					},
					{
						"v": "100%",
						"l": "Lifetime updates"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "POS addons",
				"title": "Point-of-sale modules",
				"sub": "Built for real retail, restaurant and textile operations.",
				"cols": 3,
				"items": [
					{
						"icon": "🔐",
						"title": "POS Base — Security Framework",
						"text": "Comprehensive POS security: manager PIN validation, discount limit enforcement, multi-layer session security and backend validation.",
						"chips": ["Odoo 19 ready", "€99"]
					},
					{
						"icon": "🎨",
						"title": "POS Customization Kit",
						"text": "Flexible POS UI toolkit with theme builder, layout manager, component overrides and dark/light mode — full Arabic RTL.",
						"chips": ["Arabic RTL", "€89"]
					},
					{
						"icon": "✂️",
						"title": "POS Cutting Orders",
						"text": "Advanced cutting-order management for fabric and textile POS with real-time tracking, batch serial assignment and an OWL v2 UI.",
						"chips": ["Textile", "€129"]
					},
					{
						"icon": "📦",
						"title": "POS Stock Real-Time",
						"text": "Real-time stock synchronisation with reactive badge updates, batch loading, a navbar refresh button and live warehouse sync.",
						"chips": ["Live stock", "€129"]
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Vertical & AI addons",
				"title": "ERP and AI modules",
				"sub": "Vertical depth and machine-learning intelligence for Odoo.",
				"cols": 3,
				"items": [
					{
						"icon": "🏛️",
						"title": "HDA Collection — Revenue Management",
						"text": "Enterprise revenue collection for museums, tourism sites and cultural venues — multi-point operations, real-time reporting, collector POS and treasury deposits."
					},
					{
						"icon": "🌾",
						"title": "Advanced Agricultural Management",
						"text": "Complete farm-to-warehouse ERP for Odoo 19 — OWL dashboard with Chart.js, 4-level farm hierarchy, real-time costing and a 7-step production workflow."
					},
					{
						"icon": "🎯",
						"title": "AI Marketing Copilot",
						"text": "Smart AI-driven budget optimisation for Meta, Google Ads and TikTok — real-time campaign scoring and automated decision logging.",
						"chips": ["Odoo 19", "€69"]
					},
					{
						"icon": "💡",
						"title": "AI POS Copilot",
						"text": "Real-time AI upsell suggestions in Point of Sale, based on cart context, customer history and seasonal trends, with a bilingual Arabic-first UX.",
						"chips": ["Odoo 19", "€49"]
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Find the right addon for your business",
				"text": "Browse all published modules on the Odoo Apps Store, or ask us for a guided demo.",
				"actions": [{
					"label": "Browse on Odoo Apps",
					"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech",
					"style": "primary"
				}, {
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$22 = {
	"meta": {
		"title": "إضافات Odoo 19 المميزة — منشورة على متجر Odoo — زامل تك",
		"description": "إضافات زامل تك المنشورة على متجر Odoo: إطار الأمان، عدة التخصيص، أوامر القص، المخزون الفوري، إدارة الإيرادات، ERP الزراعة ومساعدو الذكاء الاصطناعي.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🛒 متجر Odoo",
				"title": "إضافات Odoo المميزة",
				"lead": "موديولات مُختبرة في الإنتاج بُنيت من تنفيذات عملاء حقيقيين ومنشورة على متجر Odoo الرسمي. دفعة واحدة وتحديثات مدى الحياة.",
				"ctas": [{
					"label": "تصفّح على Odoo",
					"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech",
					"style": "primary",
					"big": true,
					"external": true
				}, {
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "ghost",
					"big": true
				}],
				"stats": [
					{
						"v": "8",
						"l": "إضافات منشورة"
					},
					{
						"v": "19",
						"l": "جاهزة لـ Odoo 19"
					},
					{
						"v": "100%",
						"l": "تحديثات مدى الحياة"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "إضافات نقاط البيع",
				"title": "موديولات نقاط البيع",
				"sub": "مبنية لعمليات تجزئة ومطاعم ومنسوجات حقيقية.",
				"cols": 3,
				"items": [
					{
						"icon": "🔐",
						"title": "إطار الأمان الأساسي",
						"text": "أمان شامل لنقاط البيع: رقم PIN للمدير، فرض حدود الخصم، أمان جلسات متعدد الطبقات وتحقق خلفي.",
						"chips": ["جاهز Odoo 19", "€99"]
					},
					{
						"icon": "🎨",
						"title": "عدة تخصيص نقاط البيع",
						"text": "عدة مرنة لواجهة نقاط البيع مع بانٍ للثيمات ومدير تخطيط وتجاوز المكوّنات ونمط داكن/فاتح — بعربية RTL كاملة.",
						"chips": ["عربي RTL", "€89"]
					},
					{
						"icon": "✂️",
						"title": "أوامر القص",
						"text": "إدارة متقدمة لأوامر القص للأقمشة والمنسوجات مع تتبع فوري وتعيين أرقام تسلسلية وواجهة OWL v2.",
						"chips": ["منسوجات", "€129"]
					},
					{
						"icon": "📦",
						"title": "مخزون نقاط البيع الفوري",
						"text": "مزامنة فورية للمخزون مع شارات تفاعلية وتحميل دفعي وزر تحديث في الشريط ومزامنة حية للمخازن.",
						"chips": ["مخزون حي", "€129"]
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "إضافات قطاعية وذكاء اصطناعي",
				"title": "موديولات ERP والذكاء الاصطناعي",
				"sub": "عمق قطاعي وذكاء تعلّم الآلة لـ Odoo.",
				"cols": 3,
				"items": [
					{
						"icon": "🏛️",
						"title": "إدارة الإيرادات — HDA",
						"text": "نظام تحصيل إيرادات للمتاحف والمواقع السياحية والمنشآت الثقافية — عمليات متعددة النقاط وتقارير فورية وتكامل مع نقاط التحصيل وخزائن الإيداع."
					},
					{
						"icon": "🌾",
						"title": "الإدارة الزراعية المتقدمة",
						"text": "ERP كامل من المزرعة للمخزن لـ Odoo 19 — لوحة OWL مع Chart.js وهرمية مزرعة من 4 مستويات وتكاليف فورية وسير إنتاج من 7 خطوات."
					},
					{
						"icon": "🎯",
						"title": "مساعد التسويق الذكي",
						"text": "تحسين ذكي للميزانية لـ Meta وGoogle Ads وTikTok — تصنيف فوري للحملات وتوثيق آلي للقرارات.",
						"chips": ["Odoo 19", "€69"]
					},
					{
						"icon": "💡",
						"title": "مساعد نقاط البيع الذكي",
						"text": "اقتراحات بيع إضافي فورية داخل نقطة البيع حسب السلة وسجل العميل والاتجاهات الموسمية، بواجهة عربية أولاً.",
						"chips": ["Odoo 19", "€49"]
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "اعثر على الإضافة المناسبة لعملك",
				"text": "تصفّح كل الموديولات المنشورة على متجر Odoo، أو اطلب منا عرضاً موجهاً.",
				"actions": [{
					"label": "تصفّح على Odoo",
					"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech",
					"style": "primary"
				}, {
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "wa"
				}]
			}
		}
	]
};
var addons_store_default = {
	en: en$22,
	ar: ar$22
};
//#endregion
//#region src/content/data/pages/ai-solutions.json
var ai_solutions_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$21,
	default: () => ai_solutions_default,
	en: () => en$21
});
var en$21 = {
	"meta": {
		"title": "AI Marketing & POS Copilots | XamlTech Odoo Modules",
		"description": "AI Copilots for Odoo 19 — Marketing Copilot for budget optimisation and POS Copilot for real-time upsell suggestions. Bilingual, self-hosted and privacy-first.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "✨ AI-Powered Intelligence",
				"title": "AI Copilots for Odoo 19",
				"lead": "Transform your marketing and POS operations with AI-driven intelligence. Real-time suggestions, predictive analytics and automated optimisation — all within Odoo.",
				"points": [
					"Marketing Copilot — budget optimisation",
					"POS Copilot — real-time upsell",
					"Self-hosted, bilingual, GDPR-conscious"
				],
				"ctas": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Talk to us on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "🎯 Marketing Intelligence",
				"title": "AI Marketing Copilot",
				"sub": "Optimise your marketing spend with machine-learning insights, campaign ROI forecasts and predictive scoring.",
				"cols": 3,
				"items": [
					{
						"icon": "📊",
						"title": "Smart campaign analytics",
						"text": "Track performance metrics and ROI in real time."
					},
					{
						"icon": "🎯",
						"title": "Predictive scoring",
						"text": "ML-powered lead and opportunity scoring."
					},
					{
						"icon": "💸",
						"title": "Budget allocation",
						"text": "AI-recommended budget distribution across campaigns."
					},
					{
						"icon": "🎁",
						"title": "Incentive assignment",
						"text": "Automated incentive rules based on customer segments."
					},
					{
						"icon": "📈",
						"title": "Performance dashboard",
						"text": "Visual analytics with drill-down reports."
					},
					{
						"icon": "🧾",
						"title": "Decision logging",
						"text": "A full audit trail of AI recommendations."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "🛍️ POS Intelligence",
				"title": "AI POS Copilot",
				"sub": "Boost average transaction value with real-time AI upsell suggestions based on cart context, history and stock.",
				"cols": 3,
				"items": [
					{
						"icon": "💡",
						"title": "Smart upsell widget",
						"text": "Real-time suggestions in an OWL sidebar panel."
					},
					{
						"icon": "🌐",
						"title": "Multi-language",
						"text": "Full Arabic and English bilingual interface."
					},
					{
						"icon": "📦",
						"title": "Stock optimisation",
						"text": "Suggest slow-moving items to reduce inventory."
					},
					{
						"icon": "🧠",
						"title": "Customer history",
						"text": "ML-powered recommendations per customer."
					},
					{
						"icon": "📈",
						"title": "Conversion tracking",
						"text": "Log all suggestions and acceptance rates."
					},
					{
						"icon": "✍️",
						"title": "Custom prompt templates",
						"text": "Fine-tune AI behaviour per business."
					}
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"eyebrow": "Proven impact",
				"title": "How AI transforms your business",
				"items": [
					{
						"v": "+35%",
						"l": "Marketing ROI"
					},
					{
						"v": "+24%",
						"l": "Basket size"
					},
					{
						"v": "42%",
						"l": "Suggestion acceptance"
					},
					{
						"v": "+52%",
						"l": "Combined revenue growth"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"eyebrow": "FAQ",
				"title": "Frequently asked questions",
				"items": [
					{
						"q": "Is there a free trial?",
						"a": "Both AI Copilots are free to install from the Odoo Apps Store, with demo data included. No credit card required."
					},
					{
						"q": "Do they work with Arabic/RTL?",
						"a": "Yes. AI POS Copilot is fully bilingual (Arabic and English) with RTL support, and prompts are customisable per language."
					},
					{
						"q": "What Odoo versions are supported?",
						"a": "Both modules target Odoo 19, with Odoo 18 compatibility in progress."
					},
					{
						"q": "Can I customise the AI behaviour?",
						"a": "Absolutely. Both modules include prompt templates, scoring rules and decision logs, with full API access."
					},
					{
						"q": "What about data privacy?",
						"a": "All data stays within your Odoo instance — no external API calls for AI processing — and is GDPR-conscious."
					},
					{
						"q": "How is this better than other POS apps?",
						"a": "Real-time processing, a bilingual UI, a clean architecture codebase, support and production deployments."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to transform your business?",
				"text": "Install from the Odoo Apps Store today — no setup required, demo data included.",
				"actions": [{
					"label": "Visit the addons store",
					"href": "/addons-store?lang=en",
					"style": "primary"
				}, {
					"label": "Email us",
					"href": "mailto:info@xamltech.com",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$21 = {
	"meta": {
		"title": "مساعدو الذكاء الاصطناعي للتسويق ونقاط البيع | موديولات Odoo من زامل تك",
		"description": "مساعدو ذكاء اصطناعي لـ Odoo 19 — مساعد التسويق لتحسين الميزانيات ومساعد نقاط البيع لاقتراحات البيع الإضافي الفورية. ثنائي اللغة ومستضاف ذاتياً ويحترم الخصوصية.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "✨ ذكاء اصطناعي",
				"title": "مساعدو ذكاء اصطناعي لـ Odoo 19",
				"lead": "حوّل عمليات التسويق ونقاط البيع بذكاء مدعوم بالذكاء الاصطناعي. اقتراحات فورية وتحليلات تنبؤية وتحسين آلي — كل ذلك داخل Odoo.",
				"points": [
					"مساعد التسويق — تحسين الميزانية",
					"مساعد نقاط البيع — بيع إضافي فوري",
					"مستضاف ذاتياً وثنائي اللغة ويراعي الخصوصية"
				],
				"ctas": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary",
					"big": true
				}, {
					"label": "تواصل على واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "🎯 ذكاء التسويق",
				"title": "مساعد التسويق الذكي",
				"sub": "حسّن إنفاقك التسويقي برؤى تعلّم الآلة وتوقعات عائد الحملات والتصنيف التنبؤي.",
				"cols": 3,
				"items": [
					{
						"icon": "📊",
						"title": "تحليلات حملات ذكية",
						"text": "تتبع مؤشرات الأداء والعائد لحظياً."
					},
					{
						"icon": "🎯",
						"title": "تصنيف تنبؤي",
						"text": "تصنيف العملاء والفرص بتعلّم الآلة."
					},
					{
						"icon": "💸",
						"title": "توزيع الميزانية",
						"text": "توزيع موصى به من الذكاء الاصطناعي عبر الحملات."
					},
					{
						"icon": "🎁",
						"title": "إسناد الحوافز",
						"text": "قواعد حوافز آلية حسب شرائح العملاء."
					},
					{
						"icon": "📈",
						"title": "لوحة الأداء",
						"text": "تحليلات مرئية مع تقارير تفصيلية."
					},
					{
						"icon": "🧾",
						"title": "توثيق القرارات",
						"text": "سجل تدقيق كامل لتوصيات الذكاء الاصطناعي."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "🛍️ ذكاء نقاط البيع",
				"title": "مساعد نقاط البيع الذكي",
				"sub": "ارفع متوسط قيمة الفاتورة باقتراحات بيع إضافي فورية حسب السلة والسجل والمخزون.",
				"cols": 3,
				"items": [
					{
						"icon": "💡",
						"title": "أداة البيع الإضافي",
						"text": "اقتراحات فورية في لوحة OWL الجانبية."
					},
					{
						"icon": "🌐",
						"title": "متعدد اللغات",
						"text": "واجهة ثنائية اللغة عربي وإنجليزي."
					},
					{
						"icon": "📦",
						"title": "تحسين المخزون",
						"text": "اقتراح الأصناف بطيئة الحركة لتصريف المخزون."
					},
					{
						"icon": "🧠",
						"title": "سجل العميل",
						"text": "توصيات مدرّبة لكل عميل."
					},
					{
						"icon": "📈",
						"title": "تتبع التحويل",
						"text": "تسجيل كل الاقتراحات ونسب القبول."
					},
					{
						"icon": "✍️",
						"title": "قوالب أوامر مخصصة",
						"text": "ضبط سلوك الذكاء الاصطناعي لكل نشاط."
					}
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"eyebrow": "أثر مثبت",
				"title": "كيف يغيّر الذكاء الاصطناعي أعمالك",
				"items": [
					{
						"v": "+35%",
						"l": "عائد التسويق"
					},
					{
						"v": "+24%",
						"l": "حجم السلة"
					},
					{
						"v": "42%",
						"l": "نسبة قبول الاقتراحات"
					},
					{
						"v": "+52%",
						"l": "نمو الإيرادات المشترك"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"eyebrow": "الأسئلة الشائعة",
				"title": "أسئلة شائعة",
				"items": [
					{
						"q": "هل هناك تجربة مجانية؟",
						"a": "كلا المساعدين مجانيان للتركيب من متجر Odoo، مع بيانات تجريبية مدمجة. لا حاجة لبطاقة ائتمان."
					},
					{
						"q": "هل يعملان مع العربية RTL؟",
						"a": "نعم. مساعد نقاط البيع ثنائي اللغة (عربي وإنجليزي) بدعم RTL، والقوالب قابلة للتخصيص لكل لغة."
					},
					{
						"q": "ما إصدارات Odoo المدعومة؟",
						"a": "كلا الموديولين يستهدفان Odoo 19، ونعمل على توافق Odoo 18."
					},
					{
						"q": "هل أستطيع تخصيص سلوك الذكاء الاصطناعي؟",
						"a": "بالتأكيد. الموديولان يتضمنان قوالب وقواعد تصنيف وسجلات قرارات، مع وصول كامل لـ API."
					},
					{
						"q": "ماذا عن خصوصية البيانات؟",
						"a": "كل البيانات تبقى داخل نسخة Odoo الخاصة بك — بدون استدعاءات خارجية لمعالجة الذكاء الاصطناعي — وتراعي الخصوصية."
					},
					{
						"q": "كيف هذا أفضل من تطبيقات أخرى؟",
						"a": "معالجة فورية وواجهة ثنائية اللغة وكود بنية نظيفة ودعم وتنفيذات إنتاجية."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "مستعد لتحويل أعمالك؟",
				"text": "ثبّت من متجر Odoo اليوم — بلا إعداد، مع بيانات تجريبية.",
				"actions": [{
					"label": "زيارة متجر الإضافات",
					"href": "/addons-store",
					"style": "primary"
				}, {
					"label": "راسلنا",
					"href": "mailto:info@xamltech.com",
					"style": "wa"
				}]
			}
		}
	]
};
var ai_solutions_default = {
	en: en$21,
	ar: ar$21
};
//#endregion
//#region src/content/data/pages/apk-download.json
var apk_download_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$20,
	default: () => apk_download_default,
	en: () => en$20
});
var en$20 = {
	"meta": {
		"title": "Download Android Apps — XamlTech",
		"description": "Request a secure download link for XamlTech Android apps such as Network Copilot. Links are reviewed and valid for 30 days.",
		"og_image": "https://xamltech.com/static/img/networktool-dashboard.png",
		"index": true
	},
	"hero": {
		"eyebrow": "📲 Android downloads",
		"title": "Request the app, download securely",
		"lead": "Choose the app you want and fill in the form. After we review your request, you receive a secure download link valid for 30 days.",
		"points": [
			"Secure, expiring links (30 days)",
			"Reviewed requests",
			"No public mirror with stale builds"
		],
		"ctas": [{
			"label": "Chat on WhatsApp",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "Request a download link",
	"formSub": "After review we'll email you a secure, expiring download link.",
	"submit": "Request download",
	"privacyLabel": "Privacy Policy",
	"products": ["Network Copilot (Android 10+)"],
	"cta": {
		"title": "Need it for your team?",
		"text": "We can provide builds for multiple devices and deployment guidance.",
		"actions": [{
			"label": "Contact us",
			"href": "/contact-us?lang=en",
			"style": "primary"
		}, {
			"label": "WhatsApp us",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var ar$20 = {
	"meta": {
		"title": "تحميل تطبيقات أندرويد — زامل تك",
		"description": "اطلب رابط تحميل آمناً لتطبيقات زامل تك لأندرويد مثل Network Copilot. تتم مراجعة الطلبات وتكون الروابط صالحة لمدة 30 يوماً.",
		"og_image": "https://xamltech.com/static/img/networktool-dashboard.png",
		"index": true
	},
	"hero": {
		"eyebrow": "📲 تحميل أندرويد",
		"title": "اطلب النسخة، وتحمّل بأمان",
		"lead": "اختر التطبيق الذي تريده واملأ النموذج. بعد مراجعة طلبك، يصلك رابط تحميل آمن صالح لمدة 30 يوماً.",
		"points": [
			"روابط آمنة منتهية الصلاحية (30 يوماً)",
			"مراجعة الطلبات",
			"بلا نسخ عامة ببناءات قديمة"
		],
		"ctas": [{
			"label": "محادثة واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "اطلب رابط التحميل",
	"formSub": "بعد المراجعة سنرسل لك رابط تحميل آمناً منتهي الصلاحية بالبريد.",
	"submit": "اطلب التحميل",
	"privacyLabel": "سياسة الخصوصية",
	"products": ["Network Copilot (أندرويد 10+)"],
	"cta": {
		"title": "تحتاجه لفريقك؟",
		"text": "يمكننا توفير بناءات لعدة أجهزة وإرشادات النشر.",
		"actions": [{
			"label": "تواصل معنا",
			"href": "/contact-us",
			"style": "primary"
		}, {
			"label": "واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var apk_download_default = {
	en: en$20,
	ar: ar$20
};
//#endregion
//#region src/content/data/pages/contact-us.json
var contact_us_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$19,
	default: () => contact_us_default,
	en: () => en$19
});
var en$19 = {
	"meta": {
		"title": "Contact XamlTech — Odoo Development & IT Services in Aswan, Egypt",
		"description": "Have a project or need support? Reach XamlTech by form, phone or WhatsApp — Odoo development and IT services from Aswan, Egypt.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"hero": {
		"eyebrow": "✉️ Get in Touch",
		"title": "Contact us",
		"lead": "Have a project? Need support? Reach out via the form or WhatsApp — we usually reply within one business day.",
		"points": [
			"Aswan, Egypt · Sun–Thu, 9:00–18:00",
			"info@xamltech.com",
			"+20 120 141 0140 · WhatsApp +20 100 798 2468"
		],
		"ctas": [{
			"label": "Chat on WhatsApp",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "Send a message",
	"formSub": "Tell us about your project and we'll get back to you shortly.",
	"submit": "Send message",
	"privacyLabel": "Privacy Policy",
	"cta": {
		"title": "Prefer a live walkthrough?",
		"text": "Request a demo and we'll show you the POS addons and ERP in action.",
		"actions": [{
			"label": "Request a demo",
			"href": "/request-demo?lang=en",
			"style": "primary"
		}]
	}
};
var ar$19 = {
	"meta": {
		"title": "تواصل مع زامل تك — تطوير Odoo وخدمات تقنية المعلومات في أسوان، مصر",
		"description": "لديك مشروع أو تحتاج دعماً؟ تواصل مع زامل تك عبر النموذج أو الهاتف أو واتساب — تطوير Odoo وخدمات تقنية المعلومات من أسوان، مصر.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"hero": {
		"eyebrow": "✉️ ابقَ على تواصل",
		"title": "تواصل معنا",
		"lead": "لديك مشروع؟ تحتاج دعماً؟ تواصل عبر النموذج أو واتساب — نرد عادة خلال يوم عمل واحد.",
		"points": [
			"أسوان، مصر · الأحد–الخميس، 9:00–18:00",
			"info@xamltech.com",
			"‎+20 120 141 0140 · واتساب ‎+20 100 798 2468"
		],
		"ctas": [{
			"label": "محادثة واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "أرسل رسالة",
	"formSub": "أخبرنا عن مشروعك وسنتواصل معك قريباً.",
	"submit": "أرسل الرسالة",
	"privacyLabel": "سياسة الخصوصية",
	"cta": {
		"title": "تفضّل عرضاً مباشراً؟",
		"text": "اطلب تجربة وسنعرض عليك إضافات نقاط البيع ونظام ERP أثناء العمل.",
		"actions": [{
			"label": "اطلب تجربة",
			"href": "/request-demo",
			"style": "primary"
		}]
	}
};
var contact_us_default = {
	en: en$19,
	ar: ar$19
};
//#endregion
//#region src/content/data/pages/erp-agriculture-egypt.json
var erp_agriculture_egypt_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$18,
	default: () => erp_agriculture_egypt_default,
	en: () => en$18
});
var en$18 = {
	"meta": {
		"title": "Odoo ERP for Agriculture & Farm Management in Upper Egypt — XamlTech Aswan",
		"description": "Custom Odoo systems for farms, greenhouses and agricultural operations — crop tracking, work orders, harvest management and cost analytics, integrated with accounting and payroll.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🌾 Agriculture ERP Specialist",
				"title": "Odoo ERP for agriculture & farm management in Upper Egypt",
				"lead": "We build custom Odoo systems for farms, greenhouses and agricultural operations. Crop tracking, work orders, harvest management and cost analytics — fully integrated with accounting and payroll.",
				"points": [
					"From planting to harvest — full traceability",
					"Farm → Sector → Unit → House hierarchy",
					"Irrigation, fertilising, spraying, harvest work orders",
					"Cost per crop, location and season",
					"Accounting + payroll + inventory integrated"
				],
				"ctas": [{
					"label": "Request a free consultation",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Talk to us on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Why your farm needs ERP",
				"title": "From paper chaos to data",
				"cols": 3,
				"items": [
					{
						"icon": "📄",
						"title": "Without ERP",
						"text": "Lost paper records, unknown cost per crop, untracked labour and harvest by guesswork."
					},
					{
						"icon": "🌱",
						"title": "With Odoo ERP",
						"text": "Everything digital, cost per square metre, automatic worker and payroll tracking and documented harvests."
					},
					{
						"icon": "📈",
						"title": "The result",
						"text": "Data-driven decisions, 20–30% cost reduction, higher productivity and bank-ready reporting."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Features",
				"title": "Farm management module features",
				"sub": "Vertical logic built for real farming operations.",
				"cols": 3,
				"items": [
					{
						"icon": "🗺️",
						"title": "Hierarchical locations",
						"text": "4-level hierarchy: Farm → Sector → Unit → House. Each level maps to analytic accounts, with workers and costs per location."
					},
					{
						"icon": "🌾",
						"title": "Crop management",
						"text": "Track varieties, planting dates, growth stages and expected yields — with crop rotation planning across seasons."
					},
					{
						"icon": "📋",
						"title": "Agricultural work orders",
						"text": "Orders for planting, irrigation, fertilising, spraying and harvesting, tracking labour hours, materials and equipment."
					},
					{
						"icon": "🚜",
						"title": "Harvest & inventory",
						"text": "Stock wizards auto-create warehouse receipts on harvest; lot tracking from field to storage to sale."
					},
					{
						"icon": "💰",
						"title": "Cost analytics",
						"text": "Real P&L per farm, crop and season, tracking seeds, fertiliser, water, labour and equipment."
					},
					{
						"icon": "👷",
						"title": "Worker payroll",
						"text": "Assign workers to locations, track attendance via work orders and auto-calculate daily, task-based or seasonal wages."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Crops",
				"title": "Crops we support in Upper Egypt",
				"sub": "Specialised configurations for the region's major crops.",
				"cols": 4,
				"items": [
					{
						"icon": "🎋",
						"title": "Sugarcane"
					},
					{
						"icon": "🌴",
						"title": "Date palms"
					},
					{
						"icon": "🍅",
						"title": "Vegetables"
					},
					{
						"icon": "🌾",
						"title": "Wheat & grains"
					},
					{
						"icon": "🍊",
						"title": "Citrus fruits"
					},
					{
						"icon": "🫘",
						"title": "Legumes"
					},
					{
						"icon": "🧵",
						"title": "Cotton"
					},
					{
						"icon": "🌿",
						"title": "Herbs & medicinal"
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"alt": true,
				"eyebrow": "🏆 Real deployment — Qatf Company",
				"title": "Agriculture ERP in production",
				"lead": "Built and deployed for real farming operations — not a demo.",
				"points": [
					"Problem: multiple farm locations with different crops and paper records",
					"Solution: farm_management for Odoo 17 — hierarchy, crop lifecycle, work orders, harvest stock wizards",
					"Result: complete per-crop profitability visibility",
					"Worker payroll automated, saving 2 days per month"
				],
				"media": {
					"src": "/static/img/pos-light.png",
					"alt": "Agriculture ERP deployment",
					"window": true,
					"caption": "erp.xamltech.com",
					"live": "Live"
				}
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Why XamlTech",
				"title": "Why XamlTech for agriculture ERP?",
				"cols": 3,
				"items": [
					{
						"icon": "📍",
						"title": "Based in Upper Egypt",
						"text": "We're in Aswan — we understand sugarcane seasons, Nile irrigation and date palm cultivation, with on-site visits and local support."
					},
					{
						"icon": "🌾",
						"title": "Agriculture specialist",
						"text": "We built a complete farm management module from scratch, production-tested with real farming operations."
					},
					{
						"icon": "⚙️",
						"title": "Full-stack Odoo",
						"text": "8+ years of Odoo development (v12–v19) with published modules, handling Python, OWL, PostgreSQL and infrastructure."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to digitise your farm?",
				"text": "Free consultation and site visits in Upper Egypt. Contact us now.",
				"actions": [{
					"label": "Free consultation",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$18 = {
	"meta": {
		"title": "نظام Odoo ERP للزراعة وإدارة المزارع في صعيد مصر — زامل تك أسوان",
		"description": "أنظمة Odoo مخصصة للمزارع والصوبات والعمليات الزراعية — تتبع المحاصيل وأوامر العمل وإدارة الحصاد وتحليل التكاليف، متكاملة مع المحاسبة والرواتب.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🌾 متخصصون في ERP الزراعة",
				"title": "نظام Odoo ERP للزراعة وإدارة المزارع في صعيد مصر",
				"lead": "نبني أنظمة Odoo مخصصة للمزارع والصوبات والعمليات الزراعية. تتبع المحاصيل وأوامر العمل وإدارة الحصاد وتحليل التكاليف — متكاملة مع المحاسبة والرواتب.",
				"points": [
					"من الزراعة للحصاد — تتبع كامل",
					"مزرعة → قطاع → وحدة → بيت",
					"ري وتسميد ورش وحصاد — أوامر عمل",
					"تكلفة لكل محصول وموقع وموسم",
					"محاسبة + رواتب + مخزون متكامل"
				],
				"ctas": [{
					"label": "اطلب استشارة مجانية",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "تواصل على واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "ليه المزرعة محتاجة نظام ERP؟",
				"title": "من فوضى الورق إلى البيانات",
				"cols": 3,
				"items": [
					{
						"icon": "📄",
						"title": "بدون ERP",
						"text": "سجلات ورقية ضايعة، مش عارف تكلفة كل محصول، مفيش تتبع للعمال والحصاد بالتخمين."
					},
					{
						"icon": "🌱",
						"title": "مع Odoo ERP",
						"text": "كل حاجة رقمية، تعرف تكلفة كل متر مربع، تتبع تلقائي للعمال والرواتب وحصاد موثق."
					},
					{
						"icon": "📈",
						"title": "النتيجة",
						"text": "قرارات مبنية على بيانات، تخفيض تكاليف 20–30%، إنتاجية أعلى وتقارير جاهزة للبنوك."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "المميزات",
				"title": "مميزات موديول إدارة المزارع",
				"sub": "منطق قطاعي مبني لعمليات زراعية حقيقية.",
				"cols": 3,
				"items": [
					{
						"icon": "🗺️",
						"title": "مواقع هرمية",
						"text": "هرمية من 4 مستويات: مزرعة → قطاع → وحدة → بيت. كل مستوى مرتبط بحساب تحليلي، مع عمال وتكاليف لكل موقع."
					},
					{
						"icon": "🌾",
						"title": "إدارة المحاصيل",
						"text": "تتبع الأصناف وتواريخ الزراعة ومراحل النمو والإنتاجية المتوقعة، مع تخطيط الدورة الزراعية عبر المواسم."
					},
					{
						"icon": "📋",
						"title": "أوامر العمل الزراعية",
						"text": "أوامر للزراعة والري والتسميد والرش والحصاد، مع تتبع ساعات العمل والمواد والمعدات."
					},
					{
						"icon": "🚜",
						"title": "الحصاد والمخزون",
						"text": "معالجات المخزون تُنشئ إيصالات استلام تلقائياً عند الحصاد، مع تتبع الدفعات من الحقل للمخزن للبيع."
					},
					{
						"icon": "💰",
						"title": "تحليل التكاليف",
						"text": "أرباح وخسائر حقيقية لكل مزرعة ومحصول وموسم، مع تتبع البذور والأسمدة والمياه والعمالة والمعدات."
					},
					{
						"icon": "👷",
						"title": "رواتب العمال",
						"text": "توزيع العمال على المواقع، تتبع الحضور عبر أوامر العمل وحساب الأجور اليومية أو بالقطعة أو الموسمية تلقائياً."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "المحاصيل",
				"title": "المحاصيل التي ندعمها في صعيد مصر",
				"sub": "تهيئات متخصصة لأهم محاصيل المنطقة.",
				"cols": 4,
				"items": [
					{
						"icon": "🎋",
						"title": "قصب السكر"
					},
					{
						"icon": "🌴",
						"title": "نخيل التمور"
					},
					{
						"icon": "🍅",
						"title": "خضروات"
					},
					{
						"icon": "🌾",
						"title": "قمح وحبوب"
					},
					{
						"icon": "🍊",
						"title": "حمضيات"
					},
					{
						"icon": "🫘",
						"title": "بقوليات"
					},
					{
						"icon": "🧵",
						"title": "قطن"
					},
					{
						"icon": "🌿",
						"title": "أعشاب ونباتات طبية"
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"alt": true,
				"eyebrow": "🏆 تنفيذ حقيقي — شركة قطف",
				"title": "نظام ERP الزراعة في الإنتاج",
				"lead": "بُني ونُشر لعمليات زراعية حقيقية — وليس عرضاً تجريبياً.",
				"points": [
					"المشكلة: مواقع مزرعة متعددة بمحاصيل مختلفة وسجلات ورقية",
					"الحل: موديول farm_management لإصدار Odoo 17 — الهرمية ودورة المحصول وأوامر العمل ومعالجات الحصاد",
					"النتيجة: رؤية كاملة لربحية كل محصول",
					"أتمتة رواتب العمال بتوفير يومين شهرياً"
				],
				"media": {
					"src": "/static/img/pos-rtl.png",
					"alt": "تنفيذ ERP زراعي",
					"window": true,
					"caption": "erp.xamltech.com",
					"live": "مباشر"
				}
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "لماذا زامل تك",
				"title": "ليه زامل تك للزراعة؟",
				"cols": 3,
				"items": [
					{
						"icon": "📍",
						"title": "من صعيد مصر",
						"text": "مقرنا أسوان — نفهم زراعة الصعيد: مواسم القصب وري النيل وزراعة النخيل، مع زيارات ميدانية ودعم محلي."
					},
					{
						"icon": "🌾",
						"title": "متخصصون في الزراعة",
						"text": "بنينا موديول إدارة مزارع متكاملاً من الصفر، مُختبراً في عمليات زراعية حقيقية."
					},
					{
						"icon": "⚙️",
						"title": "Odoo متكامل",
						"text": "أكثر من 8 سنوات تطوير Odoo (v12–v19) مع موديولات منشورة، نتولى Python وOWL وPostgreSQL والبنية التحتية."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "جاهز تحوّل مزرعتك لنظام رقمي؟",
				"text": "استشارة مجانية وزيارة موقع في صعيد مصر. تواصل معنا الآن.",
				"actions": [{
					"label": "استشارة مجانية",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var erp_agriculture_egypt_default = {
	en: en$18,
	ar: ar$18
};
//#endregion
//#region src/content/data/pages/home.json
var home_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$17,
	default: () => home_default,
	en: () => en$17
});
var en$17 = {
	"meta": {
		"title": "XamlTech — Odoo ERP, POS & IT Infrastructure in Egypt",
		"description": "XamlTech implements, customises and supports Odoo ERP, point-of-sale and IT infrastructure across Egypt — with custom modules published on the official Odoo Apps Store.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"hero": {
		"eyebrow": "Odoo 16–19 Partner · Aswan, Egypt",
		"title": "ERP and POS that run like they were built for you.",
		"accent": "Because they were.",
		"lead": "We implement, customise and support Odoo ERP, point-of-sale and the infrastructure behind them — publishing our own modules on the official Odoo Apps Store.",
		"ctas": [{
			"label": "Request a demo",
			"href": "/request-demo",
			"style": "primary",
			"big": true
		}, {
			"label": "Explore solutions",
			"href": "/solutions/erp",
			"style": "ghost",
			"big": true
		}],
		"media": {
			"src": "/static/img/pos-light.png",
			"alt": "XamlTech Odoo POS — live register",
			"window": true,
			"caption": "pos.xamltech.com",
			"live": "Live"
		},
		"stats": [
			{
				"v": "40+",
				"l": "Organizations served"
			},
			{
				"v": "180+",
				"l": "Open-source repositories"
			},
			{
				"v": "8",
				"l": "Published Odoo addons"
			},
			{
				"v": "16–19",
				"l": "Odoo versions mastered"
			}
		]
	},
	"chips": { "items": [
		"ERP & POS",
		"Accounting & HR",
		"Networks & Servers",
		"CCTV & Access",
		"Web & Apps",
		"Cybersecurity",
		"Cloud & Hosting",
		"AI Copilots"
	] },
	"why": {
		"eyebrow": "Why XamlTech",
		"title": "Why teams across Egypt choose us",
		"sub": "One partner for the whole stack — from the ERP that runs your business to the network and cameras that protect it.",
		"cols": 3,
		"items": [
			{
				"icon": "🏅",
				"title": "Real Odoo partner experience",
				"text": "Odoo 16 through 19 implementation, customisation and migration — with modules published and sold on the official Odoo Apps Store."
			},
			{
				"icon": "📍",
				"title": "Local Upper-Egypt support",
				"text": "Based in Aswan with on-site support, Arabic-first workflows and response times that remote-only vendors cannot match."
			},
			{
				"icon": "🧩",
				"title": "End-to-end delivery",
				"text": "ERP, POS, networking, CCTV, hosting, web and AI under one roof — no finger-pointing between vendors."
			},
			{
				"icon": "🛡️",
				"title": "Security-first engineering",
				"text": "Vulnerability assessment, encryption, access control, backups and disaster recovery built into every deployment."
			},
			{
				"icon": "⚙️",
				"title": "Custom modules that fit",
				"text": "Fabric cutting, agriculture, healthcare, ZATCA e-invoicing — we build the vertical logic your business actually needs."
			},
			{
				"icon": "🤖",
				"title": "AI where it pays off",
				"text": "AI POS and marketing copilots that optimise budgets and upsell in real time, with graceful degradation when offline."
			}
		]
	},
	"services": {
		"alt": true,
		"eyebrow": "Our services",
		"title": "Complete technology services",
		"sub": "Comprehensive technology solutions to accelerate your business growth.",
		"cols": 3,
		"items": [
			{
				"icon": "🖥️",
				"title": "IT Services",
				"text": "Full IT infrastructure — provisioning, setup and maintenance."
			},
			{
				"icon": "💻",
				"title": "Software Development",
				"text": "Custom software and app development — Python, JS, Android, iOS."
			},
			{
				"icon": "🌐",
				"title": "Web Development",
				"text": "Professional websites and web applications, built and managed."
			},
			{
				"icon": "🔄",
				"title": "Digital Transformation",
				"text": "Modernise operations and automate workflows end to end."
			},
			{
				"icon": "🔗",
				"title": "Network & Server Management",
				"text": "Design, deploy and monitor networks and server systems."
			},
			{
				"icon": "📊",
				"title": "ERP Systems (Odoo)",
				"text": "Odoo implementation and customisation — POS, accounting, inventory, HR."
			},
			{
				"icon": "🔐",
				"title": "Information Security",
				"text": "Vulnerability assessment, encryption and access control."
			},
			{
				"icon": "📹",
				"title": "CCTV & Surveillance",
				"text": "IP cameras, DVR/NVR systems, remote monitoring and security."
			},
			{
				"icon": "🧭",
				"title": "Technical Consulting",
				"text": "Planning, implementation and maintenance of your technology stack."
			},
			{
				"icon": "🛒",
				"title": "E-Commerce Solutions",
				"text": "Online stores with payment gateways, shipping and order management."
			},
			{
				"icon": "☁️",
				"title": "Cloud & Hosting",
				"text": "VPS, cloud infrastructure, email hosting, backup and disaster recovery."
			},
			{
				"icon": "📦",
				"title": "Hardware Supply",
				"text": "Servers, storage, network gear, laptops and workstations."
			},
			{
				"icon": "📣",
				"title": "Digital Marketing & Ads",
				"text": "Strategy, social media management, Google, Facebook and Instagram ads."
			}
		]
	},
	"odoo": {
		"eyebrow": "Odoo & ERP expertise",
		"title": "Odoo specialists — not generalists",
		"sub": "Deep specialisation in building and customising Odoo for real business operations.",
		"cols": 3,
		"items": [
			{
				"icon": "🏗️",
				"title": "Odoo system engineering",
				"text": "Specialised development across accounting, point of sale, inventory and HR."
			},
			{
				"icon": "🧱",
				"title": "Module customisation",
				"text": "Custom modules and integration with external systems and government portals."
			},
			{
				"icon": "🥇",
				"title": "Published Odoo partner",
				"text": "Modules on the official Odoo store — years of Odoo 16–19 experience."
			}
		]
	},
	"method": {
		"alt": true,
		"eyebrow": "Our methodology",
		"title": "A delivery process you can track",
		"sub": "Structured phases keep scope clear and go-live predictable.",
		"items": [
			{
				"title": "Requirements analysis",
				"text": "We map your processes, data and constraints into a clear specification."
			},
			{
				"title": "Design & configuration",
				"text": "Solution architecture, module selection and environment setup."
			},
			{
				"title": "Development & integration",
				"text": "Custom modules, integrations and data migration, iterated with you."
			},
			{
				"title": "Testing & acceptance",
				"text": "Functional and load testing against your real scenarios before go-live."
			},
			{
				"title": "Training & support",
				"text": "Hands-on training, documentation and ongoing technical support."
			}
		]
	},
	"demo": {
		"eyebrow": "⚡ Live demo",
		"title": "See it in action — try our POS addons live",
		"sub": "Watch the system work in real time, then request free access.",
		"cols": 2,
		"items": [
			{
				"icon": "✂️",
				"title": "Fabric cutting from POS",
				"text": "Create cutting orders directly from the point of sale."
			},
			{
				"icon": "📦",
				"title": "Live inventory updates",
				"text": "Stock badges refresh across every session in real time."
			},
			{
				"icon": "🔑",
				"title": "Manager PIN & discount limits",
				"text": "Role-based access, discount caps and manager overrides."
			},
			{
				"icon": "🎨",
				"title": "Themes, dark & light mode",
				"text": "A flexible Arabic-first POS UI with theme switching."
			}
		]
	},
	"addons": {
		"alt": true,
		"eyebrow": "Our work",
		"title": "Published Odoo addons",
		"sub": "Production-tested modules available on the Odoo Apps Store.",
		"cols": 3,
		"cta": {
			"label": "Visit the addons store",
			"href": "/addons-store"
		},
		"items": [
			{
				"icon": "✂️",
				"title": "POS Cutting Orders",
				"text": "Fabric cutting-order management from the point of sale with real-time tracking and OWL v2 UI."
			},
			{
				"icon": "📦",
				"title": "POS Stock Real-Time",
				"text": "Real-time inventory synchronisation across all sessions with live badges and warehouse sync."
			},
			{
				"icon": "🔐",
				"title": "POS Security Framework",
				"text": "Manager PIN validation, discount limits and multi-layer session security."
			}
		]
	},
	"cta": {
		"title": "Ready for your digital transformation?",
		"text": "Whether you need a custom ERP system or complete infrastructure, our team is ready to help.",
		"actions": [{
			"label": "Request a demo",
			"href": "/request-demo",
			"style": "primary"
		}, {
			"label": "Talk to us on WhatsApp",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var ar$17 = {
	"meta": {
		"title": "زامل تك — أنظمة Odoo ERP ونقاط البيع والبنية التحتية في مصر",
		"description": "نُنفّذ ونخصّص وندعم أنظمة Odoo ERP ونقاط البيع والبنية التحتية التقنية في مصر — مع موديولات مخصصة منشورة على متجر Odoo الرسمي.",
		"og_image": "https://xamltech.com/static/img/pos-rtl.png",
		"index": true
	},
	"hero": {
		"eyebrow": "شريك Odoo 16–19 · أسوان، مصر",
		"title": "أنظمة ERP ونقاط بيع تعمل وكأنها صُنعت من أجلك.",
		"accent": "لأنها كذلك فعلاً.",
		"lead": "نُنفّذ ونخصّص وندعم أنظمة Odoo ERP ونقاط البيع والبنية التحتية القائمة عليها — وننشر موديولاتنا الخاصة على متجر Odoo الرسمي.",
		"ctas": [{
			"label": "اطلب تجربة",
			"href": "/request-demo",
			"style": "primary",
			"big": true
		}, {
			"label": "استكشف الحلول",
			"href": "/solutions/erp",
			"style": "ghost",
			"big": true
		}],
		"media": {
			"src": "/static/img/pos-rtl.png",
			"alt": "نقطة بيع زامل تك — كاشير مباشر",
			"window": true,
			"caption": "pos.xamltech.com",
			"live": "مباشر"
		},
		"stats": [
			{
				"v": "+40",
				"l": "مؤسسة نخدمها"
			},
			{
				"v": "+180",
				"l": "مستودع مفتوح المصدر"
			},
			{
				"v": "8",
				"l": "إضافات Odoo منشورة"
			},
			{
				"v": "16–19",
				"l": "إصدارات Odoo المدعومة"
			}
		]
	},
	"chips": { "items": [
		"ERP ونقاط البيع",
		"المحاسبة والموارد البشرية",
		"الشبكات والسيرفرات",
		"كاميرات المراقبة",
		"الويب والتطبيقات",
		"الأمن السيبراني",
		"السحابة والاستضافة",
		"مساعدو الذكاء الاصطناعي"
	] },
	"why": {
		"eyebrow": "لماذا زامل تك",
		"title": "لماذا تختارنا المؤسسات في مصر",
		"sub": "شريك واحد لكل التقنيات — من نظام ERP الذي يدير أعمالك إلى الشبكة والكاميرات التي تحميها.",
		"cols": 3,
		"items": [
			{
				"icon": "🏅",
				"title": "خبرة حقيقية كشريك Odoo",
				"text": "تنفيذ وتخصيص وترحيل Odoo من الإصدار 16 إلى 19، مع موديولات منشورة ومُباعة على متجر Odoo الرسمي."
			},
			{
				"icon": "📍",
				"title": "دعم محلي في صعيد مصر",
				"text": "مقرنا أسوان مع دعم ميداني وسير عمل عربي أولاً واستجابة لا تستطيع الشركات عن بُعد تقديمها."
			},
			{
				"icon": "🧩",
				"title": "تسليم متكامل",
				"text": "ERP ونقاط بيع وشبكات وكاميرات واستضافة وويب وذكاء اصطناعي تحت سقف واحد — دون تهرّب من المسؤولية."
			},
			{
				"icon": "🛡️",
				"title": "هندسة تبدأ من الأمان",
				"text": "تقييم الثغرات والتشفير والتحكم في الوصول والنسخ الاحتياطي والتعافي من الكوارث في كل نشر."
			},
			{
				"icon": "⚙️",
				"title": "موديولات مخصصة تناسبك",
				"text": "أوامر القص للأقمشة، الزراعة، الرعاية الصحية، فاتورة ZATCA الإلكترونية — نبني ما يحتاجه عملك فعلاً."
			},
			{
				"icon": "🤖",
				"title": "ذكاء اصطناعي حيث يعود بالنفع",
				"text": "مساعدو نقاط البيع والتسويق الذين يحسّنون الميزانيات ويقترحون البيع الإضافي فورياً، مع تدرّج آمن عند انقطاع الشبكة."
			}
		]
	},
	"services": {
		"alt": true,
		"eyebrow": "خدماتنا",
		"title": "خدمات تقنية شاملة",
		"sub": "حلول تقنية متكاملة لتسريع نمو أعمالك.",
		"cols": 3,
		"items": [
			{
				"icon": "🖥️",
				"title": "خدمات تكنولوجيا المعلومات",
				"text": "بنية تحتية متكاملة — التجهيز والإعداد والصيانة."
			},
			{
				"icon": "💻",
				"title": "تصميم وتطوير البرمجيات",
				"text": "تطوير برمجيات وتطبيقات مخصصة — Python، JS، أندرويد، iOS."
			},
			{
				"icon": "🌐",
				"title": "تطوير وبرمجة المواقع",
				"text": "تصميم وتطوير وإدارة مواقع وتطبيقات الويب الاحترافية."
			},
			{
				"icon": "🔄",
				"title": "حلول التحول الرقمي",
				"text": "تحديث العمليات وأتمتة سير العمل من البداية للنهاية."
			},
			{
				"icon": "🔗",
				"title": "إدارة الشبكات والسيرفرات",
				"text": "تصميم ونشر ومراقبة الشبكات وأنظمة السيرفرات."
			},
			{
				"icon": "📊",
				"title": "أنظمة ERP (Odoo)",
				"text": "تنفيذ وتخصيص Odoo — نقاط بيع، محاسبة، مخزون، موارد بشرية."
			},
			{
				"icon": "🔐",
				"title": "أمن المعلومات",
				"text": "تقييم الثغرات والتشفير والتحكم في الوصول."
			},
			{
				"icon": "📹",
				"title": "كاميرات المراقبة",
				"text": "كاميرات IP، أنظمة DVR/NVR، مراقبة عن بُعد وحلول أمنية."
			},
			{
				"icon": "🧭",
				"title": "الاستشارات التقنية والدعم",
				"text": "تخطيط وتنفيذ وصيانة بنيتك التقنية."
			},
			{
				"icon": "🛒",
				"title": "حلول التجارة الإلكترونية",
				"text": "متاجر إلكترونية مع بوابات الدفع والشحن وإدارة الطلبات."
			},
			{
				"icon": "☁️",
				"title": "الحلول السحابية والاستضافة",
				"text": "VPS وبنية سحابية واستضافة بريد ونسخ احتياطي وتعافي من الكوارث."
			},
			{
				"icon": "📦",
				"title": "توريد الأجهزة والمعدات",
				"text": "سيرفرات وأنظمة تخزين وأجهزة شبكات ولابتوب ومحطات عمل."
			},
			{
				"icon": "📣",
				"title": "التسويق الرقمي والإعلانات",
				"text": "استراتيجية وإدارة سوشيال ميديا وإعلانات Google وFacebook وInstagram."
			}
		]
	},
	"odoo": {
		"eyebrow": "خبرة Odoo و ERP",
		"title": "متخصصون في Odoo — لا عموميون",
		"sub": "تخصص عميق في بناء وتخصيص Odoo لعمليات الأعمال الحقيقية.",
		"cols": 3,
		"items": [
			{
				"icon": "🏗️",
				"title": "تقنية أنظمة Odoo",
				"text": "تطوير متخصص في المحاسبة ونقاط البيع والمخزون والموارد البشرية."
			},
			{
				"icon": "🧱",
				"title": "تخصيص الموديولات",
				"text": "تطوير موديولات مخصصة والتكامل مع الأنظمة الخارجية والبوابات الحكومية."
			},
			{
				"icon": "🥇",
				"title": "شريك Odoo منشور",
				"text": "موديولات على متجر Odoo الرسمي — سنوات من خبرة Odoo 16–19."
			}
		]
	},
	"method": {
		"alt": true,
		"eyebrow": "منهجية التنفيذ",
		"title": "عملية تسليم يمكنك تتبعها",
		"sub": "مراحل منظمة تُبقي النطاق واضحاً والإطلاق متوقعاً.",
		"items": [
			{
				"title": "تحليل المتطلبات",
				"text": "نحوّل عملياتك وبياناتك وقيودك إلى مواصفات واضحة."
			},
			{
				"title": "التصميم والإعداد",
				"text": "هندسة الحل واختيار الموديولات وتجهيز البيئة."
			},
			{
				"title": "التطوير والتكامل",
				"text": "موديولات مخصصة وتكاملات وترحيل بيانات، بتكرار مستمر معك."
			},
			{
				"title": "الاختبار والقبول",
				"text": "اختبار وظيفي وتحمل على سيناريوهاتك الحقيقية قبل الإطلاق."
			},
			{
				"title": "التدريب والدعم",
				"text": "تدريب عملي وتوثيق ودعم فني مستمر."
			}
		]
	},
	"demo": {
		"eyebrow": "⚡ تجربة مباشرة",
		"title": "شاهد النظام أثناء العمل — جرّب إضافات نقاط البيع مباشرة",
		"sub": "تابع النظام لحظياً ثم اطلب وصولاً مجانياً.",
		"cols": 2,
		"items": [
			{
				"icon": "✂️",
				"title": "القص من نقطة البيع",
				"text": "إنشاء أوامر القص مباشرة من نقطة البيع."
			},
			{
				"icon": "📦",
				"title": "تحديثات مخزون فورية",
				"text": "تحديث شارات المخزون في كل الجلسات لحظياً."
			},
			{
				"icon": "🔑",
				"title": "رقم PIN للمدير وحدود الخصم",
				"text": "صلاحيات حسب الدور وحدود للخصم وتجاوز المدير."
			},
			{
				"icon": "🎨",
				"title": "ثيمات والنمط الداكن/الفاتح",
				"text": "واجهة نقاط بيع عربية مرنة مع تبديل الثيم."
			}
		]
	},
	"addons": {
		"alt": true,
		"eyebrow": "أعمالنا",
		"title": "إضافات Odoo المنشورة",
		"sub": "موديولات مُختبرة في الإنتاج ومتاحة على متجر Odoo.",
		"cols": 3,
		"cta": {
			"label": "زيارة متجر الإضافات",
			"href": "/addons-store"
		},
		"items": [
			{
				"icon": "✂️",
				"title": "أوامر القص في نقاط البيع",
				"text": "إدارة أوامر قص الأقمشة من نقطة البيع مع تتبع فوري وواجهة OWL v2."
			},
			{
				"icon": "📦",
				"title": "مخزون نقاط البيع الفوري",
				"text": "مزامنة فورية للمخزون عبر كل الجلسات مع شارات حية ومزامنة المخازن."
			},
			{
				"icon": "🔐",
				"title": "إطار أمان نقاط البيع",
				"text": "تحقق برقم PIN للمدير وحدود للخصم وأمان متعدد الطبقات للجلسات."
			}
		]
	},
	"cta": {
		"title": "هل أنت مستعد للتحول الرقمي؟",
		"text": "سواء كنت تحتاج نظام ERP مخصصاً أو بنية تحتية متكاملة، فريقنا جاهز لمساعدتك.",
		"actions": [{
			"label": "اطلب تجربة",
			"href": "/request-demo",
			"style": "primary"
		}, {
			"label": "تواصل معنا على واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var home_default = {
	en: en$17,
	ar: ar$17
};
//#endregion
//#region src/content/data/pages/insights-ai-marketing-copilot.json
var insights_ai_marketing_copilot_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$16,
	default: () => insights_ai_marketing_copilot_default,
	en: () => en$16
});
var en$16 = {
	"meta": {
		"title": "AI Marketing Copilot: Stop Wasting Marketing Budget | XamlTech",
		"description": "How AI Marketing Copilot delivers 35% ROI improvements, 18% cost reduction and automated budget allocation — the pipeline, the case study and the rollout.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📊 AI Marketing Intelligence",
				"title": "AI Marketing Copilot:",
				"accent": "Stop Wasting Marketing Budget",
				"lead": "Real marketing teams using AI Marketing Copilot increased ROI by 35%, reduced costs by 18% and automated half their marketing decisions. Here's exactly how.",
				"ctas": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "See AI solutions",
					"href": "/ai-solutions?lang=en",
					"style": "ghost"
				}],
				"stats": [
					{
						"v": "+35%",
						"l": "ROI improvement"
					},
					{
						"v": "−18%",
						"l": "Cost reduction"
					},
					{
						"v": "50%",
						"l": "Decisions automated"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "What you'll learn",
				"title": "Inside this guide",
				"items": [
					{
						"icon": "📈",
						"title": "The ROI math",
						"text": "Why marketing teams are seeing 35% ROI improvements — and where the waste was hiding."
					},
					{
						"icon": "🤖",
						"title": "How AI works",
						"text": "The machine-learning pipeline behind smart budgeting, from ingestion to recommendations."
					},
					{
						"icon": "💡",
						"title": "Real case studies",
						"text": "How agencies and brands deployed this system and what changed in three months."
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"eyebrow": "The problem",
				"title": "Marketing decisions are guesses",
				"lead": "Most marketing teams allocate budgets based on last year's numbers, gut feeling, or what worked for competitors. They run campaigns in silos with no channel attribution.",
				"points": [
					"40–50% of marketing budget is wasted on underperforming channels",
					"Budget is set once and rarely rebalanced",
					"Channel crossover effects stay invisible"
				],
				"media": {
					"src": "/static/img/ai-copilot.png",
					"alt": "AI Marketing Copilot dashboard"
				}
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "The solution",
				"title": "AI-powered budget allocation",
				"lead": "AI Marketing Copilot replaces guesswork with data-driven decisions. The system continuously analyzes campaign performance, customer behaviour, market trends, historical data and cross-channel effects.",
				"points": [
					"Campaign performance — CTR, conversion rate, CPL, CAC in real time",
					"Customer behaviour — purchase patterns, LTV, repeat-purchase probability",
					"Market trends — seasonality, competitor activity, channel saturation",
					"Historical data — budget-to-revenue correlation and channel elasticity",
					"Cross-channel effects — how spending on email affects organic conversion"
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"alt": true,
				"eyebrow": "How it works",
				"title": "The AI pipeline",
				"sub": "Five stages from raw data to an explainable recommendation.",
				"items": [
					{
						"title": "1. Data Ingestion",
						"text": "Connects directly to your Odoo database and marketing platforms (Google Ads, Meta, email providers) via API."
					},
					{
						"title": "2. Feature Engineering",
						"text": "Computes CTR, CAC, LTV, ROAS and budget elasticity for every channel and segment."
					},
					{
						"title": "3. Predictive Modeling",
						"text": "Forecasts next month's conversion rate, segment LTV, optimal budget split and lead-conversion likelihood."
					},
					{
						"title": "4. Recommendation Engine",
						"text": "Turns predictions into actionable steps — increase, pause, shift budget or target commissions."
					},
					{
						"title": "5. Audit Trail & Learning",
						"text": "Logs why each recommendation was made, its confidence score and the actual outcome to improve over time."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Key features",
				"title": "What you get",
				"items": [
					{
						"icon": "📈",
						"title": "Campaign Dashboard",
						"text": "Real-time ROAS, CAC, LTV, conversion rate and budget-vs-actual, with visual cross-campaign comparisons."
					},
					{
						"icon": "🎯",
						"title": "Predictive Scoring",
						"text": "Every lead gets a conversion-probability score (0–100). Focus sales efforts on hot leads."
					},
					{
						"icon": "💰",
						"title": "Budget Allocation Engine",
						"text": "AI recommends the exact budget split (e.g. 65% Google, 35% Meta) for maximum ROI."
					},
					{
						"icon": "🏆",
						"title": "Incentive Rules",
						"text": "Automatically assign commissions to high-LTV customer segments and sales channels."
					},
					{
						"icon": "📊",
						"title": "Drill-Down Reports",
						"text": "Compare performance by channel, customer segment, time period and geography."
					},
					{
						"icon": "🔐",
						"title": "Full Audit Trail",
						"text": "Every recommendation is logged for compliance, accountability and continuous learning."
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "FAQ",
				"title": "Getting started",
				"items": [
					{
						"q": "How do I install it?",
						"a": "Search \"AI Marketing Copilot\" on the Odoo Apps Store and install in two clicks. Demo data (30 sample campaigns, customers and financials) is included."
					},
					{
						"q": "Does it need configuration?",
						"a": "No. Install, load the demo data and the system generates 5–10 actionable recommendations from your data immediately."
					},
					{
						"q": "Can I connect my ad platforms?",
						"a": "Yes, optionally. Integrate Google Ads, Meta and email APIs for live data; until then the copilot works from your Odoo data."
					},
					{
						"q": "Does it stay inside my Odoo?",
						"a": "Yes. All data stays within your Odoo instance — no cloud APIs and no third-party data storage."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to optimize your marketing budget?",
				"text": "Install AI Marketing Copilot on the Odoo Apps Store — completely free, demo data included, no setup required.",
				"actions": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp support",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$16 = {
	"meta": {
		"title": "مساعد التسويق الذكي: أوقف إهدار ميزانية التسويق | زامل تك",
		"description": "كيف يحقق مساعد التسويق الذكي تحسناً 35% في العائد و18% خفضاً في التكلفة وتوزيعاً آلياً للميزانية — خط الأنابيب ودراسة الحالة وخطوات الإطلاق.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📊 ذكاء تسويقي",
				"title": "مساعد التسويق الذكي:",
				"accent": "أوقف إهدار ميزانية التسويق",
				"lead": "فِرق تسويق حقيقية تستخدم مساعد التسويق الذكي رفعت العائد 35% وخفضت التكاليف 18% وأتمتت نصف قراراتها التسويقية. إليك كيف بالتفصيل.",
				"ctas": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary",
					"big": true
				}, {
					"label": "حلول الذكاء",
					"href": "/ai-solutions",
					"style": "ghost"
				}],
				"stats": [
					{
						"v": "+35%",
						"l": "تحسّن العائد"
					},
					{
						"v": "−18%",
						"l": "خفض التكلفة"
					},
					{
						"v": "50%",
						"l": "قرارات آلية"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "ما ستتعلمه",
				"title": "داخل هذا الدليل",
				"items": [
					{
						"icon": "📈",
						"title": "حسابات العائد",
						"text": "لماذا يحقق الفريق تحسناً 35% في العائد — وأين كان الهدر مخفياً."
					},
					{
						"icon": "🤖",
						"title": "كيف يعمل الذكاء",
						"text": "خط الأنابيب خلف التوزيع الذكي، من الإدخال حتى التوصيات."
					},
					{
						"icon": "💡",
						"title": "دراسات حالة حقيقية",
						"text": "كيف نشرت الوكالات والعلامات هذا النظام وما تغيّر خلال ثلاثة أشهر."
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"eyebrow": "المشكلة",
				"title": "قرارات التسويق تخمينات",
				"lead": "معظم الفرق توزّع الميزانية بناءً على أرقام العام الماضي أو الحدس أو ما نجح للمنافسين. وتدير الحملات منفصلة بلا إسناد للقنوات.",
				"points": [
					"40–50% من ميزانية التسويق تُهدر على قنوات ضعيفة الأداء",
					"الميزانية تُحدد مرة ونادراً ما يُعاد توزيعها",
					"تأثيرات القنوات المتراكمة تبقى غير مرئية"
				],
				"media": {
					"src": "/static/img/ai-copilot.png",
					"alt": "لوحة مساعد التسويق الذكي"
				}
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "الحل",
				"title": "توزيع ميزانية مدعوم بالذكاء",
				"lead": "مساعد التسويق الذكي يستبدل التخمين بقرارات قائمة على البيانات. يحلل النظام باستمرار أداء الحملات وسلوك العملاء واتجاهات السوق والبيانات التاريخية وتأثيرات القنوات المتراكمة.",
				"points": [
					"أداء الحملات — CTR ومعدل التحويل وتكلفة العميل المحتمل وCAC فورياً",
					"سلوك العملاء — أنماط الشراء والقيمة العمرية واحتمال الشراء المتكرر",
					"اتجاهات السوق — الموسمية ونشاط المنافسين وتشبع القنوات",
					"البيانات التاريخية — ارتباط الميزانية بالإيراد ومرونة القنوات",
					"تأثيرات القنوات — كيف يؤثر الإنفاق على البريد في التحويل العضوي"
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"alt": true,
				"eyebrow": "كيف يعمل",
				"title": "خط أنابيب الذكاء",
				"sub": "خمس مراحل من البيانات الخام إلى توصية قابلة للشرح.",
				"items": [
					{
						"title": "1. إدخال البيانات",
						"text": "يتصل مباشرة بقاعدة بيانات Odoo ومنصات التسويق (Google Ads وMeta ومزودي البريد) عبر API."
					},
					{
						"title": "2. هندسة الخصائص",
						"text": "يحسب CTR وCAC وLTV وROAS ومرونة الميزانية لكل قناة وشريحة."
					},
					{
						"title": "3. النمذجة التنبؤية",
						"text": "يتوقع معدل التحويل الشهر القادم والقيمة العمرية للشريحة والتوزيع الأمثل واحتمال تحويل العميل المحتمل."
					},
					{
						"title": "4. محرك التوصيات",
						"text": "يحوّل التوقعات إلى خطوات قابلة للتنفيذ — زيادة أو إيقاف أو نقل ميزانية أو استهداف عمولات."
					},
					{
						"title": "5. سجل التدقيق والتعلم",
						"text": "يسجل سبب كل توصية ودرجة ثقتها والنتيجة الفعلية لتحسينها لاحقاً."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "المزايا الرئيسية",
				"title": "ما تحصل عليه",
				"items": [
					{
						"icon": "📈",
						"title": "لوحة الحملات",
						"text": "ROAS وCAC وLTV ومعدل التحويل والميزانية مقابل الفعلي فورياً، مع مقارنات مرئية بين الحملات."
					},
					{
						"icon": "🎯",
						"title": "تقييم تنبؤي",
						"text": "كل عميل محتمل يحصل على درجة احتمال تحويل (0–100). ركّز جهود البيع على العملاء الساخنين."
					},
					{
						"icon": "💰",
						"title": "محرك توزيع الميزانية",
						"text": "يوصي الذكاء بالتوزيع الدقيق (مثلاً 65% Google و35% Meta) لأقصى عائد."
					},
					{
						"icon": "🏆",
						"title": "قواعد الحوافز",
						"text": "يعيّن العمولات آلياً لشرائح العملاء والقنوات ذات القيمة العمرية الأعلى."
					},
					{
						"icon": "📊",
						"title": "تقارير تفصيلية",
						"text": "قارن الأداء حسب القناة والشريحة والفترة الجغرافية."
					},
					{
						"icon": "🔐",
						"title": "سجل تدقيق كامل",
						"text": "كل توصية مسجلة للامتثال والمساءلة والتعلم المستمر."
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "الأسئلة الشائعة",
				"title": "البدء",
				"items": [
					{
						"q": "كيف أثبّته؟",
						"a": "ابحث عن «AI Marketing Copilot» في متجر Odoo وثبّته بنقرتين. وتتضمن البيانات التجريبية 30 حملة وعميلاً وبيانات مالية."
					},
					{
						"q": "هل يحتاج إعداداً؟",
						"a": "لا. ثبّت وحمّل البيانات التجريبية، فيولّد النظام 5–10 توصيات قابلة للتنفيذ من بياناتك فوراً."
					},
					{
						"q": "هل أربط منصات إعلاناتي؟",
						"a": "نعم، اختيارياً. اربط Google Ads وMeta وواجهات البريد لبيانات حية؛ وحتى ذلك يعمل المساعد من بيانات Odoo."
					},
					{
						"q": "هل تبقى البيانات داخل Odoo؟",
						"a": "نعم. تبقى كل البيانات داخل نسخة Odoo لديك — بلا واجهات سحابية أو تخزين لدى طرف ثالث."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "جاهز لتحسين ميزانية تسويقك؟",
				"text": "ثبّت مساعد التسويق الذكي من متجر Odoo — مجاناً تماماً، ببيانات تجريبية، وبلا إعداد.",
				"actions": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary"
				}, {
					"label": "دعم واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var insights_ai_marketing_copilot_default = {
	en: en$16,
	ar: ar$16
};
//#endregion
//#region src/content/data/pages/insights-ai-pos-copilot.json
var insights_ai_pos_copilot_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$15,
	default: () => insights_ai_pos_copilot_default,
	en: () => en$15
});
var en$15 = {
	"meta": {
		"title": "AI POS Copilot: Sell More at Every Transaction | XamlTech",
		"description": "Real-time AI upsell suggestions at the register in under 100ms. The retail case study, the pipeline and how AI POS Copilot raised basket size by 24%.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🤖 Real-Time AI Recommendations",
				"title": "AI POS Copilot:",
				"accent": "Sell More at Every Transaction",
				"lead": "Every second counts at the register. AI POS Copilot analyzes the customer's cart in real time and suggests complementary products before the sale is done.",
				"ctas": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "See AI solutions",
					"href": "/ai-solutions?lang=en",
					"style": "ghost"
				}],
				"stats": [
					{
						"v": "+24%",
						"l": "Basket size increase"
					},
					{
						"v": "42%",
						"l": "Suggestion acceptance"
					},
					{
						"v": "4.9★",
						"l": "Odoo Store rating"
					},
					{
						"v": "<100ms",
						"l": "Response time"
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"eyebrow": "The problem",
				"title": "You're leaving money on the register",
				"lead": "A customer walks up with 3 items. The cashier scans, collects payment and they leave. Thirty seconds later you realize they were about to buy one more product — but nothing suggested it.",
				"points": [
					"The average retail transaction could be 15–30% larger with the right complementary suggestion",
					"Recommendation engines drive 35% of Amazon's revenue",
					"Real-time suggestions at the point of purchase are the highest-intent moment you have"
				],
				"media": {
					"src": "/static/img/ai-copilot.png",
					"alt": "AI POS Copilot in the register"
				}
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "The solution",
				"title": "AI that knows your customer's intent",
				"lead": "AI POS Copilot sits beside every cashier like an expert sales assistant. It watches the cart, analyzes the customer's purchase history and surfaces the right product at the right moment — in under 100ms.",
				"points": [
					"One-tap add, no disruption",
					"Bilingual Arabic/English with full RTL",
					"Only suggests in-stock items",
					"Learns from accept/reject feedback"
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "Case study: a 15-store retail chain",
				"items": [
					{
						"v": "145 → 180 EGP",
						"l": "Average transaction"
					},
					{
						"v": "5% → 85%",
						"l": "Suggestion rate"
					},
					{
						"v": "10% → 42%",
						"l": "Acceptance rate"
					},
					{
						"v": "3.15M EGP",
						"l": "Extra monthly revenue"
					}
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"eyebrow": "How it works",
				"title": "Real-time processing pipeline",
				"sub": "Six stages run the moment a cashier scans an item.",
				"items": [
					{
						"title": "1. Cart Analysis",
						"text": "Analyze items in the current cart, total value and customer category."
					},
					{
						"title": "2. Customer History",
						"text": "Check past purchases — what this customer usually buys together."
					},
					{
						"title": "3. Product Matching",
						"text": "Find complementary products using category and tagging rules."
					},
					{
						"title": "4. Inventory Check",
						"text": "Only suggest in-stock items to prevent frustration."
					},
					{
						"title": "5. Ranking by Relevance",
						"text": "Score candidates by margin, stock level and customer preference."
					},
					{
						"title": "6. Display",
						"text": "Show the top 1–3 suggestions in Arabic/English in the sidebar."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Key features",
				"title": "Built for the register",
				"items": [
					{
						"icon": "🌐",
						"title": "Bilingual (Arabic + English)",
						"text": "Full RTL support. Prompts, product names and UI adapt to language settings."
					},
					{
						"icon": "⚡",
						"title": "Ultra-Fast Response",
						"text": "Suggestions appear in under 100ms — no lag, no slowdown in the checkout flow."
					},
					{
						"icon": "📊",
						"title": "Conversion Tracking",
						"text": "Log every suggestion: shown, accepted, rejected. Measure the impact on revenue."
					},
					{
						"icon": "🎯",
						"title": "Zero Setup Needed",
						"text": "Install and it works immediately. Demo data included, no ML training required."
					},
					{
						"icon": "⚙️",
						"title": "Customizable Rules",
						"text": "Configure which product combinations make sense for your store."
					},
					{
						"icon": "🚀",
						"title": "All POS Screens",
						"text": "Works on touchscreen, tablet and traditional POS terminals on modern Odoo 19 setups."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Use cases by industry",
				"title": "Relevant suggestions in every aisle",
				"items": [
					{
						"icon": "🛒",
						"title": "Retail & Fashion",
						"text": "Shoes + socks, shirt + belt, dress + accessories."
					},
					{
						"icon": "☕",
						"title": "Cafe & Food Service",
						"text": "Coffee + pastry, sandwich + drink, dessert after a meal."
					},
					{
						"icon": "💊",
						"title": "Pharmacy",
						"text": "Medicine + pain relief, vitamins + supplements."
					},
					{
						"icon": "📱",
						"title": "Electronics",
						"text": "Phone + case, laptop + charger, screen + protector."
					},
					{
						"icon": "💄",
						"title": "Beauty & Cosmetics",
						"text": "Foundation + concealer, lipstick + primer, skincare combos."
					},
					{
						"icon": "🏠",
						"title": "Home & Hardware",
						"text": "Paint + brush, nails + hammer, cables + adapters."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to boost your sales?",
				"text": "Install AI POS Copilot today. Free, zero setup, and a 42% acceptance rate when suggestions are relevant.",
				"actions": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp support",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$15 = {
	"meta": {
		"title": "مساعد نقاط البيع الذكي: بِع أكثر في كل عملية | زامل تك",
		"description": "اقتراحات بيع إضافي ذكية وفورية عند الكاشير في أقل من 100ms. دراسة سلسلة تجزئة وخط الأنابيب وكيف رفع مساعد نقاط البيع حجم السلة 24%.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🤖 توصيات ذكاء فورية",
				"title": "مساعد نقاط البيع الذكي:",
				"accent": "بِع أكثر في كل عملية",
				"lead": "كل ثانية تُحسب عند الكاشير. يحلل مساعد نقاط البيع الذكي سلة العميل فورياً ويقترح منتجات مكمّلة قبل إتمام البيع.",
				"ctas": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary",
					"big": true
				}, {
					"label": "حلول الذكاء",
					"href": "/ai-solutions",
					"style": "ghost"
				}],
				"stats": [
					{
						"v": "+24%",
						"l": "زيادة حجم السلة"
					},
					{
						"v": "42%",
						"l": "قبول الاقتراحات"
					},
					{
						"v": "★4.9",
						"l": "تقييم متجر Odoo"
					},
					{
						"v": "<100ms",
						"l": "زمن الاستجابة"
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"eyebrow": "المشكلة",
				"title": "تترك أموالاً على الكاشير",
				"lead": "يأتي عميل بثلاثة منتجات. يمسحها الكاشير ويحصّل الدفع ويرحل. وبعد ثلاثين ثانية تدرك أنه كان سيشتري منتجاً آخر — لكن شيئاً لم يقترحه.",
				"points": [
					"متوسط عملية التجزئة قد يزيد 15–30% بالاقتراح المكمّل الصحيح",
					"محركات التوصية تولّد 35% من إيراد Amazon",
					"الاقتراح الفوري عند نقطة الشراء أعلى لحظة نية تملكها"
				],
				"media": {
					"src": "/static/img/ai-copilot.png",
					"alt": "مساعد نقاط البيع الذكي عند الكاشير"
				}
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "الحل",
				"title": "ذكاء يعرف نية عميلك",
				"lead": "يجلس مساعد نقاط البيع الذكي بجانب كل كاشير كمساعد بيع خبير. يراقب السلة ويحلل سجل الشراء ويرفع المنتج الصحيح في اللحظة الصحيحة — في أقل من 100ms.",
				"points": [
					"إضافة بلمسة، بلا مقاطعة",
					"ثنائي اللغة عربي/إنجليزي بدعم RTL كامل",
					"يقترح المنتجات المتوفرة فقط",
					"يتعلم من قبول أو رفض الاقتراح"
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "دراسة حالة: سلسلة تجزئة بـ 15 فرعاً",
				"items": [
					{
						"v": "145 → 180 ج.م",
						"l": "متوسط العملية"
					},
					{
						"v": "5% → 85%",
						"l": "معدل الاقتراح"
					},
					{
						"v": "10% → 42%",
						"l": "معدل القبول"
					},
					{
						"v": "3.15M ج.م",
						"l": "إيراد شهري إضافي"
					}
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"eyebrow": "كيف يعمل",
				"title": "خط المعالجة الفورية",
				"sub": "ست مراحل تعمل لحظة مسح الكاشير لأي منتج.",
				"items": [
					{
						"title": "1. تحليل السلة",
						"text": "تحليل عناصر السلة الحالية وقيمتها وفئة العميل."
					},
					{
						"title": "2. سجل العميل",
						"text": "فحص المشتريات السابقة — ما يشتريه العميل عادة معاً."
					},
					{
						"title": "3. مطابقة المنتج",
						"text": "إيجاد المنتجات المكمّلة باستخدام قواعد الفئة والوسوم."
					},
					{
						"title": "4. فحص المخزون",
						"text": "اقتراح المنتجات المتوفرة فقط لتجنب الإحباط."
					},
					{
						"title": "5. الترتيب حسب الصلة",
						"text": "تقييم المرشحين بالهامش ومستوى المخزون وتفضيل العميل."
					},
					{
						"title": "6. العرض",
						"text": "إظهار أفضل 1–3 اقتراحات بالعربية/الإنجليزية في الشريط الجانبي."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "المزايا الرئيسية",
				"title": "مصمم للكاشير",
				"items": [
					{
						"icon": "🌐",
						"title": "ثنائي اللغة عربي/إنجليزي",
						"text": "دعم RTL كامل. النصوص وأسماء المنتجات والواجهة تتكيف مع اللغة."
					},
					{
						"icon": "⚡",
						"title": "استجابة فائقة السرعة",
						"text": "تظهر الاقتراحات في أقل من 100ms — بلا تأخير في تدفق الدفع."
					},
					{
						"icon": "📊",
						"title": "تتبع التحويل",
						"text": "تسجيل كل اقتراح: ظهر، قُبل، رُفض. وقياس الأثر على الإيراد."
					},
					{
						"icon": "🎯",
						"title": "بلا إعداد",
						"text": "ثبّت ويعمل فوراً. ببيانات تجريبية، وبلا تدريب نموذج."
					},
					{
						"icon": "⚙️",
						"title": "قواعد قابلة للتخصيص",
						"text": "اضبط أي توليفات منتجات تناسب متجرك."
					},
					{
						"icon": "🚀",
						"title": "كل شاشات نقاط البيع",
						"text": "يعمل على شاشات اللمس والأجهزة اللوحية والطرفيات التقليدية في إعداد Odoo 19 الحديثة."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "حالات الاستخدام حسب القطاع",
				"title": "اقتراحات مناسبة في كل ممر",
				"items": [
					{
						"icon": "🛒",
						"title": "التجزئة والأزياء",
						"text": "حذاء + جوارب، قميص + حزام، فستان + إكسسوار."
					},
					{
						"icon": "☕",
						"title": "المقاهي والأغذية",
						"text": "قهوة + معجنات، ساندويتش + مشروب، حلوى بعد الوجبة."
					},
					{
						"icon": "💊",
						"title": "الصيدلية",
						"text": "دواء + مسكن، فيتامينات + مكمّلات."
					},
					{
						"icon": "📱",
						"title": "الإلكترونيات",
						"text": "هاتف + كفر، لابتوب + شاحن، شاشة + واقٍ."
					},
					{
						"icon": "💄",
						"title": "التجميل",
						"text": "كريم أساس + كونسيلر، أحمر شفاه + برايمر، توليفات عناية."
					},
					{
						"icon": "🏠",
						"title": "المنزل والأدوات",
						"text": "دهان + فرشاة، مسامير + مطرقة، كابلات + محوّلات."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "جاهز لمضاعفة مبيعاتك؟",
				"text": "ثبّت مساعد نقاط البيع الذكي اليوم. مجاناً، بلا إعداد، ومعدل قبول 42% عندما تكون الاقتراحات مناسبة.",
				"actions": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary"
				}, {
					"label": "دعم واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var insights_ai_pos_copilot_default = {
	en: en$15,
	ar: ar$15
};
//#endregion
//#region src/content/data/pages/insights.json
var insights_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$14,
	default: () => insights_default,
	en: () => en$14
});
var en$14 = {
	"meta": {
		"title": "AI Copilots for Odoo: Blog, Guides & Case Studies | XamlTech",
		"description": "AI Marketing Copilot and AI POS Copilot — guides, case studies and the ROI math behind AI copilots for Odoo. From the XamlTech engineering team, Aswan, Egypt.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📰 Insights",
				"title": "AI Copilots for Odoo:",
				"accent": "Blog, Guides & Case Studies",
				"lead": "Practical guides and real numbers from the AI Marketing Copilot and AI POS Copilot — written by the engineers who built them.",
				"ctas": [{
					"label": "Explore AI solutions",
					"href": "/ai-solutions?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Browse the addons store",
					"href": "/addons-store?lang=en",
					"style": "ghost"
				}]
			}
		},
		{
			"partial": "articles",
			"data": {
				"eyebrow": "Featured guides",
				"title": "Read the AI copilot guides",
				"sub": "Deep dives with the ROI math, the pipeline and the rollout steps.",
				"more": "Read guide →",
				"items": [
					{
						"tag": "AI Marketing",
						"title": "AI Marketing Copilot: Stop Wasting Marketing Budget",
						"text": "Why teams see 35% ROI improvements, 18% lower cost and half their marketing decisions automated.",
						"href": "/insights/ai-marketing-copilot?lang=en"
					},
					{
						"tag": "AI POS",
						"title": "AI POS Copilot: Sell More at Every Transaction",
						"text": "Real-time upsell suggestions in under 100ms — how the retail chain case study reached +24% basket size.",
						"href": "/insights/ai-pos-copilot?lang=en"
					},
					{
						"tag": "Setup",
						"title": "AI Copilots Installation Guide",
						"text": "Step-by-step installation, demo data and first recommendations. From install to first AI suggestion in 5 minutes.",
						"href": "/ai-solutions?lang=en"
					},
					{
						"tag": "Engineering",
						"title": "Vibe Coding: Building AI-Native Odoo Modules",
						"text": "How XamlTech uses AI-assisted development to ship production Odoo modules faster.",
						"href": "/knowledge?lang=en"
					},
					{
						"tag": "Public Sector",
						"title": "Government ERP: Odoo 19 for Public Sector",
						"text": "A full HR, payroll and medical claims system for an Egyptian government authority, compliant with Civil Service Law 81/2016.",
						"href": "/solutions/erp?lang=en"
					},
					{
						"tag": "Agriculture",
						"title": "Smart Agriculture: IoT Meets Odoo ERP",
						"text": "Precision farming with sensor data, automated irrigation scheduling and crop yield prediction.",
						"href": "/erp-agriculture-egypt?lang=en"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "FAQ",
				"title": "Common questions about AI copilots",
				"items": [
					{
						"q": "What's the difference between the two copilots?",
						"a": "AI Marketing Copilot optimizes how you acquire customers — campaign ROI, budget allocation and lead scoring. AI POS Copilot optimizes what you sell at the register — real-time complementary-product suggestions."
					},
					{
						"q": "Are the copilots free?",
						"a": "Yes, both AI Marketing Copilot and AI POS Copilot are free to install from the Odoo Apps Store — no licensing fees, no per-user charges."
					},
					{
						"q": "Where does my data go?",
						"a": "All data stays within your Odoo instance. No cloud APIs, no external processing, no third-party data storage — the AI runs locally on your server."
					},
					{
						"q": "Do they integrate with other tools?",
						"a": "Yes. Both modules provide Python APIs and JSON webhooks, so you can send recommendations to HubSpot, Klaviyo, email platforms, ad networks or your CRM."
					},
					{
						"q": "Is configuration required?",
						"a": "No. Both copilots work out of the box with zero configuration. Install, load demo data and start getting AI-powered recommendations immediately."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to transform your business?",
				"text": "Install the AI Copilots from the Odoo Apps Store today — no credit card, demo data included, zero configuration.",
				"actions": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp support",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$14 = {
	"meta": {
		"title": "مساعدو الذكاء الاصطناعي لـ Odoo: مدونة وأدلة ودراسات حالة | زامل تك",
		"description": "مساعد التسويق الذكي ومساعد نقاط البيع الذكي — أدلة ودراسات حالة وحسابات العائد. من فريق هندسة زامل تك، أسوان، مصر.",
		"og_image": "https://xamltech.com/static/img/ai-copilot.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📰 رؤى",
				"title": "مساعدو الذكاء الاصطناعي لـ Odoo:",
				"accent": "مدونة وأدلة ودراسات حالة",
				"lead": "أدلة عملية وأرقام حقيقية من مساعد التسويق الذكي ومساعد نقاط البيع الذكي — بقلم المهندسين الذين بنوهم.",
				"ctas": [{
					"label": "استكشف حلول الذكاء",
					"href": "/ai-solutions",
					"style": "primary",
					"big": true
				}, {
					"label": "تصفّح متجر الإضافات",
					"href": "/addons-store",
					"style": "ghost"
				}]
			}
		},
		{
			"partial": "articles",
			"data": {
				"eyebrow": "أدلة مميزة",
				"title": "اقرأ أدلة مساعدي الذكاء",
				"sub": "تعمّق في حسابات العائد وخط الأنابيب وخطوات الإطلاق.",
				"more": "اقرأ الدليل ←",
				"items": [
					{
						"tag": "التسويق الذكي",
						"title": "مساعد التسويق الذكي: أوقف إهدار ميزانية التسويق",
						"text": "لماذا يحقق الفريق تحسناً بنسبة 35% في العائد و18% خفضاً في التكلفة ونصف قرارات التسويق آلياً.",
						"href": "/insights/ai-marketing-copilot"
					},
					{
						"tag": "نقاط البيع الذكية",
						"title": "مساعد نقاط البيع الذكي: بِع أكثر في كل عملية",
						"text": "اقتراحات بيع إضافي فورية في أقل من 100ms — كيف حققت دراسة سلسلة التجزئة زيادة 24% في سلة الشراء.",
						"href": "/insights/ai-pos-copilot"
					},
					{
						"tag": "التثبيت",
						"title": "دليل تثبيت مساعدي الذكاء",
						"text": "تثبيت خطوة بخطوة وبيانات تجريبية وأول التوصيات. من التثبيت لأول اقتراح في 5 دقائق.",
						"href": "/ai-solutions"
					},
					{
						"tag": "الهندسة",
						"title": "Vibe Coding: بناء موديولات Odoo الذكية",
						"text": "كيف تستخدم زامل تك التطوير بمساعدة الذكاء الاصطناعي لإصدار موديولات Odoo أسرع.",
						"href": "/knowledge"
					},
					{
						"tag": "القطاع الحكومي",
						"title": "ERP حكومي: Odoo 19 للقطاع العام",
						"text": "نظام موارد بشرية ومرتبات ومطالبات طبية كامل لهيئة حكومية مصرية بتوافق قانون الخدمة المدنية 81/2016.",
						"href": "/solutions/erp"
					},
					{
						"tag": "الزراعة",
						"title": "الزراعة الذكية: إنترنت الأشياء مع Odoo ERP",
						"text": "زراعة دقيقة ببيانات المستشعرات وجدولة ري آلية والتنبؤ بمحصول.",
						"href": "/erp-agriculture-egypt"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "الأسئلة الشائعة",
				"title": "أسئلة شائعة عن مساعدي الذكاء",
				"items": [
					{
						"q": "ما الفرق بين المساعدين؟",
						"a": "مساعد التسويق الذكي يحسّن كيفية اكتساب العملاء — عائد الحملات وتوزيع الميزانية وتقييم العملاء المحتملين. ومساعد نقاط البيع يحسّن ما تبيعه عند الكاشير — اقتراحات فورية للمنتجات المكمّلة."
					},
					{
						"q": "هل المساعدان مجانيان؟",
						"a": "نعم، مساعد التسويق الذكي ومساعد نقاط البيع الذكي مجانيان للتثبيت من متجر Odoo — بلا رسوم ترخيص أو رسوم لكل مستخدم."
					},
					{
						"q": "إلى أين تذهب بياناتي؟",
						"a": "تبقى كل البيانات داخل نسخة Odoo لديك. بلا واجهات سحابية أو معالجة خارجية أو تخزين لدى طرف ثالث — يعمل الذكاء محلياً على سيرفرك."
					},
					{
						"q": "هل يتكاملان مع أدوات أخرى؟",
						"a": "نعم. يوفر الموديولان واجهات Python وwebhooks بصيغة JSON، فترسل التوصيات إلى HubSpot وKlaviyo ومنصات البريد وشبكات الإعلان أو نظام CRM."
					},
					{
						"q": "هل يلزم إعداد؟",
						"a": "لا. يعمل المساعدان فوراً بلا إعداد. ثبّت، وحمّل البيانات التجريبية، وابدأ بالتوصيات الذكية مباشرة."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "جاهز لتحويل أعمالك؟",
				"text": "ثبّت مساعدي الذكاء من متجر Odoo اليوم — بلا بطاقة ائتمانية، ببيانات تجريبية، وبلا إعداد.",
				"actions": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary"
				}, {
					"label": "دعم واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var insights_default = {
	en: en$14,
	ar: ar$14
};
//#endregion
//#region src/content/data/pages/knowledge.json
var knowledge_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$13,
	default: () => knowledge_default,
	en: () => en$13
});
var en$13 = {
	"meta": {
		"title": "Odoo Development Knowledge Base — Custom ERP Solutions | XamlTech Egypt",
		"description": "Real technical solutions from production deployments — Odoo ERP customisation, POS module development, ZATCA e-invoicing, agriculture ERP, network infrastructure and AI.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📚 Technical Knowledge Base",
				"title": "Odoo development & IT solutions knowledge base",
				"lead": "Real technical solutions from our production deployments — Odoo ERP customisation, POS module development, network infrastructure and CCTV systems.",
				"ctas": [{
					"label": "Talk to us about a project",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}]
			}
		},
		{
			"partial": "articles",
			"data": {
				"eyebrow": "Technical guides",
				"title": "From our engineering blog",
				"sub": "How we actually build things.",
				"more": "Read guide →",
				"items": [
					{
						"tag": "Textile POS",
						"title": "How we build custom Odoo POS modules for textile businesses",
						"text": "Extending PosOrder with cutting fields, OWL v3 migration lessons and real-time stock sync.",
						"href": "/insights"
					},
					{
						"tag": "ZATCA",
						"title": "Implementing ZATCA e-invoicing compliance in Odoo",
						"text": "A dual-system architecture for XML signing, QR embedding and API submission.",
						"href": "/insights"
					},
					{
						"tag": "Agriculture",
						"title": "Building agriculture ERP with Odoo for Qatf Company",
						"text": "Hierarchical locations, work orders and harvest analytics for real farms.",
						"href": "/insights"
					},
					{
						"tag": "Infrastructure",
						"title": "Hospital IT infrastructure: Furkan Surgical Center case study",
						"text": "Structured cabling, VLANs, a 128-channel CCTV system and a Dockerised software stack.",
						"href": "/portfolio"
					},
					{
						"tag": "AI",
						"title": "How to implement AI intelligence in Odoo POS and marketing",
						"text": "Two production-tested AI modules that bring automation into Odoo.",
						"href": "/ai-solutions"
					},
					{
						"tag": "Architecture",
						"title": "Odoo AI integration architecture: clean design for long-term maintenance",
						"text": "DDD, clean architecture and TDD for AI modules that scale.",
						"href": "/ai-solutions"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "FAQ",
				"title": "Frequently asked questions",
				"items": [
					{
						"q": "What Odoo versions do you support?",
						"a": "We develop for Odoo 16, 17, 18 and 19 — both Enterprise and Community. Our published addons target Odoo 18 and 19."
					},
					{
						"q": "Can you customise Odoo POS for my specific business?",
						"a": "Yes. We've built POS customisations for textile (cutting orders), agriculture (harvest tracking) and retail (multi-store sync), plus the POS Security Framework."
					},
					{
						"q": "Do you work outside Aswan?",
						"a": "Yes. While based in Aswan, we serve clients across Egypt and Saudi Arabia, with ERP handled remotely and on-site visits for infrastructure."
					},
					{
						"q": "What makes your Odoo modules different?",
						"a": "They are built from real production deployments — each addon solves a specific problem encountered with real clients."
					},
					{
						"q": "How long does an Odoo ERP implementation take?",
						"a": "Starter (3 modules): 2–4 weeks. Business (8+ modules): 2–3 months. Enterprise (unlimited modules, custom dev): 3–6 months."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Have a technical question?",
				"text": "We share real solutions from real deployments. Contact us for technical consulting.",
				"actions": [{
					"label": "Contact us",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$13 = {
	"meta": {
		"title": "قاعدة معرفة تطوير Odoo — حلول ERP مخصصة | زامل تك مصر",
		"description": "حلول تقنية حقيقية من تنفيذاتنا الإنتاجية — تخصيص Odoo ERP وتطوير موديولات نقاط البيع وفاتورة ZATCA وERP الزراعة والبنية التحتية والذكاء الاصطناعي.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📚 قاعدة معرفة تقنية",
				"title": "قاعدة معرفة تطوير Odoo وحلول تقنية المعلومات",
				"lead": "حلول تقنية حقيقية من تنفيذاتنا — تخصيص Odoo ERP وتطوير موديولات نقاط البيع والبنية التحتية وأنظمة الكاميرات.",
				"ctas": [{
					"label": "تحدّث معنا عن مشروعك",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}]
			}
		},
		{
			"partial": "articles",
			"data": {
				"eyebrow": "أدلة تقنية",
				"title": "من مدونة هندستنا",
				"sub": "كيف نبني الأشياء فعلاً.",
				"more": "اقرأ الدليل ←",
				"items": [
					{
						"tag": "نقاط بيع المنسوجات",
						"title": "كيف نبني موديولات Odoo POS مخصصة لمنسوجات",
						"text": "توسيع PosOrder بحقول القص ودروس الترحيل إلى OWL v3 ومزامنة المخزون الفورية.",
						"href": "/insights"
					},
					{
						"tag": "ZATCA",
						"title": "تنفيذ توافق فاتورة ZATCA الإلكترونية في Odoo",
						"text": "بنية نظام مزدوج لتوقيع XML وتضمين QR والتقديم عبر API.",
						"href": "/insights"
					},
					{
						"tag": "الزراعة",
						"title": "بناء ERP زراعي بـ Odoo لشركة قطف",
						"text": "مواقع هرمية وأوامر عمل وتحليلات حصاد لمزارع حقيقية.",
						"href": "/insights"
					},
					{
						"tag": "البنية التحتية",
						"title": "بنية مستشفى تقنية: دراسة حالة مركز الفرقان",
						"text": "تمديدات منظمة وVLANs ونظام كاميرات بـ 128 قناة ومنظومة برمجيات بحاويات Docker.",
						"href": "/portfolio"
					},
					{
						"tag": "الذكاء الاصطناعي",
						"title": "كيف تطبّق ذكاءً اصطناعياً في نقاط البيع والتسويق",
						"text": "موديولان مُختبران في الإنتاج يجلبان الأتمتة إلى Odoo.",
						"href": "/ai-solutions"
					},
					{
						"tag": "الهندسة",
						"title": "هندسة تكامل الذكاء الاصطناعي في Odoo لحفظ طويل الأمد",
						"text": "DDD وبنية نظيفة وTDD لموديولات ذكاء قابلة للتوسع.",
						"href": "/ai-solutions"
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"alt": true,
				"eyebrow": "الأسئلة الشائعة",
				"title": "أسئلة شائعة",
				"items": [
					{
						"q": "ما إصدارات Odoo التي تدعمونها؟",
						"a": "نطوّر لـ Odoo 16 و17 و18 و19 — بنسختي Enterprise وCommunity. وإضافاتنا المنشورة تستهدف 18 و19."
					},
					{
						"q": "هل تخصّصون نقاط بيع Odoo لنشاطي؟",
						"a": "نعم. بنينا تخصيصات للمنسوجات (أوامر القص) والزراعة (تتبع الحصاد) والتجزئة (مزامنة الفروع)، مع إطار الأمان."
					},
					{
						"q": "هل تعملون خارج أسوان؟",
						"a": "نعم. مقرنا أسوان ونخدم عملاء في مصر والسعودية، مع إدارة مشاريع ERP عن بُعد وزيارات ميدانية للبنية التحتية."
					},
					{
						"q": "ما الذي يميز موديولاتكم؟",
						"a": "مبنية من تنفيذات إنتاجية حقيقية — كل إضافة تحل مشكلة واجهناها مع عميل فعلي."
					},
					{
						"q": "كم يستغرق تنفيذ نظام Odoo ERP؟",
						"a": "Starter (3 موديولات): 2–4 أسابيع. Business (+8): 2–3 أشهر. Enterprise (غير محدود + تطوير مخصص): 3–6 أشهر."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "لديك سؤال تقني؟",
				"text": "نشارك حلولاً حقيقية من تنفيذات حقيقية. تواصل معنا للاستشارات التقنية.",
				"actions": [{
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var knowledge_default = {
	en: en$13,
	ar: ar$13
};
//#endregion
//#region src/content/data/pages/networktool.json
var networktool_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$12,
	default: () => networktool_default,
	en: () => en$12
});
var en$12 = {
	"meta": {
		"title": "Network Copilot — 40+ Android Network Diagnostic Tools | XamlTech",
		"description": "Network Copilot: 40+ network diagnostics, an explanation engine and built-in AI — offline-first on Android 10-16. Ping, DNS, Wi-Fi analyzer, LAN lab, no-root firewall.",
		"og_image": "https://xamltech.com/static/img/networktool-dashboard.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📡 Network Copilot · Android 10+ · Offline-First",
				"title": "Diagnose Like a Network Engineer —",
				"accent": "Right From Your Phone",
				"lead": "Meet Network Copilot — a professional network diagnostics and operations toolkit that runs offline-first on your device. 40+ tools, an explainable engine and built-in AI.",
				"points": [
					"40+ tools in one app",
					"Evidence → Inference → Action engine",
					"Privacy by default — no cloud required"
				],
				"ctas": [{
					"label": "Request download",
					"href": "/apk-download?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Chat on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"external": true
				}],
				"stats": [
					{
						"v": "40+",
						"l": "Diagnostic tools"
					},
					{
						"v": "0",
						"l": "Cloud required"
					},
					{
						"v": "10–16",
						"l": "Android versions"
					},
					{
						"v": "AES-256",
						"l": "On-device key storage"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Toolkit",
				"title": "Everything a field engineer needs",
				"sub": "A command-center of diagnostics and controls, offline-first by design.",
				"items": [
					{
						"icon": "🧭",
						"title": "Command-Center Dashboard",
						"text": "Live connectivity (Wi-Fi, cellular, VPN, Ethernet, metered), interface details — IPv4/IPv6, gateway, MTU — quick-launch tool tiles and an explanation engine."
					},
					{
						"icon": "🛠️",
						"title": "Core Diagnostics",
						"text": "Ping with jitter & loss, DNS lookups (A/AAAA/MX/NS/TXT/CNAME + PTR), an HTTP/HTTPS builder with DNS→TLS→first-byte timing breakdown, speed tests."
					},
					{
						"icon": "📶",
						"title": "WiFi & Signal Analyzer",
						"text": "Live band, link speed, BSSID and security, nearby network scans, and a precise room-by-room signal mapper."
					},
					{
						"icon": "🧪",
						"title": "LAN, IP & Protocol Lab",
						"text": "Subnet / IPv4 CIDR and IPv6 calculators, advanced IPv6 analysis (multicast scope, embedded v4, NAT64), mDNS/UPnP discovery and Wake-on-LAN."
					},
					{
						"icon": "🧠",
						"title": "Intelligent Engine",
						"text": "Rule-based diagnostics on an Evidence → Inference → Action model with confidence levels, custom JSON rules and professional reports (Markdown, JSON, HTML)."
					},
					{
						"icon": "🔐",
						"title": "Privacy by Default",
						"text": "Masked IPs, MACs and SSIDs with one-tap reveal, encrypted on-device key storage (AES-256-GCM), no cloud required, and reports you fully control."
					},
					{
						"icon": "📈",
						"title": "Monitor & Control",
						"text": "Live system monitor with a foreground-service notification and home-screen widget, plus Quick Settings tiles (Scan, Monitor, Speed, Port, Firewall)."
					},
					{
						"icon": "🛡️",
						"title": "App Firewall (No-Root)",
						"text": "A local VPN-based firewall that blocks selected apps entirely by dropping their traffic on-device — no byte ever leaves your phone."
					},
					{
						"icon": "👥",
						"title": "Accounts & Sync",
						"text": "Google Sign-In, history synced under your own user scope, projects shared with teams — while anonymous, fully offline use needs no account."
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "🤖 Built-in AI",
				"title": "The AI Network Analyst",
				"lead": "Turns raw scan results into plain-language answers, root-cause analysis and professional reports — your keys, stored on your device.",
				"points": [
					"Analyze — break a scan or result into a clear, structured diagnosis",
					"Explain — plain-language meaning of every number and field",
					"Troubleshoot — step-by-step fix guidance based on detected symptoms",
					"Report — a complete diagnostic report as Markdown, JSON or HTML"
				],
				"media": {
					"src": "/static/img/networktool-tools.png",
					"alt": "Network Copilot AI Copilot screen"
				}
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Screenshots",
				"title": "See it in action",
				"cols": "3",
				"items": [
					{
						"img": "/static/img/networktool-dashboard.png",
						"alt": "Network Copilot dashboard",
						"title": "Command-Center Dashboard",
						"text": "Live connectivity, interface details and tool tiles."
					},
					{
						"img": "/static/img/networktool-dashboard.png",
						"alt": "Network Copilot tools grid",
						"title": "Tools Grid — 40+ Diagnostics",
						"text": "Every tool one tap away."
					},
					{
						"img": "/static/img/networktool-internet-health.png",
						"alt": "Network Copilot AI copilot",
						"title": "AI Copilot & Local Ollama",
						"text": "Cloud or a local model on your LAN."
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"eyebrow": "🧪 Early Access",
				"title": "Become a tester",
				"sub": "Network Copilot v5.0.4 is in closed tester release across Android 10–16 (API 29–36), from narrow phones to tablets.",
				"items": [
					{
						"q": "Which devices are covered?",
						"a": "Android 10–16 (API 29–36), from narrow phones to tablets. Testers report crashes automatically via Firebase Crashlytics."
					},
					{
						"q": "What's next on the roadmap?",
						"a": "Public launch (production ad units, FCM push, scoped settings), tester-loop hardening, engineering v4 (deeper SNMP/UPnP, on-device AI inference) and team analytics."
					},
					{
						"q": "Is my data uploaded?",
						"a": "No. Diagnostics run on your device; nothing leaves it unless you explicitly export or share a report."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Want early access?",
				"text": "Request a download link and join the tester program.",
				"actions": [{
					"label": "Request download",
					"href": "/apk-download?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$12 = {
	"meta": {
		"title": "Network Copilot — أكثر من 40 أداة تشخيص شبكات لأندرويد | زامل تك",
		"description": "Network Copilot: أكثر من 40 أداة تشخيص ومحرك شرح وذكاء اصطناعي مدمج — يعمل دون اتصال على أندرويد 10-16. بينغ وDNS ومحلل واي فاي ومختبر الشبكة وجدار حماية بلا روت.",
		"og_image": "https://xamltech.com/static/img/networktool-dashboard.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "📡 Network Copilot · أندرويد 10+ · يعمل دون اتصال",
				"title": "شخّص كخبير شبكات —",
				"accent": "من هاتفك مباشرة",
				"lead": "تعرّف على Network Copilot — مجموعة احترافية لتشخيص الشبكات والعمليات تعمل دون اتصال على جهازك. أكثر من 40 أداة ومحرك قابل للشرح وذكاء اصطناعي مدمج.",
				"points": [
					"أكثر من 40 أداة في تطبيق واحد",
					"محرك Evidence → Inference → Action",
					"خصوصية افتراضية — بلا حاجة لسحابة"
				],
				"ctas": [{
					"label": "اطلب التحميل",
					"href": "/apk-download",
					"style": "primary",
					"big": true
				}, {
					"label": "محادثة واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"external": true
				}],
				"stats": [
					{
						"v": "+40",
						"l": "أداة تشخيص"
					},
					{
						"v": "0",
						"l": "حاجة لسحابة"
					},
					{
						"v": "10–16",
						"l": "إصدارات أندرويد"
					},
					{
						"v": "AES-256",
						"l": "تخزين مفاتيح على الجهاز"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "العدة",
				"title": "كل ما يحتاجه مهندس الميدان",
				"sub": "مركز قيادة للتشخيص والتحكم، مصمم ليعمل دون اتصال.",
				"items": [
					{
						"icon": "🧭",
						"title": "لوحة مركز القيادة",
						"text": "حالة الاتصال (واي فاي، خلوي، VPN، إيثرنت)، تفاصيل الواجهة — IPv4/IPv6 والبوابة وMTU — وبلاطات أدوات سريعة ومحرك شرح."
					},
					{
						"icon": "🛠️",
						"title": "أدوات التشخيص الأساسية",
						"text": "بينغ مع التذبذب والفقد، استعلامات DNS (A/AAAA/MX/NS/TXT/CNAME + PTR)، وبناء HTTP/HTTPS مع تفصيل توقيت DNS→TLS→أول بايت."
					},
					{
						"icon": "📶",
						"title": "محلل الواي فاي والإشارة",
						"text": "النطاق الحي وسرعة الرابط وBSSID والأمان وفحص الشبكات القريبة ومخطط إشارة دقيق غرفة بغرفة."
					},
					{
						"icon": "🧪",
						"title": "مختبر LAN وIP والبروتوكولات",
						"text": "حاسبات الشبكة الفرعية وIPv4 CIDR وIPv6، وتحليل IPv6 المتقدم (نطاق البث المتعدد، v4 المضمّن، NAT64)، واكتشاف mDNS/UPnP وWake-on-LAN."
					},
					{
						"icon": "🧠",
						"title": "المحرك الذكي",
						"text": "تشخيص قائم على القواعد بنموذج Evidence → Inference → Action مع مستويات ثقة وقواعد JSON مخصصة وتقارير احترافية (Markdown وJSON وHTML)."
					},
					{
						"icon": "🔐",
						"title": "خصوصية افتراضية",
						"text": "إخفاء عناوين IP وMAC وSSID بلمسة، وتخزين مفاتيح مشفّر (AES-256-GCM)، وبلا سحابة، وتقارير تتحكم بها بالكامل."
					},
					{
						"icon": "📈",
						"title": "المراقبة والتحكم",
						"text": "مراقب نظام حي مع إشعار خدمة أمامية وويدجت الشاشة، وبلاطات الإعدادات السريعة (فحص، مراقبة، سرعة، منفذ، جدار ناري)."
					},
					{
						"icon": "🛡️",
						"title": "جدار حماية بلا روت",
						"text": "جدار حماية محلي قائم على VPN يحجب تطبيقات مختارة بإسقاط حركتها على الجهاز — لا يغادر أي بايت هاتفك."
					},
					{
						"icon": "👥",
						"title": "الحسابات والمزامنة",
						"text": "تسجيل دخول Google ومزامنة السجل تحت نطاقك، ومشاريع تشاركها مع الفريق — بينما يعمل الاستخدام المجهول دون اتصال بلا حساب."
					}
				]
			}
		},
		{
			"partial": "split",
			"data": {
				"reverse": true,
				"eyebrow": "🤖 ذكاء اصطناعي مدمج",
				"title": "محلل الشبكة الذكي",
				"lead": "يحوّل نتائج الفحص الخام إلى إجابات بلغة واضحة وتحليل للسبب الجذري وتقارير احترافية — بمفاتيحك المخزنة على جهازك.",
				"points": [
					"تحليل — تفكيك فحص أو نتيجة إلى تشخيص واضح ومنظم",
					"شرح — معنى كل رقم وحقل بلغة مبسطة",
					"حل — إرشاد إصلاح خطوة بخطوة حسب الأعراض",
					"تقرير — تقرير تشخيص كامل بصيغة Markdown أو JSON أو HTML"
				],
				"media": {
					"src": "/static/img/networktool-tools.png",
					"alt": "شاشة مساعد الذكاء الاصطناعي في Network Copilot"
				}
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "لقطات الشاشة",
				"title": "شاهده أثناء العمل",
				"cols": "3",
				"items": [
					{
						"img": "/static/img/networktool-dashboard.png",
						"alt": "لوحة Network Copilot",
						"title": "لوحة مركز القيادة",
						"text": "حالة الاتصال وتفاصيل الواجهة وبلاطات الأدوات."
					},
					{
						"img": "/static/img/networktool-dashboard.png",
						"alt": "شبكة أدوات Network Copilot",
						"title": "شبكة الأدوات — أكثر من 40",
						"text": "كل أداة على بعد لمسة."
					},
					{
						"img": "/static/img/networktool-internet-health.png",
						"alt": "مساعد Network Copilot",
						"title": "المساعد الذكي وOllama المحلي",
						"text": "سحابة أو نموذج محلي على شبكتك."
					}
				]
			}
		},
		{
			"partial": "faq",
			"data": {
				"eyebrow": "🧪 وصول مبكر",
				"title": "كن مختبِراً",
				"sub": "Network Copilot v5.0.4 متاح بنسخة مختبرين مغلقة لأندرويد 10–16 (API 29–36)، من الهواتف الضيقة إلى الأجهزة اللوحية.",
				"items": [
					{
						"q": "ما الأجهزة المدعومة؟",
						"a": "أندرويد 10–16 (API 29–36)، من الهواتف الضيقة إلى الأجهزة اللوحية. يرفع المختبِرون تقارير الأعطال تلقائياً عبر Firebase Crashlytics."
					},
					{
						"q": "ما التالي في خطة التطوير؟",
						"a": "الإطلاق العام (وحدات إعلانية إنتاجية، إشعارات FCM، إعدادات محددة)، وتقوية حلقة المختبرين، والهندسة v4 (SNMP/UPnP أعمق، استدلال ذكاء على الجهاز)، وتحليلات الفريق."
					},
					{
						"q": "هل تُرفع بياناتي؟",
						"a": "لا. يعمل التشخيص على جهازك ولا يغادره شيء إلا إذا صدّرت أو شاركت تقريراً بنفسك."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "تريد وصولاً مبكراً؟",
				"text": "اطلب رابط تحميل وانضم إلى برنامج المختبرين.",
				"actions": [{
					"label": "اطلب التحميل",
					"href": "/apk-download",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var networktool_default = {
	en: en$12,
	ar: ar$12
};
//#endregion
//#region src/content/data/pages/odoo-pos-egypt.json
var odoo_pos_egypt_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$11,
	default: () => odoo_pos_egypt_default,
	en: () => en$11
});
var en$11 = {
	"meta": {
		"title": "Odoo POS System for Retail, Restaurants & Textile in Egypt — XamlTech",
		"description": "Complete Odoo point of sale for Egypt — works offline, multi-store sync, barcode scanning, receipt printing, custom POS modules on the Odoo Apps Store.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🖥️ Odoo POS Solutions",
				"title": "Odoo POS for retail, restaurants & textile in Egypt",
				"lead": "Complete point-of-sale deployment — software, hardware, training and support. Works offline, syncs across stores, prints receipts and scans barcodes. From a single store to a multi-branch chain.",
				"points": [
					"Works offline — automatic sync",
					"Multi-branch central management",
					"Receipt printing — Epson, Star, any printer",
					"Tablet, laptop, desktop — any browser",
					"Inventory + accounting + CRM built in"
				],
				"ctas": [{
					"label": "Try the live demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Free consultation on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Built for your business type",
				"title": "Every business adapts differently",
				"sub": "Here is how Odoo POS maps to yours.",
				"cols": 3,
				"items": [
					{
						"icon": "🛒",
						"title": "Supermarket & grocery",
						"text": "Barcode scanning, weight-based products, loyalty cards, daily cash reconciliation and multi-cashier shift management."
					},
					{
						"icon": "🍽️",
						"title": "Café & restaurant",
						"text": "Floor plans, table management, kitchen display (KDS), split bills, tips and kitchen printing."
					},
					{
						"icon": "🧵",
						"title": "Textile & fabric",
						"text": "Our specialty. POS Cutting Orders manages fabric cutting from the POS — dimensions, quantities, fabric type, with real-time stock per roll."
					},
					{
						"icon": "💊",
						"title": "Pharmacy",
						"text": "Expiry tracking, lot/serial numbers, prescriptions and insurance integration with barcode scanning."
					},
					{
						"icon": "👗",
						"title": "Fashion & clothing",
						"text": "Size and colour variants, tagging, customer profiles, loyalty and multi-location stock visibility."
					},
					{
						"icon": "🏬",
						"title": "Multi-store chain",
						"text": "Central management for unlimited locations, per-store pricelists, warehouses and real-time inventory sync."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "POS features",
				"title": "Odoo POS features",
				"sub": "The capabilities that keep checkout fast and accurate.",
				"cols": 3,
				"items": [
					{
						"icon": "📴",
						"title": "Offline mode",
						"text": "Internet drops? POS keeps working and every order syncs automatically when the connection returns."
					},
					{
						"icon": "📦",
						"title": "Real-time inventory",
						"text": "Every sale updates stock instantly across locations, with reactive quantity badges on product cards."
					},
					{
						"icon": "🏷️",
						"title": "Barcode & scanning",
						"text": "Plug-and-play USB HID scanners for products, customer cards and coupons; print your own barcodes."
					},
					{
						"icon": "🔐",
						"title": "Security controls",
						"text": "Manager PIN for refunds and discounts, per-user discount limits and backend validation that prevents bypasses."
					},
					{
						"icon": "💳",
						"title": "Payment methods",
						"text": "Cash, card, bank transfer and mobile payments, with split tender and credit-limit accounts."
					},
					{
						"icon": "🎁",
						"title": "Loyalty & promotions",
						"text": "Points, automatic discounts, buy-X-get-Y, coupon codes and gift cards — all configurable."
					},
					{
						"icon": "🧾",
						"title": "Receipt printing",
						"text": "ESC/POS thermal printers with custom layout, logo, Arabic text and QR codes; kitchen printing for restaurants."
					},
					{
						"icon": "📱",
						"title": "Any device",
						"text": "Runs on any browser — iPad, Android tablet, laptop, desktop or touchscreen. No special hardware."
					},
					{
						"icon": "📊",
						"title": "Reports & analytics",
						"text": "Daily sales, per-product analysis, cashier performance and payment breakdowns, exportable to Excel."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "🛒 Published on the Odoo Apps Store",
				"title": "Our custom POS modules",
				"sub": "Built from real client needs and published on the official store.",
				"cols": 3,
				"items": [
					{
						"icon": "✂️",
						"title": "POS Cutting Orders",
						"text": "Manage fabric cutting from the POS — dimensions, quantities and fabric type, with a dedicated screen and manager PIN approvals."
					},
					{
						"icon": "📦",
						"title": "POS Stock Real-Time",
						"text": "Live inventory badges, batch loading for large catalogues, navbar refresh and reactive updates with zero lag on 10K+ products."
					},
					{
						"icon": "🔐",
						"title": "POS Security Framework",
						"text": "Manager PIN for session close, refunds and critical operations, with discount limits at three levels and backend validation."
					}
				],
				"cta": {
					"label": "Browse the addons store",
					"href": "/addons-store?lang=en"
				}
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Hardware",
				"title": "POS hardware setup",
				"sub": "We handle complete procurement, configuration and installation.",
				"cols": 3,
				"items": [
					{
						"icon": "🖨️",
						"title": "Receipt printer",
						"text": "Epson TM-T20III or Star TSP143 — ESC/POS thermal printer, USB or network, with Arabic text, logos and QR codes."
					},
					{
						"icon": "📷",
						"title": "Barcode scanner",
						"text": "USB HID plug-and-play scanner for 1D and 2D barcodes; wireless Bluetooth options available."
					},
					{
						"icon": "💰",
						"title": "Cash drawer",
						"text": "RJ11 cash drawer that opens automatically on cash payment, with manual key override."
					},
					{
						"icon": "🖥️",
						"title": "Customer display",
						"text": "LCD/LED display showing order items and totals in Arabic and English."
					},
					{
						"icon": "💻",
						"title": "POS terminal",
						"text": "Any browser device — iPad, Android tablet, touchscreen PC or laptop. We recommend 10\"+."
					},
					{
						"icon": "🔗",
						"title": "IoT Box (optional)",
						"text": "Connects printers, scanners, scales and displays. Alternative direct USB/network printing available."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Packages",
				"title": "POS packages",
				"cols": 3,
				"items": [
					{
						"icon": "🟢",
						"title": "POS Starter",
						"text": "Single store, one POS. Basic product configuration, receipt printer setup, staff training (2 sessions) and 3 months support."
					},
					{
						"icon": "🔵",
						"title": "POS Business",
						"text": "Multi-store deployment, custom UI/receipts, barcode setup, loyalty, inventory integration, hardware procurement and 6 months support."
					},
					{
						"icon": "🟣",
						"title": "POS Enterprise",
						"text": "Everything in Business plus custom modules (cutting orders, security, workflows), full ERP integration and 12 months support."
					}
				],
				"cta": {
					"label": "Get a POS quote",
					"href": "/contact-us?lang=en"
				}
			}
		},
		{
			"partial": "split",
			"data": {
				"alt": true,
				"eyebrow": "🏆 Real deployment",
				"title": "POS in production: Al-Ghazl Al-Arabi",
				"lead": "A textile trading company needed a POS with fabric cutting management.",
				"points": [
					"Problem: cutting orders had to be created directly from the POS screen",
					"Solution: we built POS Cutting Orders (ma_pos_cutting)",
					"Result: multi-session POS across two locations with real-time stock sync",
					"Cutting orders processed 3× faster than the paper-based system"
				],
				"media": {
					"src": "/static/img/pos-light.png",
					"alt": "POS deployment",
					"window": true,
					"caption": "pos.xamltech.com",
					"live": "Live"
				}
			}
		},
		{
			"partial": "faq",
			"data": {
				"eyebrow": "FAQ",
				"title": "POS frequently asked questions",
				"items": [
					{
						"q": "How much does Odoo POS cost in Egypt?",
						"a": "Odoo Community POS is free; Enterprise requires licensing. Our implementation packages start from a basic single-store setup, and custom modules on the Odoo Apps Store start at €99."
					},
					{
						"q": "Does Odoo POS work without internet?",
						"a": "Yes. Odoo POS is a Progressive Web App that keeps working when the internet drops, syncing all transactions automatically when the connection returns."
					},
					{
						"q": "What printers work with Odoo POS?",
						"a": "ESC/POS thermal printers such as Epson TM-T20III and Star TSP143, connected via the Odoo IoT Box or direct network printing."
					},
					{
						"q": "Can I use Odoo POS for a restaurant?",
						"a": "Yes — floor plans, table management, kitchen display, split bills and tips, with kitchen printing that sends orders straight to the kitchen printer."
					},
					{
						"q": "How long does POS setup take?",
						"a": "Single store: 3–5 days. Multi-store: 1–2 weeks. With custom modules: 2–4 weeks, including product import, hardware setup, receipt configuration and training."
					},
					{
						"q": "Do you serve areas outside Aswan?",
						"a": "Yes. Software setup is done remotely; for hardware installation we travel to your location. We serve all of Egypt, with a focus on Upper Egypt."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to get started with Odoo POS?",
				"text": "Try our live demo — or get a free consultation on WhatsApp.",
				"actions": [{
					"label": "Try the live demo",
					"href": "/request-demo?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$11 = /*#__PURE__*/ JSON.parse("{\"meta\":{\"title\":\"نظام نقاط بيع Odoo للتجزئة والمطاعم والمنسوجات في مصر — زامل تك\",\"description\":\"نظام نقاط بيع Odoo متكامل في مصر — يعمل بدون إنترنت، مزامنة بين الفروع، قراءة باركود، طباعة فواتير، وموديولات مخصصة على متجر Odoo.\",\"og_image\":\"https://xamltech.com/static/img/pos-light.png\",\"index\":true},\"sections\":[{\"partial\":\"hero\",\"data\":{\"eyebrow\":\"🖥️ حلول نقاط البيع\",\"title\":\"نظام نقاط بيع Odoo للتجزئة والمطاعم والمنسوجات في مصر\",\"lead\":\"نشر كامل لنقاط البيع — برمجيات، أجهزة، تدريب ودعم فني. يعمل بدون إنترنت، ويتزامن بين الفروع، ويطبع الفواتير ويقرأ الباركود. من محل واحد إلى سلسلة فروع.\",\"points\":[\"يعمل بدون إنترنت — مزامنة تلقائية\",\"فروع متعددة بإدارة مركزية\",\"طباعة فواتير — Epson، Star، أي طابعة\",\"تابلت أو لابتوب أو كمبيوتر — أي متصفح\",\"مخزون + محاسبة + CRM مدمج\"],\"ctas\":[{\"label\":\"جرّب النظام مباشرة\",\"href\":\"/request-demo\",\"style\":\"primary\",\"big\":true},{\"label\":\"استشارة مجانية على واتساب\",\"href\":\"https://wa.me/201007982468\",\"style\":\"wa\",\"big\":true,\"external\":true}]}},{\"partial\":\"features\",\"data\":{\"eyebrow\":\"مصمم لنوع نشاطك\",\"title\":\"كل نشاط يحتاج حلولاً مختلفة\",\"sub\":\"هكذا تتكيف نقاط بيع Odoo معك.\",\"cols\":3,\"items\":[{\"icon\":\"🛒\",\"title\":\"سوبر ماركت وبقالة\",\"text\":\"قراءة باركود، منتجات بالوزن، بطاقات ولاء، تسوية نقدية يومية وإدارة ورديات متعددة.\"},{\"icon\":\"🍽️\",\"title\":\"كافيهات ومطاعم\",\"text\":\"مخططات صالات، إدارة طاولات، شاشة مطبخ (KDS)، تقسيم الفواتير، بقشيش وطباعة المطبخ.\"},{\"icon\":\"🧵\",\"title\":\"أقمشة ومنسوجات\",\"text\":\"تخصصنا. موديول أوامر القص يدير قص الأقمشة من نقطة البيع — الأبعاد والكميات والنوع مع مخزون فوري لكل لفة.\"},{\"icon\":\"💊\",\"title\":\"صيدليات\",\"text\":\"تتبع تواريخ الصلاحية، أرقام الدفعات/المسلسل، الوصفات وتكامل التأمين مع قراءة الباركود.\"},{\"icon\":\"👗\",\"title\":\"ملابس وأزياء\",\"text\":\"متغيرات المقاس واللون، الوسم، ملفات العملاء، الولاء ورؤية المخزون متعدد المواقع.\"},{\"icon\":\"🏬\",\"title\":\"سلاسل فروع\",\"text\":\"إدارة مركزية لمواقع غير محدودة، قوائم أسعار لكل فرع، مخازن ومزامنة فورية للمخزون.\"}]}},{\"partial\":\"features\",\"data\":{\"alt\":true,\"eyebrow\":\"مميزات النظام\",\"title\":\"مميزات نقاط بيع Odoo\",\"sub\":\"القدرات التي تُبقي الدفع سريعاً ودقيقاً.\",\"cols\":3,\"items\":[{\"icon\":\"📴\",\"title\":\"وضع دون إنترنت\",\"text\":\"انقطع الإنترنت؟ يواصل النظام العمل وتتزامن كل الطلبات تلقائياً عند عودة الاتصال.\"},{\"icon\":\"📦\",\"title\":\"مخزون فوري\",\"text\":\"كل عملية بيع تحدّث المخزون لحظياً في كل المواقع، مع شارات كمية تفاعلية على بطاقات المنتجات.\"},{\"icon\":\"🏷️\",\"title\":\"الباركود والمسح\",\"text\":\"ماسحات USB جاهزة للتشغيل للمنتجات وبطاقات العملاء والكوبونات، مع طباعة باركوداتك.\"},{\"icon\":\"🔐\",\"title\":\"ضوابط أمنية\",\"text\":\"رقم PIN للمدير لعمليات الاسترجاع والخصم، وحدود خصم لكل مستخدم وتحقق خلفي يمنع التجاوز.\"},{\"icon\":\"💳\",\"title\":\"طرق الدفع\",\"text\":\"نقدي، بطاقة، تحويل بنكي ومدفوعات موبايل، مع دفع مقسم وحسابات بحدود ائتمانية.\"},{\"icon\":\"🎁\",\"title\":\"الولاء والعروض\",\"text\":\"نقاط، خصومات تلقائية، اشترِ X واحصل على Y، كوبونات وبطاقات هدايا — كلها قابلة للتهيئة.\"},{\"icon\":\"🧾\",\"title\":\"طباعة الفواتير\",\"text\":\"طابعات حرارية ESC/POS بتصميم مخصص وشعار ونص عربي ورموز QR، وطباعة المطبخ للمطاعم.\"},{\"icon\":\"📱\",\"title\":\"أي جهاز\",\"text\":\"يعمل على أي متصفح — iPad، تابلت أندرويد، لابتوب، كمبيوتر أو شاشة لمس. بلا أجهزة خاصة.\"},{\"icon\":\"📊\",\"title\":\"التقارير والتحليلات\",\"text\":\"مبيعات يومية، تحليل لكل منتج، أداء الكاشير وتفاصيل طرق الدفع، قابلة للتصدير إلى Excel.\"}]}},{\"partial\":\"features\",\"data\":{\"eyebrow\":\"🛒 على متجر Odoo\",\"title\":\"موديولات نقاط البيع المخصصة\",\"sub\":\"بُنيت من احتياجات عملاء حقيقيين ونُشرت على المتجر الرسمي.\",\"cols\":3,\"items\":[{\"icon\":\"✂️\",\"title\":\"أوامر القص\",\"text\":\"إدارة قص الأقمشة من نقطة البيع — الأبعاد والكميات والنوع، مع شاشة مخصصة واعتماد PIN للمدير.\"},{\"icon\":\"📦\",\"title\":\"مخزون فوري\",\"text\":\"شارات مخزون حية، تحميل دفعي للكتالوجات الكبيرة، زر تحديث في الشريط وتحديثات تفاعلية بلا تأخر.\"},{\"icon\":\"🔐\",\"title\":\"إطار الأمان\",\"text\":\"رقم PIN للمدير لإغلاق الجلسة والاسترجاع والعمليات الحساسة، مع حدود خصم بثلاثة مستويات وتحقق خلفي.\"}],\"cta\":{\"label\":\"تصفّح متجر الإضافات\",\"href\":\"/addons-store\"}}},{\"partial\":\"features\",\"data\":{\"alt\":true,\"eyebrow\":\"الأجهزة\",\"title\":\"تجهيز أجهزة نقاط البيع\",\"sub\":\"نتولى التوريد والتهيئة والتركيب كاملاً.\",\"cols\":3,\"items\":[{\"icon\":\"🖨️\",\"title\":\"طابعة الفواتير\",\"text\":\"Epson TM-T20III أو Star TSP143 — طابعة حرارية ESC/POS، USB أو شبكة، مع نص عربي وشعارات ورموز QR.\"},{\"icon\":\"📷\",\"title\":\"ماسح الباركود\",\"text\":\"ماسح USB HID جاهز للتشغيل لباركود 1D و2D، مع خيارات بلوتوث لاسلكية.\"},{\"icon\":\"💰\",\"title\":\"درج النقود\",\"text\":\"درج RJ11 يُفتح تلقائياً عند الدفع النقدي، مع مفتاح يدوي احتياطي.\"},{\"icon\":\"🖥️\",\"title\":\"شاشة العميل\",\"text\":\"شاشة LCD/LED تعرض أصناف الطلب والإجمالي بالعربية والإنجليزية.\"},{\"icon\":\"💻\",\"title\":\"جهاز نقطة البيع\",\"text\":\"أي جهاز بمتصفح — iPad، تابلت أندرويد، كمبيوتر لمسي أو لابتوب. ننصح بـ 10 بوصات وأكثر.\"},{\"icon\":\"🔗\",\"title\":\"صندوق IoT (اختياري)\",\"text\":\"يوصل الطابعات والماسحات والموازين والشاشات، مع بديل الطباعة المباشرة USB/شبكة.\"}]}},{\"partial\":\"features\",\"data\":{\"eyebrow\":\"الباقات\",\"title\":\"باقات نقاط البيع\",\"cols\":3,\"items\":[{\"icon\":\"🟢\",\"title\":\"POS Starter\",\"text\":\"محل واحد ونقطة بيع واحدة. تهيئة أساسية للمنتجات، إعداد الطابعة، تدريب (جلستان) ودعم 3 أشهر.\"},{\"icon\":\"🔵\",\"title\":\"POS Business\",\"text\":\"نشر متعدد الفروع، واجهة/فواتير مخصصة، إعداد الباركود، الولاء، تكامل المخزون، توريد الأجهزة ودعم 6 أشهر.\"},{\"icon\":\"🟣\",\"title\":\"POS Enterprise\",\"text\":\"كل ما في Business مع موديولات مخصصة (أوامر القص، الأمان، سير العمل)، وتكامل ERP كامل ودعم 12 شهراً.\"}],\"cta\":{\"label\":\"اطلب عرض سعر لنقاط البيع\",\"href\":\"/contact-us\"}}},{\"partial\":\"split\",\"data\":{\"alt\":true,\"eyebrow\":\"🏆 تنفيذ حقيقي\",\"title\":\"نقاط البيع في الإنتاج: الغزل العربي\",\"lead\":\"شركة تجارة أقمشة احتاجت نظام نقاط بيع بإدارة قص الأقمشة.\",\"points\":[\"المشكلة: إنشاء أوامر القص مباشرة من شاشة نقطة البيع\",\"الحل: بنينا موديول أوامر القص (ma_pos_cutting)\",\"النتيجة: نقاط بيع بجلسات متعددة في موقعين مع مزامنة فورية\",\"معالجة أوامر القص أسرع 3 مرات من النظام الورقي\"],\"media\":{\"src\":\"/static/img/pos-rtl.png\",\"alt\":\"تنفيذ نقاط بيع\",\"window\":true,\"caption\":\"pos.xamltech.com\",\"live\":\"مباشر\"}}},{\"partial\":\"faq\",\"data\":{\"eyebrow\":\"الأسئلة الشائعة\",\"title\":\"أسئلة شائعة عن نقاط البيع\",\"items\":[{\"q\":\"كم تكلفة نقاط بيع Odoo في مصر؟\",\"a\":\"نسخة Odoo Community مجانية، وEnterprise يتطلب ترخيصاً. باقات التنفيذ تبدأ من إعداد محل واحد، والموديولات المخصصة على متجر Odoo تبدأ من 99 يورو.\"},{\"q\":\"هل تعمل نقاط بيع Odoo بدون إنترنت؟\",\"a\":\"نعم. النظام تطبيق ويب تقدمي يواصل العمل عند انقطاع الإنترنت، وتتزامن كل العمليات تلقائياً عند عودة الاتصال.\"},{\"q\":\"ما الطابعات المدعومة؟\",\"a\":\"الطابعات الحرارية ESC/POS مثل Epson TM-T20III وStar TSP143، عبر صندوق Odoo IoT أو الطباعة المباشرة على الشبكة.\"},{\"q\":\"هل أستطيع استخدام النظام لمطعم؟\",\"a\":\"نعم — مخططات الصالات، إدارة الطاولات، شاشة المطبخ، تقسيم الفواتير والباقشيش، مع طباعة المطبخ.\"},{\"q\":\"كم يستغرق الإعداد؟\",\"a\":\"محل واحد: 3–5 أيام. فروع متعددة: 1–2 أسبوع. مع موديولات مخصصة: 2–4 أسابيع، شاملاً استيراد المنتجات والأجهزة والفواتير والتدريب.\"},{\"q\":\"هل تخدمون مناطق خارج أسوان؟\",\"a\":\"نعم. إعداد البرمجيات يتم عن بُعد، ولتركيب الأجهزة ننزل لموقعك. نخدم كل مصر مع تركيز على الصعيد.\"}]}},{\"partial\":\"cta\",\"data\":{\"title\":\"جاهز تبدأ مع Odoo POS؟\",\"text\":\"جرّب النظام مباشرة — أو احصل على استشارة مجانية على واتساب.\",\"actions\":[{\"label\":\"جرّب النظام مباشرة\",\"href\":\"/request-demo\",\"style\":\"primary\"},{\"label\":\"واتساب\",\"href\":\"https://wa.me/201007982468\",\"style\":\"wa\"}]}}]}");
var odoo_pos_egypt_default = {
	en: en$11,
	ar: ar$11
};
//#endregion
//#region src/content/data/pages/portfolio.json
var portfolio_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$10,
	default: () => portfolio_default,
	en: () => en$10
});
var en$10 = {
	"meta": {
		"title": "Our Projects: Hospital IT, Textile POS, Agriculture ERP — XamlTech Egypt",
		"description": "Trusted by hospitals, businesses and organisations across Egypt. Hospital IT, 128-channel CCTV, textile POS, ZATCA e-invoicing, agriculture ERP and open source.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🏆 Our Work",
				"title": "Partners & projects",
				"lead": "Trusted by hospitals, businesses and organisations across Egypt — and verifiable in the open on GitHub.",
				"ctas": [{
					"label": "Start your project",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "See our GitHub",
					"href": "https://github.com/XamlTech",
					"style": "ghost",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Partners",
				"title": "Our partners",
				"cols": 2,
				"items": [{
					"icon": "🏥",
					"title": "Al-Furqan Specialised Center",
					"text": "IT infrastructure, CCTV, networking and hospital software for a leading surgical center in Aswan."
				}, {
					"icon": "🚑",
					"title": "Red Crescent Hospital",
					"text": "Full hospital IT — patient management, network infrastructure, CCTV surveillance and digital healthcare."
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Project highlights",
				"title": "What we delivered",
				"cols": 3,
				"items": [
					{
						"icon": "🏥",
						"title": "Hospital management",
						"text": "Full Odoo ERP for medical facilities — patients, appointments, pharmacy and billing with Arabic workflows."
					},
					{
						"icon": "📹",
						"title": "Hospital CCTV",
						"text": "128-channel IP camera system — operating rooms, corridors, pharmacy and parking, with 24/7 remote monitoring."
					},
					{
						"icon": "🧵",
						"title": "Textile POS",
						"text": "Custom Odoo 18 POS with cutting orders for Al-Ghazl Al-Arabi — real-time stock and multi-session sync."
					},
					{
						"icon": "🧾",
						"title": "ZATCA e-invoice",
						"text": "Saudi Arabia Phase 2 compliance — XML invoices, QR codes, API submission and government portal sync."
					},
					{
						"icon": "🌾",
						"title": "Farm management",
						"text": "Agriculture ERP for Qatf Company — farm locations, crop tracking, work orders and harvest analytics."
					},
					{
						"icon": "🔗",
						"title": "Medical network",
						"text": "Complete LAN/WLAN for a hospital campus — structured cabling, VLANs, server room and redundant links."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Open source & GitHub",
				"title": "180+ public repositories",
				"sub": "Real code, not just marketing. Verify our skills on GitHub.",
				"cols": 3,
				"items": [
					{
						"icon": "🐍",
						"title": "hospital-flask",
						"text": "Hospital management system in Python Flask — patient registration, appointments, pharmacy and billing."
					},
					{
						"icon": "⚙️",
						"title": "mvc-erp",
						"text": "Full ERP in ASP.NET MVC and C# — accounting, inventory and HR modules across multiple stacks."
					},
					{
						"icon": "🧰",
						"title": "Openrol",
						"text": "Python-based open-source business application demonstrating our Python ecosystem expertise."
					},
					{
						"icon": "🗺️",
						"title": "web_google_maps_drawing",
						"text": "Odoo module for Google Maps with drawing tools — geolocation, polygon mapping and route planning."
					},
					{
						"icon": "🛒",
						"title": "Odoo Apps Store",
						"text": "Published POS modules — Cutting Orders, Stock Real-Time and Security Framework, production-tested."
					},
					{
						"icon": "🌟",
						"title": "180+ repositories",
						"text": "Android, iOS, Python, JavaScript, C#, Java and VR — a full-stack profile with a Google Challenge Scholarship."
					}
				],
				"cta": {
					"label": "View GitHub profile",
					"href": "https://github.com/XamlTech"
				}
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Want to be our next success story?",
				"text": "Tell us about your goals and we'll propose the right approach.",
				"actions": [{
					"label": "Contact us",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$10 = {
	"meta": {
		"title": "مشاريعنا: بنية تحتية للمستشفيات، نقاط بيع للمنسوجات، ERP للزراعة — زامل تك مصر",
		"description": "موثوقون من مستشفيات وشركات ومؤسسات في جميع أنحاء مصر. بنية تحتية للمستشفيات، 128 قناة مراقبة، نقاط بيع للمنسوجات، ZATCA، ERP زراعي ومصادر مفتوحة.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🏆 أعمالنا",
				"title": "الشركاء والأعمال",
				"lead": "موثوقون من مستشفيات وشركات ومؤسسات في جميع أنحاء مصر — وبإمكانك التحقق من كودنا على GitHub.",
				"ctas": [{
					"label": "ابدأ مشروعك",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "زيارة GitHub",
					"href": "https://github.com/XamlTech",
					"style": "ghost",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "الشركاء",
				"title": "شركاؤنا",
				"cols": 2,
				"items": [{
					"icon": "🏥",
					"title": "مركز الفرقان التخصصي",
					"text": "بنية تحتية وكاميرات مراقبة وشبكات وبرمجيات مستشفى لمركز جراحي رائد في أسوان."
				}, {
					"icon": "🚑",
					"title": "مستشفى الهلال الأحمر",
					"text": "بنية تحتية كاملة — إدارة المرضى والشبكات وكاميرات المراقبة والرعاية الرقمية."
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "أبرز المشاريع",
				"title": "ما نفذناه",
				"cols": 3,
				"items": [
					{
						"icon": "🏥",
						"title": "إدارة المستشفيات",
						"text": "نظام Odoo كامل للمنشآت الطبية — المرضى والمواعيد والصيدلة والفوترة بسير عمل عربي."
					},
					{
						"icon": "📹",
						"title": "كاميرات المستشفى",
						"text": "نظام كاميرات IP بـ 128 قناة — غرف العمليات والممرات والصيدلة والمواقف، بمراقبة عن بُعد 24/7."
					},
					{
						"icon": "🧵",
						"title": "نقاط بيع المنسوجات",
						"text": "نقاط بيع Odoo 18 مخصصة بأوامر قص للغزل العربي — مخزون فوري ومزامنة متعددة الجلسات."
					},
					{
						"icon": "🧾",
						"title": "فاتورة ZATCA",
						"text": "توافق المرحلة الثانية للسعودية — فواتير XML ورموز QR وتقديم عبر API ومزامنة البوابة الحكومية."
					},
					{
						"icon": "🌾",
						"title": "إدارة المزارع",
						"text": "ERP زراعي لشركة قطف — مواقع المزرعة وتتبع المحاصيل وأوامر العمل وتحليلات الحصاد."
					},
					{
						"icon": "🔗",
						"title": "شبكة طبية",
						"text": "شبكة LAN/WLAN كاملة لحرم مستشفى — تمديدات منظمة وVLANs وغرفة سيرفرات وروابط احتياطية."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "المصادر المفتوحة و GitHub",
				"title": "+180 مستودعاً عاماً",
				"sub": "كود حقيقي وليس تسويقاً فقط. تحقق من مهاراتنا على GitHub.",
				"cols": 3,
				"items": [
					{
						"icon": "🐍",
						"title": "hospital-flask",
						"text": "نظام إدارة مستشفى بـ Python Flask — تسجيل المرضى والمواعيد والصيدلة والفوترة."
					},
					{
						"icon": "⚙️",
						"title": "mvc-erp",
						"text": "نظام ERP كامل بـ ASP.NET MVC و C# — المحاسبة والمخزون والموارد البشرية عبر تقنيات متعددة."
					},
					{
						"icon": "🧰",
						"title": "Openrol",
						"text": "تطبيق أعمال مفتوح المصدر بلغة Python يعكس خبرتنا في منظومة بايثون."
					},
					{
						"icon": "🗺️",
						"title": "web_google_maps_drawing",
						"text": "موديول Odoo لتكامل خرائط Google بأدوات الرسم — تحديد الموقع والخرائط المتعددة الأضلاع وتخطيط المسارات."
					},
					{
						"icon": "🛒",
						"title": "متجر Odoo",
						"text": "موديولات نقاط بيع منشورة — أوامر القص والمخزون الفوري وإطار الأمان، مُختبرة في الإنتاج."
					},
					{
						"icon": "🌟",
						"title": "+180 مستودعاً",
						"text": "أندرويد وiOS وPython وJavaScript وC# وJava وVR — ملف مطوّر متكامل مع منحة Google Challenge."
					}
				],
				"cta": {
					"label": "زيارة ملف GitHub",
					"href": "https://github.com/XamlTech"
				}
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "هل تريد أن تكون قصة نجاحنا القادمة؟",
				"text": "أخبرنا بأهدافك ونقترح لك النهج الأمثل.",
				"actions": [{
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var portfolio_default = {
	en: en$10,
	ar: ar$10
};
//#endregion
//#region src/content/data/pages/pricing.json
var pricing_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$9,
	default: () => pricing_default,
	en: () => en$9
});
var en$9 = {
	"meta": {
		"title": "Transparent Pricing & ROI Calculator — XamlTech Solutions",
		"description": "Odoo POS and AI addon pricing — one-time purchase, lifetime updates and 24/7 support. Calculate your ROI in minutes.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "💰 Transparent Pricing",
				"title": "Simple, scalable pricing",
				"lead": "No hidden fees, no surprise charges. One-time purchase — lifetime use. Calculate your ROI in minutes.",
				"ctas": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Visit the addons store",
					"href": "/addons-store?lang=en",
					"style": "ghost",
					"big": true
				}]
			}
		},
		{
			"partial": "pricing",
			"data": {
				"eyebrow": "Odoo POS addons",
				"title": "Choose your solution",
				"sub": "All modules include lifetime updates and 24/7 support email.",
				"cols": 3,
				"items": [
					{
						"title": "POS Security Framework",
						"text": "Manager PIN, discount limits, role-based access.",
						"currency": "EUR",
						"price": "€89",
						"features": [
							"Odoo 18/19 compatible",
							"12,924 lines of code",
							"80+ Python & JS tests",
							"Lifetime updates"
						],
						"cta": {
							"label": "Buy on Odoo Apps",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					},
					{
						"tag": "Best value",
						"featured": true,
						"title": "POS Customization Kit",
						"text": "Theme builder, dark mode, custom layouts.",
						"currency": "EUR",
						"price": "€79",
						"features": [
							"Odoo 18/19 compatible",
							"43,297 lines of code",
							"50+ Python & JS tests",
							"CSS variables included"
						],
						"cta": {
							"label": "Buy on Odoo Apps",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					},
					{
						"title": "POS Cutting Orders",
						"text": "Fabric/textile, real-time tracking, cutting UI.",
						"currency": "EUR",
						"price": "€129",
						"features": [
							"Odoo 18/19 compatible",
							"39,729 lines of code",
							"165+ Python & JS tests",
							"Textile-specific features"
						],
						"cta": {
							"label": "Buy on Odoo Apps",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					}
				]
			}
		},
		{
			"partial": "pricing",
			"data": {
				"alt": true,
				"eyebrow": "AI-powered solutions",
				"title": "AI Copilots (Odoo 19)",
				"cols": 2,
				"items": [{
					"title": "AI Marketing Copilot",
					"text": "Budget optimisation, campaign scoring, real-time analytics.",
					"currency": "EUR",
					"price": "€69",
					"features": [
						"Odoo 19 native",
						"8,410 lines of code",
						"Thompson Sampling algorithm",
						"Dashboard analytics"
					],
					"cta": {
						"label": "Buy on Odoo Apps",
						"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
					}
				}, {
					"title": "AI POS Copilot",
					"text": "Real-time upsell, smart suggestions, Arabic-first UX.",
					"currency": "EUR",
					"price": "€49",
					"features": [
						"Odoo 19 native",
						"2,623 lines of code",
						"Multi-LLM support",
						"Graceful degradation"
					],
					"cta": {
						"label": "Buy on Odoo Apps",
						"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
					}
				}]
			}
		},
		{
			"partial": "roi",
			"data": {
				"eyebrow": "ROI calculator",
				"title": "Calculate your return on investment",
				"sub": "Estimate your payback period and first-year return with any XamlTech solution.",
				"currency": "EUR",
				"monthlyLabel": "Estimated monthly savings",
				"investmentLabel": "One-time investment",
				"monthlyDefault": 500,
				"investment": 129,
				"savingsLabel": "Monthly savings",
				"paybackLabel": "Payback period",
				"annualLabel": "Annual ROI (year 1)"
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to invest in your growth?",
				"text": "All modules include lifetime updates and priority support. No long-term contracts.",
				"actions": [{
					"label": "Visit the addons store",
					"href": "/addons-store?lang=en",
					"style": "primary"
				}, {
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$9 = {
	"meta": {
		"title": "تسعير شفاف وحاسبة العائد على الاستثمار — حلول زامل تك",
		"description": "أسعار إضافات Odoo لنقاط البيع والذكاء الاصطناعي — دفعة واحدة، تحديثات مدى الحياة ودعم 24/7. احسب عائدك في دقائق.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "💰 تسعير شفاف",
				"title": "تسعير بسيط وقابل للتوسع",
				"lead": "بدون رسوم خفية أو مفاجآت. دفعة واحدة — استخدام مدى الحياة. احسب عائدك في دقائق.",
				"ctas": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary",
					"big": true
				}, {
					"label": "تصفّح متجر الإضافات",
					"href": "/addons-store",
					"style": "ghost",
					"big": true
				}]
			}
		},
		{
			"partial": "pricing",
			"data": {
				"eyebrow": "إضافات نقاط البيع",
				"title": "اختر الحل المناسب",
				"sub": "كل الموديولات تشمل تحديثات مدى الحياة ودعم بريد 24/7.",
				"cols": 3,
				"items": [
					{
						"title": "POS Security Framework",
						"text": "رقم PIN للمدير، حدود الخصم، صلاحيات حسب الدور.",
						"currency": "EUR",
						"price": "€89",
						"features": [
							"متوافق مع Odoo 18/19",
							"12,924 سطر كود",
							"+80 اختبار Python و JS",
							"تحديثات مدى الحياة"
						],
						"cta": {
							"label": "اشترِ من Odoo",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					},
					{
						"tag": "أفضل قيمة",
						"featured": true,
						"title": "POS Customization Kit",
						"text": "بانٍ للثيمات، الوضع الداكن، تخطيطات مخصصة.",
						"currency": "EUR",
						"price": "€79",
						"features": [
							"متوافق مع Odoo 18/19",
							"43,297 سطر كود",
							"+50 اختبار Python و JS",
							"متغيرات CSS مدمجة"
						],
						"cta": {
							"label": "اشترِ من Odoo",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					},
					{
						"title": "POS Cutting Orders",
						"text": "أقمشة ومنسوجات، تتبع فوري، واجهة قص.",
						"currency": "EUR",
						"price": "€129",
						"features": [
							"متوافق مع Odoo 18/19",
							"39,729 سطر كود",
							"+165 اختبار Python و JS",
							"ميزات خاصة بالمنسوجات"
						],
						"cta": {
							"label": "اشترِ من Odoo",
							"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
						}
					}
				]
			}
		},
		{
			"partial": "pricing",
			"data": {
				"alt": true,
				"eyebrow": "حلول الذكاء الاصطناعي",
				"title": "مساعدو الذكاء الاصطناعي (Odoo 19)",
				"cols": 2,
				"items": [{
					"title": "AI Marketing Copilot",
					"text": "تحسين الميزانية، تصنيف الحملات، تحليلات فورية.",
					"currency": "EUR",
					"price": "€69",
					"features": [
						"أصلي على Odoo 19",
						"8,410 سطر كود",
						"خوارزمية Thompson Sampling",
						"تحليلات لوحة المعلومات"
					],
					"cta": {
						"label": "اشترِ من Odoo",
						"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
					}
				}, {
					"title": "AI POS Copilot",
					"text": "بيع إضافي فوري، اقتراحات ذكية، تجربة عربية أولاً.",
					"currency": "EUR",
					"price": "€49",
					"features": [
						"أصلي على Odoo 19",
						"2,623 سطر كود",
						"دعم عدة نماذج LLM",
						"تدرّج آمن عند التعطل"
					],
					"cta": {
						"label": "اشترِ من Odoo",
						"href": "https://apps.odoo.com/apps/modules/browse?search=xamltech"
					}
				}]
			}
		},
		{
			"partial": "roi",
			"data": {
				"eyebrow": "حاسبة العائد",
				"title": "احسب عائد استثمارك",
				"sub": "قدّر فترة الاسترداد والعائد للسنة الأولى مع أي حل من زامل تك.",
				"currency": "EUR",
				"monthlyLabel": "التوفير الشهري المتوقع",
				"investmentLabel": "الاستثمار لمرة واحدة",
				"monthlyDefault": 500,
				"investment": 129,
				"savingsLabel": "التوفير الشهري",
				"paybackLabel": "فترة الاسترداد",
				"annualLabel": "عائد السنة الأولى"
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "مستعد للاستثمار في نموك؟",
				"text": "كل الموديولات تشمل تحديثات مدى الحياة ودعماً مميزاً. بلا عقود طويلة.",
				"actions": [{
					"label": "تصفّح متجر الإضافات",
					"href": "/addons-store",
					"style": "primary"
				}, {
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "wa"
				}]
			}
		}
	]
};
var pricing_default = {
	en: en$9,
	ar: ar$9
};
//#endregion
//#region src/content/data/pages/privacy-policy.json
var privacy_policy_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$8,
	default: () => privacy_policy_default,
	en: () => en$8
});
var en$8 = {
	"meta": {
		"title": "Privacy Policy — XamlTech",
		"description": "How XamlTech (Aswan, Egypt) collects, uses, shares and protects your personal data across our website, ERP systems, AI assistants, addons store and Android apps.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [{
		"partial": "hero",
		"data": {
			"eyebrow": "🔒 Privacy & Security",
			"title": "Privacy Policy",
			"lead": "At XamlTech we are committed to protecting your privacy and personal data. This page explains clearly and simply how your data is collected, used and protected across all our services."
		}
	}, {
		"partial": "richtext",
		"data": { "html": "<h2>1. Introduction and scope</h2><p>This Privacy Policy applies to all digital services provided by XamlTech, based in Aswan, Arab Republic of Egypt, including:</p><ul><li>The website xamltech.com and all of its pages.</li><li>ERP, HR, payroll, medical and accounting systems.</li><li>AI assistants, AI agents and semantic search (RAG).</li><li>The addons store and Android (APK) download requests.</li><li>Android applications published by the company, such as Network Copilot.</li><li>Contact forms, quote and demo requests, and WhatsApp correspondence.</li></ul><p>By using any of these services you agree to this policy. If you do not agree, please do not use the service.</p><h2>2. Data we collect</h2><p><strong>Data you provide directly:</strong></p><ul><li>Name, email and phone number when you submit a contact form or request a demo.</li><li>Your name, email, device details and request reason when submitting an APK download request.</li><li>Account and profile data when you use our internal systems.</li></ul><p><strong>Data we collect automatically:</strong></p><ul><li>Usage data: pages you visit, visit duration, links you click.</li><li>Technical data: browser type, operating system, device type, approximate IP address.</li><li>Security logs: login attempts and access points used to protect the systems.</li></ul><h2>3. Sensitive data</h2><p>We do not collect sensitive data (such as health, religious, or other special-category data) unless you are a government organisation or an entity that has engaged us for such data.</p><h2>4. Sources of data</h2><ul><li>Directly from you: through forms, correspondence and requests.</li><li>Automatically: via analytics and logging when you use the website.</li><li>From third parties: where legally required (e.g. authentication services) and with your consent or legal authorisation.</li></ul><h2>5. How we use your data</h2><ul><li>To provide the services you requested and perform our contracts with you.</li><li>To process app-download requests and enable secure download links.</li><li>To improve and personalise the experience and content.</li><li>To contact you about your requests, our offers and support.</li><li>To protect the systems and detect fraudulent or unauthorised activity.</li><li>To comply with Egyptian legal and accounting obligations.</li></ul><h2>6. Legal basis for processing</h2><p>We process your data on the following bases, in line with Egypt's Personal Data Protection Law No. 151 of 2020 and its executive regulations:</p><ul><li>Consent: when you voluntarily provide your data through forms and requests.</li><li>Performance of a contract: to provide the services and products you engage us for.</li><li>Legal obligation: to comply with tax, accounting and governmental legislation.</li><li>Legitimate interest: to improve services and protect system security, balancing our interest against your rights.</li></ul><h2>7. Cookies</h2><p>We use cookies and similar technologies to:</p><ul><li>Operate the website (login session, form protection).</li><li>Store your preferences (dark/light mode, language).</li><li>Measure performance and analytics in aggregate, without identifying individuals.</li></ul><p>You can manage or delete cookies from your browser settings at any time; some settings may affect website functionality.</p><h2>8. Data sharing and disclosure</h2><p>We do not sell or rent your personal data to any party. We may share your data only in the following cases:</p><ul><li>Technical service providers: hosting, analytics, mailing, payment gateways — under contracts that ensure confidentiality.</li><li>AI providers: only when you use the AI assistant and explicitly invoke an external provider, and under that provider's terms.</li><li>Government bodies: when Egyptian law or a court order requires it.</li></ul><h2>9. Storage and security</h2><ul><li>Data is encrypted in transit using HTTPS and TLS.</li><li>Sensitive data and secrets (such as download-link keys) are stored encrypted or as secure derivatives (HMAC).</li><li>App download links are temporary, expiring (30 days) and unpredictable.</li><li>Access rights follow least privilege, with immutable audit logs.</li><li>Periodic backups with tested recovery policies.</li></ul><p>Despite all measures, absolute security of any internet transmission cannot be guaranteed; we undertake to report any breach promptly and to the extent required by law.</p><h2>10. Data retention</h2><ul><li>Accounting data and contracts: as required by Egyptian law.</li><li>Download requests and chat logs: deleted once the purpose ends or upon your request.</li><li>Aggregated analytics data: for a period not exceeding one year.</li></ul><p>When the purpose ends, we delete the data or make it non-identifiable.</p><h2>11. Your rights</h2><p>Under Egypt's Personal Data Protection Law No. 151 of 2020, you have the right to:</p><ul><li>Access your data and verify its accuracy.</li><li>Request correction or updating of your data.</li><li>Request suspension, restriction or deletion of processing where legally provided.</li><li>Withdraw your consent at any time without affecting the lawfulness of prior processing.</li><li>Object to processing for marketing purposes.</li><li>File a complaint with Egypt's National Personal Data Protection Centre.</li></ul><p>You can exercise any of these rights by emailing info@xamltech.com; we will respond within no more than 30 days.</p><h2>12. Android apps and APK download requests</h2><ul><li>Network Copilot: network diagnostics (IP, MAC, SSID, DNS, ping, scanning) are performed entirely on your device and do not leave it unless you export or share results yourself.</li><li>App downloads (APK): when you request a download, we collect your name and email (optionally your phone, Android version and request context) to issue a secure, expiring link.</li><li>Beta builds upload anonymous crash reports through tools (Firebase) to help stability, without images or personal content.</li></ul><h2>13. AI and semantic search</h2><ul><li>Your query is processed within our systems to answer your question and is not used to build an independent profile of you.</li><li>If you choose an external (cloud) AI provider, a brief prompt is sent to that provider under its terms; you can choose a local provider (Ollama) to keep data inside your environment.</li><li>We keep a conversation log to power the multi-turn \"memory\" feature; you can delete it from the assistant UI at any time.</li><li>We do not use your questions to train external models without explicit consent.</li></ul><h2>14. Children</h2><p>Our services are intended for organisations, companies and professionals, and we do not knowingly collect data from persons under 18. If we learn we have collected such data, we delete it immediately.</p><h2>15. Changes to this policy</h2><p>We may update this policy from time to time to keep pace with service developments or legal requirements. On a material change we update the \"last updated\" date and announce the change on this page.</p><h2>16. Contact us</h2><p>For any enquiry, to exercise your rights or to file a complaint:</p><ul><li>Email: info@xamltech.com</li><li>Location: Aswan, Arab Republic of Egypt</li></ul><h2>Glossary</h2><ul><li>Personal data: any data that identifies you directly or indirectly.</li><li>Processing: any operation on data, from collection and storage to use, sharing and deletion.</li><li>HMAC: a cryptographic signing technique that ensures the integrity of a download link and prevents tampering.</li><li>National Personal Data Protection Centre: the Egyptian regulator tasked with enforcing Law No. 151 of 2020.</li></ul>" }
	}]
};
var ar$8 = /*#__PURE__*/ JSON.parse("{\"meta\":{\"title\":\"سياسة الخصوصية — زامل تك\",\"description\":\"كيف تجمع زامل تك (أسوان، مصر) بياناتك الشخصية وتستخدمها وتشاركها وتحميها عبر موقعنا ومنظومات ERP والمساعد الذكي ومتجر الإضافات وتطبيقات أندرويد.\",\"og_image\":\"https://xamltech.com/static/img/pos-light.png\",\"index\":true},\"sections\":[{\"partial\":\"hero\",\"data\":{\"eyebrow\":\"🔒 الخصوصية والأمان\",\"title\":\"سياسة الخصوصية\",\"lead\":\"نلتزم في زامل تك بحماية خصوصيتك وبياناتك الشخصية، وتوفّر لك هذه الصفحة شرحاً واضحاً وبسيطاً لكيفية جمع بياناتك واستخدامها وحمايتها عبر جميع خدماتنا.\"}},{\"partial\":\"richtext\",\"data\":{\"html\":\"<h2>1. المقدمة ومجال التطبيق</h2><p>تنطبق سياسة الخصوصية هذه على جميع الخدمات الرقمية التي تقدّمها شركة زامل تك ومقرها أسوان، جمهورية مصر العربية، وتشمل:</p><ul><li>الموقع الإلكتروني xamltech.com وجميع صفحاته.</li><li>منظومات ERP والموارد البشرية والمرتبات والطبية والمحاسبية.</li><li>المساعد الذكي والوكلاء الأذكياء والبحث الدلالي (RAG).</li><li>متجر الإضافات وطلبات تحميل تطبيقات أندرويد (APK).</li><li>تطبيقات أندرويد التي تصدرها الشركة مثل Network Copilot.</li><li>نماذج التواصل وطلبات العروض وطلب التجربة والمراسلات عبر واتساب.</li></ul><p>باستخدامك أياً من هذه الخدمات فإنك توافق على ما ورد في هذه السياسة. إذا لم توافق، فيرجى عدم استخدام الخدمة.</p><h2>2. البيانات التي نجمعها</h2><p><strong>البيانات التي تقدّمها بنفسك:</strong></p><ul><li>الاسم والبريد الإلكتروني ورقم الهاتف عند إرسال نماذج التواصل أو طلب التجربة.</li><li>اسمك وبريدك وبيانات جهازك وسبب الطلب عند تقديم طلب تحميل تطبيق (APK).</li><li>بيانات الحسابات والملف الوظيفي عند التعامل مع منظوماتنا.</li></ul><p><strong>البيانات التي نجمعها تلقائياً:</strong></p><ul><li>بيانات الاستخدام: الصفحات التي تزورها ومدة الزيارة والروابط التي تنقرها.</li><li>بيانات تقنية: نوع المتصفح ونظام التشغيل ونوع الجهاز وعنوان IP التقريبي.</li><li>سجلات الأمان: محاولات الدخول ونقاط الوصول لحماية المنظومة.</li></ul><h2>3. البيانات الحساسة</h2><p>لا نجمع بيانات حساسة (كالبيانات الصحية أو الدينية أو الفئات الخاصة) إلا إذا كنت منظمة حكومية أو جهة أوكلتنا بمثل هذه البيانات.</p><h2>4. مصادر جمع البيانات</h2><ul><li>منك مباشرة: عبر النماذج والمراسلات والطلبات.</li><li>تلقائياً: من خلال التحليلات والسجلات عند استخدامك الموقع.</li><li>من أطراف ثالثة: في الحالات القانونية (مثل خدمات المصادقة) وبموافقتك أو بإذن قانوني.</li></ul><h2>5. كيف نستخدم بياناتك</h2><ul><li>تقديم الخدمات التي طلبتها وتنفيذ العقود معك.</li><li>معالجة طلبات تحميل التطبيقات وتمكين رابط التحميل المؤمَّن.</li><li>تحسين تجربة الاستخدام والمحتوى وتخصيصه.</li><li>التواصل معك بشأن طلباتك وعروضنا وخدمات الدعم.</li><li>حماية المنظومة وكشف الأنشطة الاحتيالية أو غير المسموح بها.</li><li>الامتثال للالتزامات القانونية والمحاسبية المصرية.</li></ul><h2>6. الأساس القانوني للمعالجة</h2><p>نعالج بياناتك وفق الأسس التالية، بما يتوافق مع قانون حماية البيانات الشخصية المصري رقم 151 لسنة 2020 ولائحته التنفيذية:</p><ul><li>الموافقة: عندما تقدّم بياناتك طوعاً عبر النماذج والطلبات.</li><li>تنفيذ العقد: لتقديم الخدمات والمنتجات التي تتعاقد عليها.</li><li>الالتزام القانوني: للامتثال للتشريعات الضريبية والمحاسبية والحكومية.</li><li>المصلحة المشروعة: لتحسين الخدمات وحماية أمان الأنظمة، مع الموازنة بين مصلحتنا وحقوقك.</li></ul><h2>7. ملفات تعريف الارتباط (Cookies)</h2><p>نستخدم ملفات تعريف الارتباط والتقنيات المشابهة من أجل:</p><ul><li>التشغيل الأساسي للموقع (جلسة تسجيل الدخول، حماية النماذج).</li><li>حفظ تفضيلاتك (الوضع الليلي/النهاري، اللغة).</li><li>قياس الأداء والتحليلات بشكل مجمَّع لا يُعرِّف الأفراد.</li></ul><p>يمكنك إدارة أو حذف ملفات تعريف الارتباط من إعدادات متصفحك في أي وقت، وقد تؤثر بعض الإعدادات على وظائف الموقع.</p><h2>8. مشاركة البيانات والإفصاح</h2><p>لا نبيع بياناتك الشخصية ولا نؤجرها لأي طرف. قد نشارك بياناتك في الحالات التالية فقط:</p><ul><li>مقدّمو الخدمات التقنية: استضافة وتحليلات ومراسلات وبوابات دفع — بعقود تضمن سرية البيانات.</li><li>مزوّدو الذكاء الاصطناعي: فقط عند استخدامك للمساعد الذكي واستدعائك صراحةً لمزود خارجي، ووفق شروط ذلك المزود.</li><li>الجهات الحكومية: عندما يتطلب القانون المصري أو أمر قضائي ذلك.</li></ul><h2>9. التخزين والأمن</h2><ul><li>تُشفَّر البيانات أثناء النقل باستخدام HTTPS وTLS.</li><li>تُخزَّن البيانات الحساسة والأسرار (مثل مفاتيح روابط التحميل) مشفّرة أو كمشتقات آمنة (HMAC).</li><li>روابط تحميل التطبيقات مؤقتة ومنتهية الصلاحية (30 يوماً) ولا يمكن التنبؤ بها.</li><li>تُبنى صلاحيات الوصول على مبدأ «أقل امتياز» مع سجلات تدقيق ثابتة.</li><li>نسخ احتياطي دوري مع سياسات استعادة مجرّبة.</li></ul><p>رغم كل التدابير، لا يمكن ضمان أمان مطلق لأي نقل عبر الإنترنت، ونتعهد بالتبليغ الفوري عن أي خرق في حدود القانون.</p><h2>10. فترة الاحتفاظ بالبيانات</h2><ul><li>البيانات المحاسبية والعقود: بحسب المتطلبات القانونية المصرية.</li><li>طلبات التحميل وسجلات الدردشات: تُحذف بعد انتهاء الغرض منها أو عند طلبك.</li><li>بيانات التحليلات المجمعة: لفترة لا تتجاوز سنة.</li></ul><p>عند انتهاء الغرض، نحذف البيانات أو نجعلها غير قابلة للتعريف.</p><h2>11. حقوقك</h2><p>وفق قانون حماية البيانات الشخصية المصري رقم 151 لسنة 2020، يحق لك:</p><ul><li>الحق في الاطلاع على بياناتك والتحقق من صحتها.</li><li>الحق في طلب تصحيح أو تحديث بياناتك.</li><li>الحق في طلب إيقاف معالجة بياناتك أو تقييدها أو حذفها في الحالات المقررة قانوناً.</li><li>الحق في سحب موافقتك في أي وقت دون مساس بمشروعية المعالجة السابقة.</li><li>الحق في الاعتراض على المعالجة للأغراض التسويقية.</li><li>الحق في تقديم شكوى إلى الجهاز القومي لحماية البيانات الشخصية في مصر.</li></ul><p>يمكنك ممارسة أي من هذه الحقوق عبر البريد info@xamltech.com وسنرد خلال مدة لا تتجاوز 30 يوماً.</p><h2>12. تطبيقات أندرويد وطلبات تحميل APK</h2><ul><li>Network Copilot: تُنجز أدوات التشخيص الشبكي (IP وMAC وSSID وDNS وبينغ والفحص) على جهازك بالكامل ولا تغادره إلا إذا صدّرت أو شاركت النتائج بنفسك.</li><li>تحميل التطبيقات (APK): عند طلب تحميل تطبيق، نجمع اسمك وبريدك الإلكتروني (وبشكل اختياري هاتفك وإصدار أندرويد وسياق الطلب) لإصدار رابط آمن منتهي الصلاحية.</li><li>النسخ التجريبية (beta) ترفع تقارير أعطال مجهولة المصدر عبر أدوات (Firebase) للمساعدة في الاستقرار دون صور أو محتوى شخصي.</li></ul><h2>13. الذكاء الاصطناعي والبحث الدلالي</h2><ul><li>يُعالج استعلامك داخل منظوماتنا للرد على سؤالك ولا يُستخدم لتشكيل ملف تعريف مستقل عنك.</li><li>عند اختيارك مزود ذكاء اصطناعي خارجي (سحابي)، يُرسل استعلام مختصر لذلك المزود وفق شروطه، ويمكنك اختيار مزود محلي (Ollama) لتبقى البيانات داخل بيئتك.</li><li>نحتفظ بسجل المحادثات لتشغيل خاصية «التذكر متعدد الدورات»، ويمكنك حذفه من واجهة المساعد في أي وقت.</li><li>لا نستخدم أسئلتك لتدريب نماذج أطراف خارجية دون موافقة صريحة.</li></ul><h2>14. الأطفال</h2><p>خدماتنا موجّهة للجهات والشركات والمهنيين، ولا نجمع عمداً بيانات أشخاص تقل أعمارهم عن 18 عاماً. إذا علمنا بجمعنا هذه البيانات، نحذفها فوراً.</p><h2>15. التعديلات على السياسة</h2><p>قد نحدّث هذه السياسة من وقت لآخر لمواكبة تطور الخدمات أو المتطلبات القانونية. عند إجراء تعديل جوهري، نحدّث تاريخ «آخر تحديث» ونعلن عن التغيير في هذه الصفحة.</p><h2>16. تواصل معنا</h2><p>لأي استفسار أو ممارسة حقوقك أو تقديم شكوى:</p><ul><li>البريد الإلكتروني: info@xamltech.com</li><li>الموقع: أسوان — الجمهورية العربية المصرية</li></ul><h2>مسرد المصطلحات</h2><ul><li>البيانات الشخصية: أي بيانات تحدّد هويتك بشكل مباشر أو غير مباشر.</li><li>المعالجة: أي عملية على البيانات من جمع وتخزين واستخدام ومشاركة وحذف.</li><li>HMAC: تقنية توقيع تشفيري تضمن سلامة رابط التحميل وعدم العبث به.</li><li>الجهاز القومي لحماية البيانات الشخصية: الجهة الرقابية المصرية المنوط بها تطبيق القانون رقم 151 لسنة 2020.</li></ul>\"}}]}");
var privacy_policy_default = {
	en: en$8,
	ar: ar$8
};
//#endregion
//#region src/content/data/pages/request-demo.json
var request_demo_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$7,
	default: () => request_demo_default,
	en: () => en$7
});
var en$7 = {
	"meta": {
		"title": "Try Odoo POS Live Demo Free — XamlTech Cutting Orders & Stock Modules",
		"description": "Enter your details and get instant access to the live Odoo 19 POS demo with cutting orders, real-time stock and the security framework.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"hero": {
		"eyebrow": "🖥️ Live Odoo 19 Demo",
		"title": "Try our POS addons live",
		"lead": "Enter your details and get access to the live demo environment. Explore cutting orders, real-time stock, the security framework and the AI copilot.",
		"points": [
			"Instant access after review",
			"No setup required",
			"Demo data included"
		],
		"ctas": [{
			"label": "Chat on WhatsApp",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "Demo access",
	"formSub": "Fill in your details and we'll send you the demo credentials.",
	"submit": "Request free demo",
	"privacyLabel": "Privacy Policy",
	"products": [
		"POS Cutting Orders",
		"POS Stock Real-Time",
		"POS Security Framework",
		"POS Customization Kit",
		"AI POS Copilot",
		"AI Marketing Copilot",
		"Full Odoo ERP",
		"IT Infrastructure / CCTV"
	],
	"cta": {
		"title": "Want to see it with your own data?",
		"text": "We can run a guided walkthrough focused on your business type.",
		"actions": [{
			"label": "Contact us",
			"href": "/contact-us?lang=en",
			"style": "primary"
		}, {
			"label": "WhatsApp us",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var ar$7 = {
	"meta": {
		"title": "جرّب عرض Odoo POS المباشر مجاناً — أوامر القص والمخزون من زامل تك",
		"description": "أدخل بياناتك واحصل على وصول فوري لعرض Odoo 19 المباشر مع أوامر القص والمخزون الفوري وإطار الأمان.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"hero": {
		"eyebrow": "🖥️ عرض Odoo 19 مباشر",
		"title": "جرّب إضافات نقاط البيع مباشرة",
		"lead": "أدخل بياناتك واحصل على وصول لبيئة العرض المباشر. استكشف أوامر القص والمخزون الفوري وإطار الأمان ومساعد الذكاء الاصطناعي.",
		"points": [
			"وصول فوري بعد المراجعة",
			"بلا إعداد مطلوب",
			"بيانات تجريبية مدمجة"
		],
		"ctas": [{
			"label": "محادثة واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa",
			"big": true,
			"external": true
		}]
	},
	"formTitle": "وصول العرض",
	"formSub": "املأ بياناتك وسنرسل لك بيانات دخول العرض.",
	"submit": "اطلب تجربة مجانية",
	"privacyLabel": "سياسة الخصوصية",
	"products": [
		"أوامر القص",
		"مخزون نقاط البيع الفوري",
		"إطار أمان نقاط البيع",
		"عدة تخصيص نقاط البيع",
		"مساعد نقاط البيع الذكي",
		"مساعد التسويق الذكي",
		"نظام Odoo ERP كامل",
		"البنية التحتية / الكاميرات"
	],
	"cta": {
		"title": "تريد رؤيته ببياناتك؟",
		"text": "يمكننا تنفيذ جلسة موجّهة مخصصة لنوع نشاطك.",
		"actions": [{
			"label": "تواصل معنا",
			"href": "/contact-us",
			"style": "primary"
		}, {
			"label": "واتساب",
			"href": "https://wa.me/201007982468",
			"style": "wa"
		}]
	}
};
var request_demo_default = {
	en: en$7,
	ar: ar$7
};
//#endregion
//#region src/content/data/pages/solutions-erp.json
var solutions_erp_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$6,
	default: () => solutions_erp_default,
	en: () => en$6
});
var en$6 = {
	"meta": {
		"title": "Odoo ERP Development & Custom Module Implementation in Egypt — XamlTech",
		"description": "XamlTech implements Odoo ERP for sales, accounting, inventory and HR — deployment, customisation, integration, BI, training and 24/7 support from Aswan, Egypt.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "⚙️ ERP Solutions",
				"title": "An integrated ERP system for your business",
				"lead": "Are your operations disorganised? We implement Odoo ERP to manage everything — sales, accounting, inventory and HR — in one unified system.",
				"points": [
					"One source of truth for every department",
					"Automated workflows and financial reporting",
					"On-premise or cloud, Arabic-first"
				],
				"ctas": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Talk to us on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}],
				"stats": [
					{
						"v": "8+",
						"l": "Years of Odoo development"
					},
					{
						"v": "40+",
						"l": "ERP deployments"
					},
					{
						"v": "16–19",
						"l": "Odoo versions"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "ERP services",
				"title": "What we deliver",
				"sub": "From analysis to go-live and long-term support.",
				"cols": 3,
				"items": [
					{
						"icon": "🚀",
						"title": "Implementation",
						"text": "Full Odoo deployment — requirements analysis, configuration, data migration, testing and go-live support."
					},
					{
						"icon": "🧱",
						"title": "Customisation",
						"text": "Custom module development, workflow automation, report builder and UI/UX tailoring to match your processes."
					},
					{
						"icon": "🔗",
						"title": "Integration",
						"text": "Third-party integration — payment gateways, shipping, ZATCA e-invoicing, APIs and data sync."
					},
					{
						"icon": "📊",
						"title": "Business intelligence",
						"text": "Custom dashboards, KPI tracking, financial reports and data analytics for informed decisions."
					},
					{
						"icon": "🎓",
						"title": "Training",
						"text": "Comprehensive user and admin training, video tutorials and documentation in English and Arabic."
					},
					{
						"icon": "🛠️",
						"title": "Technical support",
						"text": "24/7 support, bug fixes, performance optimisation, security patches and version upgrades."
					}
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"alt": true,
				"eyebrow": "📋 Methodology",
				"title": "Implementation process",
				"sub": "A structured path from first workshop to steady state.",
				"items": [
					{
						"title": "Requirements analysis",
						"text": "We analyse your business needs and define the optimal strategy."
					},
					{
						"title": "Design",
						"text": "Module and feature configuration tailored to your operations."
					},
					{
						"title": "Development",
						"text": "Custom modules and integration with external systems."
					},
					{
						"title": "UAT",
						"text": "User acceptance testing and performance validation."
					},
					{
						"title": "Training & support",
						"text": "Comprehensive training and ongoing support."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "🏭 Industries",
				"title": "Industries we serve",
				"sub": "Vertical know-how baked into every deployment.",
				"cols": 4,
				"items": [
					{
						"icon": "🛒",
						"title": "Retail & trade"
					},
					{
						"icon": "🍽️",
						"title": "Restaurants & cafés"
					},
					{
						"icon": "🏭",
						"title": "Manufacturing"
					},
					{
						"icon": "🌾",
						"title": "Agriculture"
					},
					{
						"icon": "💊",
						"title": "Pharmacy & healthcare"
					},
					{
						"icon": "🧵",
						"title": "Textile & fashion"
					},
					{
						"icon": "🏛️",
						"title": "Government & venues"
					},
					{
						"icon": "📦",
						"title": "Distribution & logistics"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Pricing",
				"title": "Packages",
				"sub": "Flexible pricing for every business size.",
				"cols": 3,
				"items": [
					{
						"icon": "🟢",
						"title": "Starter",
						"text": "Basic Odoo setup, 3 modules, training, 3 months support. Perfect for small businesses starting their ERP journey."
					},
					{
						"icon": "🔵",
						"title": "Business",
						"text": "Full implementation, 8+ modules, customisation, integration, 6 months support. For growing companies."
					},
					{
						"icon": "🟣",
						"title": "Enterprise",
						"text": "Complete deployment, unlimited modules, custom development, multi-company, 12 months support. For large organisations."
					}
				],
				"cta": {
					"label": "Get a tailored quote",
					"href": "/contact-us?lang=en"
				}
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to start your ERP project?",
				"text": "Your trusted partner for digital transformation, ERP systems, AI solutions and IT infrastructure.",
				"actions": [{
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "primary"
				}, {
					"label": "Contact us",
					"href": "/contact-us?lang=en",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$6 = {
	"meta": {
		"title": "تطوير أنظمة Odoo ERP وتخصيص الموديولات في مصر — زامل تك",
		"description": "تنفذ زامل تك أنظمة Odoo ERP للمبيعات والمحاسبة والمخزون والموارد البشرية — تنفيذ وتخصيص وتكامل وذكاء أعمال وتدريب ودعم فني من أسوان، مصر.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "⚙️ حلول ERP",
				"title": "نظام ERP متكامل لأعمالك",
				"lead": "هل عملياتك غير منظمة؟ ننفذ نظام Odoo ERP لإدارة كل شيء — من المبيعات للمحاسبة، ومن المخزون للموارد البشرية — في نظام موحّد.",
				"points": [
					"مصدر واحد للحقيقة لكل الأقسام",
					"سير عمل آلي وتقارير مالية",
					"على الخوادم أو السحابة، وبالعربية أولاً"
				],
				"ctas": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary",
					"big": true
				}, {
					"label": "تواصل على واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}],
				"stats": [
					{
						"v": "+8",
						"l": "سنوات تطوير Odoo"
					},
					{
						"v": "+40",
						"l": "نظام ERP منفذ"
					},
					{
						"v": "16–19",
						"l": "إصدارات Odoo"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "خدمات ERP",
				"title": "ما نقدمه",
				"sub": "من التحليل حتى الإطلاق والدعم طويل الأمد.",
				"cols": 3,
				"items": [
					{
						"icon": "🚀",
						"title": "التنفيذ",
						"text": "نشر Odoo كاملاً — تحليل المتطلبات، التهيئة، ترحيل البيانات، الاختبار ودعم الإطلاق."
					},
					{
						"icon": "🧱",
						"title": "التخصيص",
						"text": "تطوير موديولات مخصصة، أتمتة سير العمل، بناء التقارير وتخصيص الواجهة بما يناسب عملياتك."
					},
					{
						"icon": "🔗",
						"title": "التكامل",
						"text": "التكامل مع الأنظمة الخارجية — بوابات الدفع، الشحن، فاتورة ZATCA الإلكترونية، واجهات API ومزامنة البيانات."
					},
					{
						"icon": "📊",
						"title": "ذكاء الأعمال",
						"text": "لوحات مخصصة، تتبع مؤشرات الأداء، تقارير مالية وتحليلات لاتخاذ قرارات مدروسة."
					},
					{
						"icon": "🎓",
						"title": "التدريب",
						"text": "تدريب شامل للمستخدمين والمسؤولين، مقاطع فيديو وتوثيق بالعربية والإنجليزية."
					},
					{
						"icon": "🛠️",
						"title": "الدعم الفني",
						"text": "دعم 24/7، إصلاح الأخطاء، تحسين الأداء، تحديثات الأمان وترقية الإصدارات."
					}
				]
			}
		},
		{
			"partial": "steps",
			"data": {
				"alt": true,
				"eyebrow": "📋 المنهجية",
				"title": "عملية التنفيذ",
				"sub": "مسار منظم من أول ورشة عمل حتى الاستقرار.",
				"items": [
					{
						"title": "تحليل المتطلبات",
						"text": "نحلل احتياجات عملك ونحدد الاستراتيجية المثلى."
					},
					{
						"title": "التصميم",
						"text": "تهيئة الموديولات والمميزات لتلائم عملياتك."
					},
					{
						"title": "التطوير",
						"text": "تطوير الموديولات والتكامل مع الأنظمة الخارجية."
					},
					{
						"title": "اختبار القبول",
						"text": "اختبار قبول المستخدم والتحقق من الأداء."
					},
					{
						"title": "التدريب والدعم",
						"text": "تدريب شامل ودعم مستمر."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "🏭 القطاعات",
				"title": "القطاعات التي نخدمها",
				"sub": "خبرة قطاعية مدمجة في كل تنفيذ.",
				"cols": 4,
				"items": [
					{
						"icon": "🛒",
						"title": "التجزئة والتجارة"
					},
					{
						"icon": "🍽️",
						"title": "المطاعم والكافيهات"
					},
					{
						"icon": "🏭",
						"title": "التصنيع"
					},
					{
						"icon": "🌾",
						"title": "الزراعة"
					},
					{
						"icon": "💊",
						"title": "الصيدلة والرعاية الصحية"
					},
					{
						"icon": "🧵",
						"title": "المنسوجات والأزياء"
					},
					{
						"icon": "🏛️",
						"title": "الجهات الحكومية والمتاحف"
					},
					{
						"icon": "📦",
						"title": "التوزيع والخدمات اللوجستية"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "الباقات",
				"title": "الباقات",
				"sub": "تسعير مرن لكل أحجام الأعمال.",
				"cols": 3,
				"items": [
					{
						"icon": "🟢",
						"title": "Starter",
						"text": "تهيئة Odoo أساسية، 3 موديولات، تدريب، ودعم 3 أشهر. مثالية للشركات الصغيرة التي تبدأ رحلتها مع ERP."
					},
					{
						"icon": "🔵",
						"title": "Business",
						"text": "تنفيذ كامل، +8 موديولات، تخصيص، تكامل، ودعم 6 أشهر. للشركات النامية."
					},
					{
						"icon": "🟣",
						"title": "Enterprise",
						"text": "نشر كامل، موديولات غير محدودة، تطوير مخصص، تعدد الشركات، ودعم 12 شهراً. للمؤسسات الكبيرة."
					}
				],
				"cta": {
					"label": "اطلب عرض سعر مخصص",
					"href": "/contact-us"
				}
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "هل أنت مستعد لبدء مشروع ERP؟",
				"text": "شريكك الموثوق للتحول الرقمي وأنظمة ERP وحلول الذكاء الاصطناعي والبنية التحتية.",
				"actions": [{
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "primary"
				}, {
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "wa"
				}]
			}
		}
	]
};
var solutions_erp_default = {
	en: en$6,
	ar: ar$6
};
//#endregion
//#region src/content/data/pages/solutions-it-infrastructure.json
var solutions_it_infrastructure_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$5,
	default: () => solutions_it_infrastructure_default,
	en: () => en$5
});
var en$5 = {
	"meta": {
		"title": "Network Infrastructure & CCTV Surveillance Installation in Aswan, Egypt — XamlTech",
		"description": "Design and implementation of LAN/WAN, enterprise Wi-Fi, firewalls, servers, virtualisation and IP CCTV systems across Aswan and Egypt. Free site survey.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🖧 IT Infrastructure",
				"title": "Secure & reliable infrastructure",
				"lead": "Slow network? Weak security? We design and implement professional network infrastructure, server systems and surveillance solutions for your business.",
				"points": [
					"Structured cabling, switching and routing",
					"Enterprise Wi-Fi and firewalls",
					"Servers, virtualisation and IP CCTV"
				],
				"ctas": [{
					"label": "Request a site survey",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Talk to us on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}],
				"stats": [
					{
						"v": "99.9%",
						"l": "Uptime target"
					},
					{
						"v": "128",
						"l": "Camera channels"
					},
					{
						"v": "24/7",
						"l": "Monitoring"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "What we build",
				"title": "Network & security solutions",
				"sub": "One partner for cabling, connectivity, servers and surveillance.",
				"cols": 3,
				"items": [
					{
						"icon": "🔌",
						"title": "LAN / WAN",
						"text": "Complete network design — structured cabling, VLANs, routing and switching for offices and data centres."
					},
					{
						"icon": "📶",
						"title": "Wireless",
						"text": "Enterprise Wi-Fi — seamless roaming, load balancing, captive portals and centralised management."
					},
					{
						"icon": "🛡️",
						"title": "Firewall & VPN",
						"text": "Advanced firewall, site-to-site VPN, remote access and intrusion detection/prevention."
					},
					{
						"icon": "🖥️",
						"title": "Servers",
						"text": "Linux/Windows servers, virtualisation (Proxmox, VMware), Docker and high-availability clustering."
					},
					{
						"icon": "📹",
						"title": "CCTV & surveillance",
						"text": "IP cameras (2–8MP), DVR/NVR, remote monitoring, mobile apps, up to 128 channels."
					},
					{
						"icon": "📈",
						"title": "Monitoring",
						"text": "24/7 network monitoring, uptime tracking, alerting, bandwidth analysis and performance optimisation."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "Why our infrastructure",
				"title": "Built to last",
				"cols": 4,
				"items": [
					{
						"icon": "🔐",
						"title": "Security",
						"text": "Enterprise-grade cybersecurity and data protection."
					},
					{
						"icon": "⚡",
						"title": "Speed",
						"text": "Optimised for maximum throughput and low latency."
					},
					{
						"icon": "🧱",
						"title": "Stability",
						"text": "Redundant systems with a 99.9% uptime target."
					},
					{
						"icon": "📈",
						"title": "Scalable",
						"text": "Grows with your business without major overhauls."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Need a quote for networks or cameras?",
				"text": "Free site surveys across Aswan and Egypt.",
				"actions": [{
					"label": "Request a site survey",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$5 = {
	"meta": {
		"title": "تصميم الشبكات وأنظمة كاميرات المراقبة في أسوان ومصر — زامل تك",
		"description": "تصميم وتنفيذ شبكات LAN/WAN والواي فاي المؤسسي والجدران النارية والسيرفرات والافتراضية وأنظمة كاميرات IP في أسوان ومصر. زيارة موقع مجانية.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🖧 البنية التحتية",
				"title": "بنية تحتية آمنة وموثوقة",
				"lead": "شبكة بطيئة؟ أمن ضعيف؟ نصمم وننفذ بنية تحتية احترافية للشبكات وأنظمة السيرفرات وحلول المراقبة لعملك.",
				"points": [
					"تمديدات منظمة وسويتشات وراوتر",
					"واي فاي مؤسسي وجدران نارية",
					"سيرفرات وافتراضية وكاميرات IP"
				],
				"ctas": [{
					"label": "اطلب زيارة موقع",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "تواصل على واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}],
				"stats": [
					{
						"v": "99.9%",
						"l": "هدف وقت التشغيل"
					},
					{
						"v": "128",
						"l": "قناة كاميرا"
					},
					{
						"v": "24/7",
						"l": "مراقبة"
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "ما نبنيه",
				"title": "حلول الشبكات والأمن",
				"sub": "شريك واحد للتمديدات والاتصال والسيرفرات والمراقبة.",
				"cols": 3,
				"items": [
					{
						"icon": "🔌",
						"title": "LAN / WAN",
						"text": "تصميم شبكة كامل — تمديدات منظمة، VLANs، راوتر وسويتش للمكاتب ومراكز البيانات."
					},
					{
						"icon": "📶",
						"title": "الشبكات اللاسلكية",
						"text": "واي فاي مؤسسي — تجوال سلس، موازنة حمل، بوابات دخول وإدارة مركزية."
					},
					{
						"icon": "🛡️",
						"title": "جدار الحماية و VPN",
						"text": "جدار حماية متقدم، VPN بين المواقع، وصول عن بُعد وكشف/منع الاختراق."
					},
					{
						"icon": "🖥️",
						"title": "السيرفرات",
						"text": "سيرفرات Linux/Windows، افتراضية (Proxmox، VMware)، Docker وتجميع عالي التوفر."
					},
					{
						"icon": "📹",
						"title": "كاميرات المراقبة والأمن",
						"text": "كاميرات IP (2–8 ميجا)، DVR/NVR، مراقبة عن بُعد، تطبيقات موبايل حتى 128 قناة."
					},
					{
						"icon": "📈",
						"title": "المراقبة",
						"text": "مراقبة شبكة 24/7، تتبع التشغيل، تنبيهات، تحليل النطاق وتحسين الأداء."
					}
				]
			}
		},
		{
			"partial": "features",
			"data": {
				"alt": true,
				"eyebrow": "لماذا بنيتنا التحتية",
				"title": "مبنية لتدوم",
				"cols": 4,
				"items": [
					{
						"icon": "🔐",
						"title": "الأمان",
						"text": "أمن سيبراني بمستوى المؤسسات وحماية البيانات."
					},
					{
						"icon": "⚡",
						"title": "السرعة",
						"text": "مُحسّنة لأقصى إنتاجية وأقل زمن استجابة."
					},
					{
						"icon": "🧱",
						"title": "الاستقرار",
						"text": "أنظمة احتياطية بهدف 99.9% وقت تشغيل."
					},
					{
						"icon": "📈",
						"title": "قابلية التوسع",
						"text": "تنمو مع عملك دون إعادة بناء كبيرة."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "تحتاج عرض سعر لشبكات أو كاميرات؟",
				"text": "زيارات موقع مجانية في أسوان ومصر.",
				"actions": [{
					"label": "اطلب زيارة موقع",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var solutions_it_infrastructure_default = {
	en: en$5,
	ar: ar$5
};
//#endregion
//#region src/content/data/pages/solutions-web.json
var solutions_web_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$4,
	default: () => solutions_web_default,
	en: () => en$4
});
var en$4 = {
	"meta": {
		"title": "Professional Web Development & Design Services in Egypt — XamlTech",
		"description": "Modern, SEO-optimised websites, e-commerce stores, UI/UX design and web apps built by XamlTech in Aswan, Egypt.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🌐 Web Development",
				"title": "A professional website that delivers results",
				"lead": "Your website isn't generating results? We build modern, SEO-optimised websites that convert visitors into customers.",
				"points": [
					"Responsive, fast and accessible",
					"SEO and schema markup baked in",
					"Arabic RTL and English"
				],
				"ctas": [{
					"label": "Request a quote",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Talk to us on WhatsApp",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "What we build",
				"title": "Web services",
				"sub": "From a company site to a full web platform.",
				"cols": 3,
				"items": [
					{
						"icon": "🏢",
						"title": "Company websites",
						"text": "Professional company websites with modern design, responsive layout and clear brand messaging."
					},
					{
						"icon": "🛒",
						"title": "E-commerce",
						"text": "Online stores with payment integration, inventory sync, product catalogues and order management."
					},
					{
						"icon": "🎨",
						"title": "UI/UX design",
						"text": "Beautiful, intuitive interfaces designed for engagement, accessibility and conversion."
					},
					{
						"icon": "⚙️",
						"title": "Web applications",
						"text": "Custom web apps, dashboards, portals and SaaS platforms built with modern frameworks."
					},
					{
						"icon": "🔍",
						"title": "SEO",
						"text": "Search engine optimisation, content strategy, meta tags, schema markup and performance tuning."
					},
					{
						"icon": "🔧",
						"title": "Maintenance",
						"text": "Ongoing updates, security patches, content management, hosting and performance monitoring."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready for a website that performs?",
				"text": "Tell us about your goals and we'll propose the right approach.",
				"actions": [{
					"label": "Request a quote",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$4 = {
	"meta": {
		"title": "خدمات تطوير وتصميم المواقع الاحترافية في مصر — زامل تك",
		"description": "مواقع حديثة محسّنة لمحركات البحث ومتاجر إلكترونية وتصميم واجهات وتطبيقات ويب من زامل تك في أسوان، مصر.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🌐 تطوير الويب",
				"title": "موقع احترافي يحقق نتائج",
				"lead": "موقعك لا يحقق نتائج؟ نبني مواقع حديثة محسّنة لمحركات البحث تحوّل الزوار إلى عملاء.",
				"points": [
					"متجاوب وسريع ومتاح للجميع",
					"SEO وبيانات منظمة مدمجة",
					"عربي RTL وإنجليزي"
				],
				"ctas": [{
					"label": "اطلب عرض سعر",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "تواصل على واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa",
					"big": true,
					"external": true
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "ما نبنيه",
				"title": "خدمات الويب",
				"sub": "من موقع شركة إلى منصة ويب متكاملة.",
				"cols": 3,
				"items": [
					{
						"icon": "🏢",
						"title": "مواقع الشركات",
						"text": "مواقع شركات احترافية بتصميم حديث وتخطيط متجاوب ورسالة علامة واضحة."
					},
					{
						"icon": "🛒",
						"title": "التجارة الإلكترونية",
						"text": "متاجر إلكترونية مع تكامل الدفع ومزامنة المخزون وكتالوج المنتجات وإدارة الطلبات."
					},
					{
						"icon": "🎨",
						"title": "تصميم واجهات المستخدم",
						"text": "واجهات جميلة وبديهية مصممة للتفاعل وسهولة الوصول والتحويل."
					},
					{
						"icon": "⚙️",
						"title": "تطبيقات الويب",
						"text": "تطبيقات ويب مخصصة ولوحات تحكم وبوابات ومنصات SaaS بأطر حديثة."
					},
					{
						"icon": "🔍",
						"title": "تحسين محركات البحث",
						"text": "تحسين محركات البحث، استراتيجية المحتوى، الوسوم الوصفية، البيانات المنظمة وتحسين الأداء."
					},
					{
						"icon": "🔧",
						"title": "الصيانة",
						"text": "تحديثات مستمرة، تصحيحات أمنية، إدارة محتوى، استضافة ومراقبة أداء."
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "مستعد لموقع يعمل بكفاءة؟",
				"text": "أخبرنا بأهدافك ونقترح لك النهج الأمثل.",
				"actions": [{
					"label": "اطلب عرض سعر",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var solutions_web_default = {
	en: en$4,
	ar: ar$4
};
//#endregion
//#region src/content/data/pages/terms.json
var terms_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$3,
	default: () => terms_default,
	en: () => en$3
});
var en$3 = {
	"meta": {
		"title": "Terms of Service — XamlTech",
		"description": "The terms governing the use of XamlTech's website, Odoo addons, ERP systems, AI copilots and Android apps — payment, delivery, warranty, liability and Egypt law.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [{
		"partial": "hero",
		"data": {
			"eyebrow": "📜 Legal",
			"title": "Terms of Service",
			"lead": "These terms govern your use of XamlTech's website and services. Please read them carefully before using any of our services or purchasing any addon."
		}
	}, {
		"partial": "richtext",
		"data": { "html": "<h2>1. Acceptance of terms</h2><p>By accessing xamltech.com, installing any XamlTech addon, or using any of our ERP, AI or Android services, you agree to be bound by these Terms of Service. If you do not agree, please do not use the services.</p><h2>2. Quotes and orders</h2><p>Quotes remain valid for 30 days unless stated otherwise. An order becomes binding when we confirm it in writing (email accepted). The customer explicitly waives their own standard terms and conditions, even if submitted after these terms. Any deviation must be agreed in writing in advance.</p><h2>3. Prices and payment</h2><p>Invoices are payable within 21 working days unless another timeframe is stated on the invoice or order. Where Egyptian law applies withholding at source on invoice amounts, the customer shall provide the corresponding withholding certificate so that the correct amount is credited.</p><h2>4. Late payment</h2><p>If an amount remains outstanding more than sixty (60) days after its due date, XamlTech reserves the right to suspend the service and any related licences until the account is settled, without prejudice to any other rights.</p><h2>5. Delivery and performance</h2><p>XamlTech undertakes to supply performant services in due time in accordance with the agreed timeframes. However, none of its obligations shall be regarded as a commitment to a specific result, and delays outside its control (including third-party platforms, hosting or network outages) do not constitute a breach.</p><h2>6. Addons and licences</h2><p>Unless expressly stated otherwise, addons are provided under the licence published on their Odoo Apps Store listing. Source code for purchased modules is made available as described on the listing. Free modules are provided as-is.</p><h2>7. Warranty and support</h2><p>We warrant that delivered work materially conforms to the agreed specification. Support and warranty periods, where included, are set out in the relevant order. Bug fixes for reported defects in delivered work are provided free of charge during the agreed warranty period.</p><h2>8. Intellectual property</h2><p>Unless otherwise agreed in writing, XamlTech retains ownership of its pre-existing tools, libraries, themes and generic components. The customer receives the rights expressly granted in the applicable licence or order. Custom deliverables may be transferred on full payment, as stated in the order.</p><h2>9. Claims</h2><p>To be admissible, any claim must be notified to XamlTech by recorded delivery (or equivalent traceable email) to its registered address. Claims are assessed in good faith and, where valid, resolved by repair, replacement or an agreed remedy.</p><h2>10. Limitation of liability</h2><p>To the maximum extent permitted by law, XamlTech shall not be liable for indirect, incidental or consequential damages, loss of profit or loss of data. Our aggregate liability is limited to the amounts actually paid for the specific service giving rise to the claim.</p><h2>11. Confidentiality</h2><p>Each party shall keep confidential any non-public information received from the other and use it solely to perform the contract.</p><h2>12. Governing law and jurisdiction</h2><p>All our contractual relations are governed exclusively by Egyptian law. Any dispute shall be subject to the exclusive jurisdiction of the competent courts in Egypt.</p><h2>13. Changes to these terms</h2><p>We may update these terms from time to time. The version published on this page at the time of your order governs that order. Material changes will be announced on this page.</p><h2>14. Contact</h2><ul><li>Email: info@xamltech.com</li><li>Phone: +20 120 141 0140 · +20 100 798 2468</li><li>Location: Aswan, Arab Republic of Egypt</li></ul>" }
	}]
};
var ar$3 = {
	"meta": {
		"title": "الشروط والأحكام — زامل تك",
		"description": "الشروط المنظمة لاستخدام موقع زامل تك وإضافات Odoo ومنظومات ERP ومساعدي الذكاء وتطبيقات أندرويد — الدفع والتسليم والضمان والمسؤولية والقانون المصري.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [{
		"partial": "hero",
		"data": {
			"eyebrow": "📜 قانوني",
			"title": "الشروط والأحكام",
			"lead": "تنظّم هذه الشروط استخدامك لموقع زامل تك وخدماتها. يرجى قراءتها بعناية قبل استخدام أي من خدماتنا أو شراء أي إضافة."
		}
	}, {
		"partial": "richtext",
		"data": { "html": "<h2>1. قبول الشروط</h2><p>بدخولك إلى xamltech.com أو تثبيت أي إضافة من زامل تك أو استخدام أي من خدمات ERP أو الذكاء أو أندرويد، فإنك توافق على الالتزام بهذه الشروط. إذا لم توافق، فيرجى عدم استخدام الخدمات.</p><h2>2. العروض والطلبات</h2><p>تبقى العروض صالحة لمدة 30 يوماً ما لم يُذكر خلاف ذلك. ويصبح الطلب مُلزماً عند تأكيدنا الكتابي له (ويُقبل البريد الإلكتروني). يتنازل العميل صراحةً عن شروطه وأحكامه القياسية حتى لو قُدّمت بعد هذه الشروط، وأي خروج عن ذلك يجب الاتفاق عليه كتابةً مسبقاً.</p><h2>3. الأسعار والدفع</h2><p>تُسدد الفواتير خلال 21 يوم عمل ما لم تُذكر مدة أخرى في الفاتورة أو الطلب. وعندما يطبّق القانون المصري خصماً من المنبع على مبالغ الفواتير، يلتزم العميل بتزويدنا بشهادة الخصم المقابلة ليُقيَّد المبلغ الصحيح.</p><h2>4. التأخر في السداد</h2><p>إذا بقي مبلغ مستحق لأكثر من ستين (60) يوماً بعد تاريخ استحقاقه، فمن حق زامل تك إيقاف الخدمة وأي تراخيص متعلقة بها حتى تسوية الحساب، دون الإخلال بأي حقوق أخرى.</p><h2>5. التسليم والتنفيذ</h2><p>تتعهد زامل تك بتقديم خدمات عالية الأداء في وقتها وفق الأطر الزمنية المتفق عليها. لكن لا يُعد أي من التزاماتها تعهداً بنتيجة محددة، ولا يُشكّل التأخر خارج نطاق سيطرتها (بما يشمل منصات الأطراف الثالثة أو الاستضافة أو انقطاع الشبكة) إخلالاً.</p><h2>6. الإضافات والتراخيص</h2><p>ما لم يُنص صراحةً على غير ذلك، تُقدَّم الإضافات بموجب الترخيص المنشور على صفحتها في متجر Odoo للتطبيقات. ويتاح الكود المصدري للموديولات المشتراة وفق ما هو موضح في الصفحة. وتُقدَّم الموديولات المجانية كما هي.</p><h2>7. الضمان والدعم</h2><p>نضمن أن العمل المُسلَّم مطابق جوهرياً للمواصفات المتفق عليها. وتُحدَّد فترات الدعم والضمان، حيث تُشمَل، في الطلب ذي الصلة. وتُقدَّم إصلاحات العيوب المُبلَّغ عنها في العمل المُسلَّم مجاناً خلال فترة الضمان المتفق عليها.</p><h2>8. الملكية الفكرية</h2><p>ما لم يُتفق كتابةً على غير ذلك، تحتفظ زامل تك بملكية أدواتها ومكتباتها وثيماتها ومكوّناتها العامة. ويحصل العميل على الحقوق الممنوحة صراحةً في الترخيص أو الطلب المعمول به. وقد تنتقل المخرجات المخصصة عند السداد الكامل وفق ما يحدده الطلب.</p><h2>9. المطالبات</h2><p>لتكون مقبولة، يجب إخطار زامل تك بأي مطالبة عبر خطاب مسجّل (أو بريد إلكتروني مكافئ قابل للتتبع) يُرسل إلى عنوانها المسجّل. وتُقيَّم المطالبات بحسن نية، وتُحل عند صحتها بالإصلاح أو الاستبدال أو أي علاج متفق عليه.</p><h2>10. حدود المسؤولية</h2><p>إلى أقصى حد يسمح به القانون، لا تُسأل زامل تك عن الأضرار غير المباشرة أو العارضة أو التبعية أو فقدان الربح أو البيانات. وتقتصر مسؤوليتنا الإجمالية على المبالغ المدفوعة فعلاً للخدمة المعنية بالمطالبة.</p><h2>11. السرية</h2><p>يلتزم كل طرف بالحفاظ على سرية أي معلومات غير عامة يتلقاها من الطرف الآخر واستخدامها لتنفيذ العقد فقط.</p><h2>12. القانون الواجب والاختصاص القضائي</h2><p>تخضع جميع علاقاتنا التعاقدية حصرياً للقانون المصري، وتبقى أي منازعة من اختصاص المحاكم المختصة في مصر.</p><h2>13. التعديلات على هذه الشروط</h2><p>قد نحدّث هذه الشروط من وقت لآخر. ويحكم الإصدار المنشور على هذه الصفحة وقت طلبك ذلك الطلب، وسيُعلن عن أي تغيير جوهري في هذه الصفحة.</p><h2>14. تواصل معنا</h2><ul><li>البريد الإلكتروني: info@xamltech.com</li><li>الهاتف: ‎+20 120 141 0140 · ‎+20 100 798 2468</li><li>الموقع: أسوان — الجمهورية العربية المصرية</li></ul>" }
	}]
};
var terms_default = {
	en: en$3,
	ar: ar$3
};
//#endregion
//#region src/content/data/pages/testimonials.json
var testimonials_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$2,
	default: () => testimonials_default,
	en: () => en$2
});
var en$2 = {
	"meta": {
		"title": "Customer Success Stories: How XamlTech Transformed Business Operations",
		"description": "Real businesses, real results. Hospitals, retailers and enterprises on transforming operations with XamlTech ERP, POS and IT infrastructure.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "⭐⭐⭐⭐⭐ 4.8/5 average rating",
				"title": "Customer success stories",
				"lead": "Real businesses, real results. See how hospitals, retailers and enterprises transformed their operations with XamlTech solutions.",
				"ctas": [{
					"label": "Start your project",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "Request a demo",
					"href": "/request-demo?lang=en",
					"style": "ghost",
					"big": true
				}]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "Success metrics",
				"items": [
					{
						"v": "40+",
						"l": "Organisations served"
					},
					{
						"v": "1M+",
						"l": "Transactions processed"
					},
					{
						"v": "+30%",
						"l": "Avg. ROI improvement"
					},
					{
						"v": "2–6wk",
						"l": "Avg. deployment time"
					}
				]
			}
		},
		{
			"partial": "testimonials",
			"data": {
				"alt": true,
				"eyebrow": "What our clients say",
				"title": "Trusted by 40+ organisations",
				"sub": "Across healthcare, retail and manufacturing.",
				"items": [
					{
						"stars": "★★★★★",
						"quote": "XamlTech completely transformed our hospital's IT infrastructure. The CCTV system is reliable, the network design handles 200+ concurrent users without lag, and the support is outstanding.",
						"name": "Dr. Ahmed Hassan",
						"role": "Administrator, Furkan Surgical Center"
					},
					{
						"stars": "★★★★★",
						"quote": "The POS Cutting Orders module saved us hours daily. Before, cutting operations were manual and error-prone. Now everything is tracked in real time.",
						"name": "Marwa Sayed",
						"role": "Manager, Al-Ghazl Al-Arabi Trading"
					},
					{
						"stars": "★★★★★",
						"quote": "Working with XamlTech on our farm ERP was seamless. Their agriculture expertise is unmatched — they understand crop cycles, harvest planning and work order logic.",
						"name": "Khalid Amin",
						"role": "CEO, Qatf Company (Agriculture)"
					},
					{
						"stars": "★★★★★",
						"quote": "ZATCA compliance for Saudi Arabia was a nightmare until XamlTech stepped in. They handled XML generation, QR codes and the government portal integration.",
						"name": "Noor Al-Rashid",
						"role": "IT Director, Saudi Trading Group"
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Ready to transform your business?",
				"text": "Join 40+ organisations that trust XamlTech for mission-critical solutions.",
				"actions": [{
					"label": "Contact us",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$2 = {
	"meta": {
		"title": "قصص نجاح العملاء: كيف غيّرت زامل تك عمليات الأعمال",
		"description": "أعمال حقيقية ونتائج حقيقية. مستشفيات وشركات تجزئة ومؤسسات تحوّل عملياتها مع حلول زامل تك لأنظمة ERP ونقاط البيع والبنية التحتية.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "⭐⭐⭐⭐⭐ تقييم 4.8/5",
				"title": "قصص نجاح العملاء",
				"lead": "أعمال حقيقية ونتائج حقيقية. اكتشف كيف حوّل عملاؤنا من مستشفيات وشركات تجزئة ومؤسسات عملياتهم مع حلول زامل تك.",
				"ctas": [{
					"label": "ابدأ مشروعك",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "اطلب تجربة",
					"href": "/request-demo",
					"style": "ghost",
					"big": true
				}]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "مؤشرات النجاح",
				"items": [
					{
						"v": "+40",
						"l": "مؤسسة نخدمها"
					},
					{
						"v": "مليون+",
						"l": "عملية معالجة"
					},
					{
						"v": "+30%",
						"l": "متوسط تحسن العائد"
					},
					{
						"v": "2–6 أسابيع",
						"l": "متوسط زمن التنفيذ"
					}
				]
			}
		},
		{
			"partial": "testimonials",
			"data": {
				"alt": true,
				"eyebrow": "ماذا يقول عملاؤنا",
				"title": "موثوقون من أكثر من 40 مؤسسة",
				"sub": "في القطاع الصحي والتجزئة والتصنيع.",
				"items": [
					{
						"stars": "★★★★★",
						"quote": "غيّرت زامل تك بنية مستشفانا التحتية بالكامل. نظام الكاميرات موثوق، وتصميم الشبكة يتحمل أكثر من 200 مستخدم متزامن دون تأخر، والدعم ممتاز.",
						"name": "د. أحمد حسن",
						"role": "مدير مركز الفرقان التخصصي"
					},
					{
						"stars": "★★★★★",
						"quote": "موديول أوامر القص وفّر علينا ساعات يومياً. قبل ذلك كانت عمليات القص يدوية ومعرضة للأخطاء، والآن كل شيء متتبع لحظياً.",
						"name": "مروة سيد",
						"role": "مديرة، الغزل العربي للتجارة"
					},
					{
						"stars": "★★★★★",
						"quote": "كان العمل مع زامل تك على نظام ERP الزراعي سلساً. خبرتهم الزراعية لا مثيل لها — يفهمون دورات المحاصيل وتخطيط الحصاد ومنطق أوامر العمل.",
						"name": "خالد أمين",
						"role": "الرئيس التنفيذي، شركة قطف للزراعة"
					},
					{
						"stars": "★★★★★",
						"quote": "كان توافق ZATCA للسعودية كابوساً حتى تدخلت زامل تك. تولّوا توليد XML ورموز QR وتكامل البوابة الحكومية.",
						"name": "نور الرشيد",
						"role": "مدير تقنية المعلومات، مجموعة تجارية سعودية"
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "مستعد لتحويل أعمالك؟",
				"text": "انضم لأكثر من 40 مؤسسة تثق بزامل تك للحلول الحرجة.",
				"actions": [{
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var testimonials_default = {
	en: en$2,
	ar: ar$2
};
//#endregion
//#region src/content/data/pages/theme-info.json
var theme_info_exports = /* @__PURE__ */ __exportAll({
	ar: () => ar$1,
	default: () => theme_info_default,
	en: () => en$1
});
var en$1 = {
	"meta": {
		"title": "XamlTech Theme — Professional Dark/Light RTL Odoo Website Theme",
		"description": "A professional dark/light Odoo website theme with full RTL Arabic support, CSS design tokens, PWA and JSON-LD SEO. Built for technology companies and ERP consultancies.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🎨 Odoo 19 Website Theme",
				"title": "XamlTech",
				"accent": "Theme",
				"lead": "Professional dark/light toggle theme with full RTL Arabic support. Built for technology companies and ERP consultancies.",
				"points": [
					"Dark mode by default",
					"Full RTL Arabic",
					"CSS design tokens",
					"PWA + JSON-LD SEO"
				],
				"ctas": [{
					"label": "Talk to us",
					"href": "/contact-us?lang=en",
					"style": "primary",
					"big": true
				}, {
					"label": "See the addons store",
					"href": "/addons-store?lang=en",
					"style": "ghost"
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "Theme Features",
				"title": "Built for real products",
				"items": [
					{
						"icon": "🌙",
						"title": "Dark Mode Default",
						"text": "Deep-space dark theme with luminous accent blue. Reduces eye strain and looks stunning on modern displays."
					},
					{
						"icon": "☀️",
						"title": "One-Click Light Mode",
						"text": "Seamless toggle persisted in localStorage. Follows the OS colour-scheme preference automatically."
					},
					{
						"icon": "🌐",
						"title": "Full RTL Arabic",
						"text": "Every component supports right-to-left. Cairo font covers Arabic and Latin in one stack."
					},
					{
						"icon": "🎨",
						"title": "CSS Design Tokens",
						"text": "Zero hardcoded colours. All values in :root CSS variables — one-line rebranding."
					},
					{
						"icon": "📐",
						"title": "Fluid Responsive",
						"text": "CSS clamp() everywhere. Pixel-perfect from 320px to 4K. No media-query breakpoint hacks."
					},
					{
						"icon": "📲",
						"title": "PWA + Service Worker",
						"text": "Offline-first. Web App Manifest included. Installable on mobile home screens."
					},
					{
						"icon": "🛒",
						"title": "Addons Store",
						"text": "Live search, category filter, sort and staggered animations — all vanilla JS, no jQuery."
					},
					{
						"icon": "💬",
						"title": "WhatsApp Integration",
						"text": "Floating button and contact form with a pre-filled WhatsApp deep-link. No plugin needed."
					},
					{
						"icon": "🔍",
						"title": "SEO + JSON-LD",
						"text": "Organization, Service, FAQPage and ItemList structured data. AI-readable on every page."
					}
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "Screenshots",
				"items": [
					{
						"v": "🌙",
						"l": "Dark Mode"
					},
					{
						"v": "☀️",
						"l": "Light Mode"
					},
					{
						"v": "🌐",
						"l": "RTL Arabic"
					},
					{
						"v": "🛒",
						"l": "Addons Store"
					},
					{
						"v": "📱",
						"l": "Mobile Responsive"
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "Want this theme for your Odoo site?",
				"text": "We can adapt the XamlTech theme to your brand and content.",
				"actions": [{
					"label": "Contact us",
					"href": "/contact-us?lang=en",
					"style": "primary"
				}, {
					"label": "WhatsApp us",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var ar$1 = {
	"meta": {
		"title": "ثيم زامل تك — ثيم موقع Odoo احترافي داكن/فاتح بدعم RTL",
		"description": "ثيم موقع Odoo احترافي داكن/فاتح بدعم كامل للعربية RTL وCSS design tokens وPWA وSEO بـ JSON-LD. مصمم لشركات التقنية ومكاتب ERP.",
		"og_image": "https://xamltech.com/static/img/pos-light.png",
		"index": true
	},
	"sections": [
		{
			"partial": "hero",
			"data": {
				"eyebrow": "🎨 ثيم موقع Odoo 19",
				"title": "ثيم",
				"accent": "زامل تك",
				"lead": "ثيم احترافي بنمطين داكن/فاتح مع دعم كامل للعربية RTL. مصمم لشركات التقنية ومكاتب ERP.",
				"points": [
					"النمط الداكن افتراضياً",
					"دعم كامل للعربية RTL",
					"CSS design tokens",
					"PWA وSEO بـ JSON-LD"
				],
				"ctas": [{
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "primary",
					"big": true
				}, {
					"label": "تصفّح متجر الإضافات",
					"href": "/addons-store",
					"style": "ghost"
				}]
			}
		},
		{
			"partial": "features",
			"data": {
				"eyebrow": "مزايا الثيم",
				"title": "مبني لمنتجات حقيقية",
				"items": [
					{
						"icon": "🌙",
						"title": "النمط الداكن افتراضياً",
						"text": "نمط داكن عميق مع لون مميز مضيء. يقلل إجهاد العين ويبدو رائعاً على الشاشات الحديثة."
					},
					{
						"icon": "☀️",
						"title": "النمط الفاتح بلمسة",
						"text": "تبديل سلس يُحفظ في localStorage ويتابع تفضيل النظام تلقائياً."
					},
					{
						"icon": "🌐",
						"title": "دعم كامل للعربية RTL",
						"text": "كل مكوّن يدعم الاتجاه من اليمين لليسار. خط Cairo يغطي العربية واللاتينية."
					},
					{
						"icon": "🎨",
						"title": "CSS design tokens",
						"text": "بلا ألوان مثبّتة. كل القيم في متغيرات :root — إعادة تعريف الهوية بسطر واحد."
					},
					{
						"icon": "📐",
						"title": "تجاوب مرن",
						"text": "CSS clamp() في كل مكان. دقة من 320px إلى 4K بلا حيل نقاط توقف."
					},
					{
						"icon": "📲",
						"title": "PWA + service worker",
						"text": "يعمل دون اتصال. Web App Manifest مضمّن. قابل للتثبيت على شاشة الهاتف."
					},
					{
						"icon": "🛒",
						"title": "متجر الإضافات",
						"text": "بحث حي وفلترة فئات وترتيب وحركات متدرجة — بجافاسكربت خالص بلا jQuery."
					},
					{
						"icon": "💬",
						"title": "تكامل واتساب",
						"text": "زر عائم ونموذج تواصل برابط واتساب مُعبّأ مسبقاً. بلا إضافة خارجية."
					},
					{
						"icon": "🔍",
						"title": "SEO + JSON-LD",
						"text": "بيانات منظمة Organization وService وFAQPage وItemList. مقروء للذكاء الاصطناعي في كل صفحة."
					}
				]
			}
		},
		{
			"partial": "stats",
			"data": {
				"title": "لقطات الشاشة",
				"items": [
					{
						"v": "🌙",
						"l": "النمط الداكن"
					},
					{
						"v": "☀️",
						"l": "النمط الفاتح"
					},
					{
						"v": "🌐",
						"l": "العربية RTL"
					},
					{
						"v": "🛒",
						"l": "متجر الإضافات"
					},
					{
						"v": "📱",
						"l": "متجاوب للجوال"
					}
				]
			}
		},
		{
			"partial": "cta",
			"data": {
				"title": "تريد هذا الثيم لموقع Odoo؟",
				"text": "يمكننا تكييف ثيم زامل تك لهويتك ومحتواك.",
				"actions": [{
					"label": "تواصل معنا",
					"href": "/contact-us",
					"style": "primary"
				}, {
					"label": "واتساب",
					"href": "https://wa.me/201007982468",
					"style": "wa"
				}]
			}
		}
	]
};
var theme_info_default = {
	en: en$1,
	ar: ar$1
};
var strings_default = {
	en: {
		"brand": "XamlTech",
		"tagline": "More Than Technology — We Build Digital Growth",
		"skip": "Skip to content",
		"ctaDemo": "Request a demo",
		"ctaContact": "Contact us",
		"ctaWhatsapp": "WhatsApp chat",
		"login": "Sign in",
		"langSwitch": "العربية",
		"themeDark": "Switch to light mode",
		"themeLight": "Switch to dark mode",
		"menu": "Menu",
		"footerAbout": "Your trusted partner for digital transformation, ERP systems, AI solutions, and IT infrastructure. Based in Aswan, Egypt.",
		"rights": "All rights reserved.",
		"madeIn": "Aswan, Egypt",
		"getInTouch": "Get in Touch",
		"viewsWith": "views with",
		"nav": [
			{
				"label": "Home",
				"href": "/"
			},
			{
				"label": "Solutions",
				"href": "#",
				"children": [
					{
						"label": "ERP & Odoo",
						"href": "/solutions/erp"
					},
					{
						"label": "IT Infrastructure",
						"href": "/solutions/it-infrastructure"
					},
					{
						"label": "Web Development",
						"href": "/solutions/web"
					},
					{
						"label": "Odoo POS",
						"href": "/odoo-pos-egypt"
					},
					{
						"label": "Agriculture ERP",
						"href": "/erp-agriculture-egypt"
					},
					{
						"label": "AI Solutions",
						"href": "/ai-solutions"
					}
				]
			},
			{
				"label": "Addons Store",
				"href": "/addons-store"
			},
			{
				"label": "Portfolio",
				"href": "/portfolio"
			},
			{
				"label": "Blog",
				"href": "/insights"
			},
			{
				"label": "Pricing",
				"href": "/pricing"
			},
			{
				"label": "Contact",
				"href": "/contact-us"
			}
		],
		"footer": [
			{
				"title": "Solutions",
				"links": [
					{
						"label": "ERP & Odoo",
						"href": "/solutions/erp"
					},
					{
						"label": "IT Infrastructure",
						"href": "/solutions/it-infrastructure"
					},
					{
						"label": "Web Development",
						"href": "/solutions/web"
					},
					{
						"label": "AI Copilots",
						"href": "/ai-solutions"
					},
					{
						"label": "Addons Store",
						"href": "/addons-store"
					},
					{
						"label": "Pricing",
						"href": "/pricing"
					}
				]
			},
			{
				"title": "Resources",
				"links": [
					{
						"label": "Blog & Guides",
						"href": "/insights"
					},
					{
						"label": "AI Marketing Guide",
						"href": "/insights/ai-marketing-copilot"
					},
					{
						"label": "AI POS Guide",
						"href": "/insights/ai-pos-copilot"
					},
					{
						"label": "Knowledge Base",
						"href": "/knowledge"
					},
					{
						"label": "Testimonials",
						"href": "/testimonials"
					}
				]
			},
			{
				"title": "Company",
				"links": [
					{
						"label": "Request a demo",
						"href": "/request-demo"
					},
					{
						"label": "Portfolio",
						"href": "/portfolio"
					},
					{
						"label": "Contact",
						"href": "/contact-us"
					},
					{
						"label": "Privacy Policy",
						"href": "/privacy-policy"
					},
					{
						"label": "Terms of Service",
						"href": "/terms"
					}
				]
			}
		],
		"contact": {
			"address": "Aswan, Egypt",
			"phone": "+20 120 141 0140",
			"phone2": "+20 100 798 2468",
			"email": "info@xamltech.com",
			"whatsapp": "201007982468",
			"hours": "Sun–Thu, 9:00–18:00",
			"appsStore": "Odoo Apps Store",
			"github": "GitHub",
			"networktool": "Network Copilot (Android)"
		},
		"form": {
			"name": "Full name",
			"email": "Email address",
			"phone": "Phone (optional)",
			"company": "Company (optional)",
			"product": "Interested in",
			"message": "How can we help?",
			"send": "Send message",
			"submitDemo": "Request free demo",
			"sent": "Thank you! Your message has been received. We will get back to you shortly.",
			"error": "Please complete the required fields with a valid email.",
			"select": "Select a solution",
			"privacyNote": "By submitting you agree to our privacy policy."
		}
	},
	ar: {
		"brand": "زامل تك",
		"tagline": "أكثر من تقنية — نبني نمواً رقمياً حقيقياً",
		"skip": "تخطَّ إلى المحتوى",
		"ctaDemo": "اطلب تجربة",
		"ctaContact": "تواصل معنا",
		"ctaWhatsapp": "محادثة واتساب",
		"login": "تسجيل الدخول",
		"langSwitch": "English",
		"themeDark": "التبديل إلى النمط الفاتح",
		"themeLight": "التبديل إلى النمط الداكن",
		"menu": "القائمة",
		"footerAbout": "شريكك الموثوق في التحول الرقمي وأنظمة ERP وحلول الذكاء الاصطناعي والبنية التحتية لتقنية المعلومات. مقرنا أسوان، مصر.",
		"rights": "جميع الحقوق محفوظة.",
		"madeIn": "أسوان، مصر",
		"getInTouch": "ابقَ على تواصل",
		"viewsWith": "مشاهدات مع",
		"nav": [
			{
				"label": "الرئيسية",
				"href": "/"
			},
			{
				"label": "الحلول",
				"href": "#",
				"children": [
					{
						"label": "ERP و Odoo",
						"href": "/solutions/erp"
					},
					{
						"label": "البنية التحتية والشبكات",
						"href": "/solutions/it-infrastructure"
					},
					{
						"label": "تطوير المواقع",
						"href": "/solutions/web"
					},
					{
						"label": "نقاط البيع Odoo",
						"href": "/odoo-pos-egypt"
					},
					{
						"label": "ERP الزراعة",
						"href": "/erp-agriculture-egypt"
					},
					{
						"label": "حلول الذكاء الاصطناعي",
						"href": "/ai-solutions"
					}
				]
			},
			{
				"label": "متجر الإضافات",
				"href": "/addons-store"
			},
			{
				"label": "أعمالنا",
				"href": "/portfolio"
			},
			{
				"label": "المدونة",
				"href": "/insights"
			},
			{
				"label": "الأسعار",
				"href": "/pricing"
			},
			{
				"label": "تواصل معنا",
				"href": "/contact-us"
			}
		],
		"footer": [
			{
				"title": "الحلول",
				"links": [
					{
						"label": "ERP و Odoo",
						"href": "/solutions/erp"
					},
					{
						"label": "البنية التحتية والشبكات",
						"href": "/solutions/it-infrastructure"
					},
					{
						"label": "تطوير المواقع",
						"href": "/solutions/web"
					},
					{
						"label": "مساعدو الذكاء الاصطناعي",
						"href": "/ai-solutions"
					},
					{
						"label": "متجر الإضافات",
						"href": "/addons-store"
					},
					{
						"label": "الأسعار",
						"href": "/pricing"
					}
				]
			},
			{
				"title": "مصادر",
				"links": [
					{
						"label": "المدونة والأدلة",
						"href": "/insights"
					},
					{
						"label": "دليل مساعد التسويق",
						"href": "/insights/ai-marketing-copilot"
					},
					{
						"label": "دليل مساعد نقاط البيع",
						"href": "/insights/ai-pos-copilot"
					},
					{
						"label": "قاعدة المعرفة",
						"href": "/knowledge"
					},
					{
						"label": "شهادات العملاء",
						"href": "/testimonials"
					}
				]
			},
			{
				"title": "الشركة",
				"links": [
					{
						"label": "اطلب تجربة",
						"href": "/request-demo"
					},
					{
						"label": "أعمالنا",
						"href": "/portfolio"
					},
					{
						"label": "تواصل معنا",
						"href": "/contact-us"
					},
					{
						"label": "سياسة الخصوصية",
						"href": "/privacy-policy"
					},
					{
						"label": "الشروط والأحكام",
						"href": "/terms"
					}
				]
			}
		],
		"contact": {
			"address": "أسوان، مصر",
			"phone": "+20 120 141 0140",
			"phone2": "+20 100 798 2468",
			"email": "info@xamltech.com",
			"whatsapp": "201007982468",
			"hours": "الأحد–الخميس، 9:00–18:00",
			"appsStore": "متجر Odoo للتطبيقات",
			"github": "جيتهَب",
			"networktool": "Network Copilot (أندروید)"
		},
		"form": {
			"name": "الاسم الكامل",
			"email": "البريد الإلكتروني",
			"phone": "الهاتف (اختياري)",
			"company": "الشركة (اختياري)",
			"product": "مهتم بـ",
			"message": "كيف يمكننا مساعدتك؟",
			"send": "أرسل الرسالة",
			"submitDemo": "اطلب تجربة مجانية",
			"sent": "شكراً لك! تم استلام رسالتك وسنتواصل معك قريباً.",
			"error": "يرجى إكمال الحقول المطلوبة ببريد إلكتروني صحيح.",
			"select": "اختر حلاً",
			"privacyNote": "بإرسالك للبيانات فأنت توافق على سياسة الخصوصية."
		}
	}
};
//#endregion
//#region src/content.ts
var pageModules = /* #__PURE__ */ Object.assign({
	"./content/data/pages/addons-store.json": addons_store_exports,
	"./content/data/pages/ai-solutions.json": ai_solutions_exports,
	"./content/data/pages/apk-download.json": apk_download_exports,
	"./content/data/pages/contact-us.json": contact_us_exports,
	"./content/data/pages/erp-agriculture-egypt.json": erp_agriculture_egypt_exports,
	"./content/data/pages/home.json": home_exports,
	"./content/data/pages/insights-ai-marketing-copilot.json": insights_ai_marketing_copilot_exports,
	"./content/data/pages/insights-ai-pos-copilot.json": insights_ai_pos_copilot_exports,
	"./content/data/pages/insights.json": insights_exports,
	"./content/data/pages/knowledge.json": knowledge_exports,
	"./content/data/pages/networktool.json": networktool_exports,
	"./content/data/pages/odoo-pos-egypt.json": odoo_pos_egypt_exports,
	"./content/data/pages/portfolio.json": portfolio_exports,
	"./content/data/pages/pricing.json": pricing_exports,
	"./content/data/pages/privacy-policy.json": privacy_policy_exports,
	"./content/data/pages/request-demo.json": request_demo_exports,
	"./content/data/pages/solutions-erp.json": solutions_erp_exports,
	"./content/data/pages/solutions-it-infrastructure.json": solutions_it_infrastructure_exports,
	"./content/data/pages/solutions-web.json": solutions_web_exports,
	"./content/data/pages/terms.json": terms_exports,
	"./content/data/pages/testimonials.json": testimonials_exports,
	"./content/data/pages/theme-info.json": theme_info_exports
});
var LANGS = ["ar", "en"];
var routes = [
	{
		path: "/",
		slug: "home",
		index: true,
		kind: "home"
	},
	{
		path: "/solutions/erp",
		slug: "solutions-erp",
		index: true,
		kind: "sections"
	},
	{
		path: "/solutions/it-infrastructure",
		slug: "solutions-it-infrastructure",
		index: true,
		kind: "sections"
	},
	{
		path: "/solutions/web",
		slug: "solutions-web",
		index: true,
		kind: "sections"
	},
	{
		path: "/odoo-pos-egypt",
		slug: "odoo-pos-egypt",
		index: true,
		kind: "sections"
	},
	{
		path: "/erp-agriculture-egypt",
		slug: "erp-agriculture-egypt",
		index: true,
		kind: "sections"
	},
	{
		path: "/addons-store",
		slug: "addons-store",
		index: true,
		kind: "sections"
	},
	{
		path: "/ai-solutions",
		slug: "ai-solutions",
		index: true,
		kind: "sections"
	},
	{
		path: "/pricing",
		slug: "pricing",
		index: true,
		kind: "sections"
	},
	{
		path: "/portfolio",
		slug: "portfolio",
		index: true,
		kind: "sections"
	},
	{
		path: "/testimonials",
		slug: "testimonials",
		index: true,
		kind: "sections"
	},
	{
		path: "/knowledge",
		slug: "knowledge",
		index: true,
		kind: "sections"
	},
	{
		path: "/insights",
		slug: "insights",
		index: true,
		kind: "sections"
	},
	{
		path: "/insights/ai-marketing-copilot",
		slug: "insights-ai-marketing-copilot",
		index: true,
		kind: "sections"
	},
	{
		path: "/insights/ai-pos-copilot",
		slug: "insights-ai-pos-copilot",
		index: true,
		kind: "sections"
	},
	{
		path: "/networktool",
		slug: "networktool",
		index: true,
		kind: "sections"
	},
	{
		path: "/apk-download",
		slug: "apk-download",
		index: true,
		kind: "form"
	},
	{
		path: "/request-demo",
		slug: "request-demo",
		index: true,
		kind: "form"
	},
	{
		path: "/contact-us",
		slug: "contact-us",
		index: true,
		kind: "form"
	},
	{
		path: "/theme-info",
		slug: "theme-info",
		index: true,
		kind: "sections"
	},
	{
		path: "/privacy-policy",
		slug: "privacy-policy",
		index: true,
		kind: "sections"
	},
	{
		path: "/terms",
		slug: "terms",
		index: true,
		kind: "sections"
	}
];
var aliases = {
	"/contactus": "/contact-us",
	"/home": "/",
	"/blogs": "/insights"
};
var bySlug = /* @__PURE__ */ new Map();
var pages = {};
for (const key of Object.keys(pageModules)) {
	const base = key.split("/").pop()?.replace(/\.json$/, "") ?? "";
	const data = pageModules[key].default;
	if (!bySlug.has(base)) bySlug.set(base, data);
	const rt = routes.find((r) => r.slug === base);
	if (rt) pages[rt.path] = data;
}
function routeFor(path) {
	return routes.find((r) => r.path === path);
}
function getStrings(lang) {
	return strings_default[lang];
}
function getPage(path, lang) {
	const rt = routeFor(path);
	const data = rt ? bySlug.get(rt.slug) : void 0;
	if (!data) throw new Error(`no content for ${path}`);
	return data[lang];
}
function str(v) {
	if (v == null) return "";
	if (typeof v === "string") return v;
	if (typeof v === "number" || typeof v === "boolean") return String(v);
	return JSON.stringify(v);
}
function arrOf(v) {
	return Array.isArray(v) ? v : [];
}
function cv(obj, key) {
	return str(obj?.[key]);
}
//#endregion
//#region src/i18n.ts
function ogLocaleFor(lang) {
	return lang === "ar" ? "ar_EG" : "en_US";
}
function selfLabel(lang) {
	return lang === "ar" ? "العربية" : "English";
}
function otherLang(lang) {
	return lang === "ar" ? "en" : "ar";
}
function localizeHref(lang, href) {
	if (href === "" || href.startsWith("http") || href.startsWith("mailto:") || href.startsWith("tel:") || href.startsWith("#")) return href;
	if (lang === "en") return href + (href.includes("?") ? "&" : "?") + "lang=en";
	return href;
}
function langSwitchHref(lang, path) {
	if (lang === "ar") return `${path}?lang=en`;
	return `${path}`;
}
//#endregion
//#region src/site.ts
function navItems(lang) {
	return arrOf(getStrings(lang).nav).map((m) => {
		const href = cv(m, "href");
		const ni = {
			label: cv(m, "label"),
			href: localizeHref(lang, href),
			external: href.startsWith("http"),
			children: []
		};
		ni.children = arrOf(m.children).map((k) => {
			const khref = cv(k, "href");
			return {
				label: cv(k, "label"),
				href: localizeHref(lang, khref),
				external: khref.startsWith("http"),
				children: []
			};
		});
		return ni;
	});
}
function footerCols(lang) {
	return arrOf(getStrings(lang).footer).map((c) => ({
		title: cv(c, "title"),
		links: arrOf(c.links).map((l) => {
			const href = cv(l, "href");
			return {
				label: cv(l, "label"),
				href: localizeHref(lang, href),
				external: href.startsWith("http")
			};
		})
	}));
}
function contactOf(lang) {
	const c = getStrings(lang).contact;
	return c && typeof c === "object" ? c : {};
}
//#endregion
//#region src/components/header.tsx
function Logo() {
	return /* @__PURE__ */ jsxs("svg", {
		viewBox: "0 0 32 32",
		"aria-hidden": "true",
		className: "brand-logo",
		children: [
			/* @__PURE__ */ jsx("rect", {
				x: "1",
				y: "1",
				width: "30",
				height: "30",
				rx: "8",
				fill: "#16a34a"
			}),
			/* @__PURE__ */ jsx("path", {
				d: "M9 22.5 16 9l3.4 6.4H23L16 9",
				fill: "none",
				stroke: "#fff",
				strokeWidth: "2.4",
				strokeLinecap: "round",
				strokeLinejoin: "round"
			}),
			/* @__PURE__ */ jsx("path", {
				d: "M16 9v13.5",
				stroke: "#fff",
				strokeWidth: "2.4",
				strokeLinecap: "round"
			})
		]
	});
}
function Header({ lang, path }) {
	const strings = getStrings(lang);
	const contact = contactOf(lang);
	const items = navItems(lang);
	const [menuOpen, setMenuOpen] = useState(false);
	const [openSub, setOpenSub] = useState(null);
	const homeHref = lang === "en" ? "/?lang=en" : "/";
	const demoHref = lang === "en" ? "/request-demo?lang=en" : "/request-demo";
	return /* @__PURE__ */ jsxs(Fragment, { children: [/* @__PURE__ */ jsx("header", {
		className: "site-header",
		id: "top",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container nav",
			children: [
				/* @__PURE__ */ jsxs("a", {
					className: "brand",
					href: homeHref,
					"aria-label": "XamlTech",
					children: [/* @__PURE__ */ jsx(Logo, {}), /* @__PURE__ */ jsxs("span", { children: [/* @__PURE__ */ jsxs("b", { children: ["Xaml", /* @__PURE__ */ jsx("span", {
						className: "brand-accent",
						children: "Tech"
					})] }), /* @__PURE__ */ jsx("small", { children: strings.tagline })] })]
				}),
				/* @__PURE__ */ jsxs("nav", {
					className: `nav-links${menuOpen ? " open" : ""}`,
					id: "site-nav",
					"aria-label": "Primary",
					children: [/* @__PURE__ */ jsx("ul", { children: items.map((it) => /* @__PURE__ */ jsx(NavEntry, {
						item: it,
						open: openSub === it.label,
						onToggle: () => setOpenSub((s) => s === it.label ? null : it.label)
					}, it.label)) }), /* @__PURE__ */ jsxs("div", {
						className: "nav-foot",
						children: [/* @__PURE__ */ jsx("a", {
							className: "btn btn-primary btn-block",
							href: demoHref,
							onClick: () => setMenuOpen(false),
							children: strings.ctaDemo
						}), /* @__PURE__ */ jsx("a", {
							className: "btn btn-wa btn-block",
							href: `https://wa.me/${String(contact.whatsapp)}`,
							target: "_blank",
							rel: "noopener",
							onClick: () => setMenuOpen(false),
							children: strings.ctaWhatsapp
						})]
					})]
				}),
				/* @__PURE__ */ jsxs("div", {
					className: "hdr-actions",
					children: [
						/* @__PURE__ */ jsx("a", {
							className: "hdr-phone",
							href: "tel:+201201410140",
							children: /* @__PURE__ */ jsx("span", {
								dir: "ltr",
								children: "+20 120 141 0140"
							})
						}),
						/* @__PURE__ */ jsx("a", {
							className: "btn btn-primary btn-sm hdr-cta",
							href: demoHref,
							children: strings.ctaDemo
						}),
						/* @__PURE__ */ jsx("a", {
							className: "lang-btn",
							href: langSwitchHref(lang, path),
							"aria-label": `Language: ${otherLang(lang) === "en" ? "English" : "العربية"}`,
							children: selfLabel(otherLang(lang))
						}),
						/* @__PURE__ */ jsx("button", {
							className: "icon-btn burger",
							"aria-label": strings.menu,
							"aria-controls": "site-nav",
							"aria-expanded": menuOpen,
							onClick: () => setMenuOpen((v) => !v),
							children: /* @__PURE__ */ jsx("svg", {
								viewBox: "0 0 24 24",
								"aria-hidden": "true",
								children: /* @__PURE__ */ jsx("path", { d: "M4 7h16M4 12h16M4 17h16" })
							})
						})
					]
				})
			]
		})
	}), /* @__PURE__ */ jsx("div", {
		className: "nav-scrim",
		hidden: !menuOpen,
		onClick: () => setMenuOpen(false)
	})] });
}
function NavEntry({ item, open, onToggle }) {
	if (item.children.length) return /* @__PURE__ */ jsxs("li", {
		className: `nav-item has-menu${open ? " open" : ""}`,
		children: [/* @__PURE__ */ jsxs("button", {
			className: "nav-link",
			"aria-expanded": open,
			"aria-haspopup": "true",
			onClick: onToggle,
			children: [item.label, /* @__PURE__ */ jsx("svg", {
				className: "i-caret",
				viewBox: "0 0 24 24",
				"aria-hidden": "true",
				children: /* @__PURE__ */ jsx("path", { d: "M6 9l6 6 6-6" })
			})]
		}), /* @__PURE__ */ jsx("ul", {
			className: "nav-menu",
			children: item.children.map((c) => /* @__PURE__ */ jsx("li", { children: /* @__PURE__ */ jsx("a", {
				href: c.href,
				...c.external ? {
					target: "_blank",
					rel: "noopener"
				} : {},
				children: c.label
			}) }, c.label))
		})]
	});
	return /* @__PURE__ */ jsx("li", {
		className: "nav-item",
		children: /* @__PURE__ */ jsx("a", {
			className: "nav-link",
			href: item.href,
			...item.external ? {
				target: "_blank",
				rel: "noopener"
			} : {},
			children: item.label
		})
	});
}
//#endregion
//#region src/components/footer.tsx
function Footer({ lang, year }) {
	const strings = getStrings(lang);
	const contact = contactOf(lang);
	const cols = footerCols(lang);
	const homeHref = lang === "en" ? "/?lang=en" : "/";
	const y = year ?? (/* @__PURE__ */ new Date()).getUTCFullYear();
	return /* @__PURE__ */ jsx("footer", {
		className: "site-footer",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container footer",
			children: [
				/* @__PURE__ */ jsxs("div", {
					className: "footer-brand",
					children: [
						/* @__PURE__ */ jsxs("b", { children: ["Xaml", /* @__PURE__ */ jsx("small", { children: /* @__PURE__ */ jsx("span", {
							className: "brand-accent",
							children: "Tech"
						}) })] }),
						/* @__PURE__ */ jsx("p", { children: strings.footerAbout }),
						/* @__PURE__ */ jsxs("div", {
							className: "footer-social",
							children: [/* @__PURE__ */ jsx("a", {
								href: "https://github.com/XamlTech",
								target: "_blank",
								rel: "noopener",
								"aria-label": "GitHub",
								children: /* @__PURE__ */ jsx("svg", {
									viewBox: "0 0 24 24",
									"aria-hidden": "true",
									children: /* @__PURE__ */ jsx("path", { d: "M12 2a10 10 0 0 0-3.2 19.5c.5.1.7-.2.7-.5v-1.9c-2.8.6-3.4-1.2-3.4-1.2-.5-1.1-1.1-1.4-1.1-1.4-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.3 1.1 2.9.8.1-.6.3-1.1.6-1.4-2.2-.2-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.2-.5-1.3.1-2.6 0 0 .8-.3 2.8 1a9.5 9.5 0 0 1 5 0c1.9-1.3 2.7-1 2.7-1 .6 1.3.2 2.4.1 2.6.6.7 1 1.6 1 2.7 0 3.9-2.4 4.8-4.6 5 .4.3.7.9.7 1.9v2.8c0 .3.2.6.7.5A10 10 0 0 0 12 2z" })
								})
							}), /* @__PURE__ */ jsx("a", {
								href: "https://apps.odoo.com/apps/modules/browse?search=xamltech",
								target: "_blank",
								rel: "noopener",
								"aria-label": "Odoo Apps",
								children: String(contact.appsStore)
							})]
						})
					]
				}),
				cols.map((col) => /* @__PURE__ */ jsxs("div", {
					className: "footer-links",
					children: [/* @__PURE__ */ jsx("h4", { children: col.title }), col.links.map((l) => /* @__PURE__ */ jsx("a", {
						href: l.href,
						...l.external ? {
							target: "_blank",
							rel: "noopener"
						} : {},
						children: l.label
					}, l.label))]
				}, col.title)),
				/* @__PURE__ */ jsxs("div", {
					className: "footer-links",
					children: [
						/* @__PURE__ */ jsx("h4", { children: strings.getInTouch }),
						/* @__PURE__ */ jsx("a", {
							href: "tel:+201201410140",
							children: /* @__PURE__ */ jsx("span", {
								dir: "ltr",
								children: String(contact.phone || "")
							})
						}),
						/* @__PURE__ */ jsx("a", {
							href: "tel:+201007982468",
							children: /* @__PURE__ */ jsx("span", {
								dir: "ltr",
								children: String(contact.phone2 || "")
							})
						}),
						/* @__PURE__ */ jsx("a", {
							href: `mailto:${String(contact.email)}`,
							children: String(contact.email)
						}),
						/* @__PURE__ */ jsx("a", {
							href: `https://wa.me/${String(contact.whatsapp)}`,
							target: "_blank",
							rel: "noopener",
							children: strings.ctaWhatsapp
						}),
						/* @__PURE__ */ jsx("span", {
							className: "footer-tiny",
							children: String(contact.address)
						})
					]
				}),
				/* @__PURE__ */ jsxs("div", {
					className: "footer-text",
					children: [
						/* @__PURE__ */ jsxs("p", { children: [
							"© ",
							y,
							" XamlTech. ",
							strings.rights
						] }),
						/* @__PURE__ */ jsx("p", {
							className: "footer-tiny",
							children: strings.madeIn
						}),
						/* @__PURE__ */ jsx("a", {
							className: "footer-tiny",
							href: homeHref,
							children: homeHref === "/" ? "xamltech.com" : "xamltech.com/en"
						})
					]
				})
			]
		})
	});
}
//#endregion
//#region src/components/sections.tsx
function SectionHead({ eyebrow, title, sub }) {
	if (!eyebrow && !title && !sub) return null;
	return /* @__PURE__ */ jsxs("div", {
		className: "section-head",
		children: [
			eyebrow ? /* @__PURE__ */ jsx("span", {
				className: "section-label",
				children: eyebrow
			}) : null,
			title ? /* @__PURE__ */ jsx("h2", {
				className: "section-title",
				children: title
			}) : null,
			sub ? /* @__PURE__ */ jsx("p", {
				className: "section-sub",
				children: sub
			}) : null
		]
	});
}
function Btn({ href, label, style, big, external }) {
	const variant = style && style !== "primary" ? `btn-${style}` : "btn-primary";
	return /* @__PURE__ */ jsx("a", {
		className: `btn ${variant}${big ? " btn-lg" : ""}`,
		href,
		...external ? {
			target: "_blank",
			rel: "noopener"
		} : {},
		children: label
	});
}
function Ticks({ points }) {
	if (!points.length) return null;
	return /* @__PURE__ */ jsx("ul", {
		className: "ticks",
		children: points.map((p, i) => /* @__PURE__ */ jsx("li", { children: p }, i))
	});
}
function Hero({ data, lang }) {
	const points = arrOf(data.points).map(str);
	const ctas = arrOf(data.ctas);
	const stats = arrOf(data.stats);
	const media = data.media ?? null;
	const title = cv(data, "title");
	const accent = cv(data, "accent");
	const eyebrow = cv(data, "eyebrow");
	const lead = cv(data, "lead");
	return /* @__PURE__ */ jsxs("section", {
		className: "hero",
		children: [/* @__PURE__ */ jsxs("div", {
			className: "container hero-inner",
			children: [/* @__PURE__ */ jsxs("div", {
				className: "hero-copy",
				children: [
					eyebrow ? /* @__PURE__ */ jsx("span", {
						className: "pill",
						children: eyebrow
					}) : null,
					/* @__PURE__ */ jsxs("h1", {
						className: "hero-title",
						children: [title, accent ? /* @__PURE__ */ jsxs("span", {
							className: "grad",
							children: [" ", accent]
						}) : null]
					}),
					lead ? /* @__PURE__ */ jsx("p", {
						className: "hero-lead",
						children: lead
					}) : null,
					/* @__PURE__ */ jsx(Ticks, { points }),
					ctas.length ? /* @__PURE__ */ jsx("div", {
						className: "hero-actions",
						children: ctas.map((c, i) => /* @__PURE__ */ jsx(Btn, {
							href: localizeHref(lang, cv(c, "href")),
							label: cv(c, "label"),
							style: cv(c, "style"),
							big: !!c.big,
							external: !!c.external
						}, i))
					}) : null
				]
			}), media ? /* @__PURE__ */ jsx("div", {
				className: "hero-media",
				children: /* @__PURE__ */ jsx(BrowserWindow, { media })
			}) : null]
		}), stats.length ? /* @__PURE__ */ jsx("div", {
			className: "container",
			children: /* @__PURE__ */ jsx("div", {
				className: "stats",
				children: stats.map((s, i) => /* @__PURE__ */ jsxs("div", {
					className: "s",
					children: [/* @__PURE__ */ jsx("b", { children: cv(s, "v") }), /* @__PURE__ */ jsx("small", { children: cv(s, "l") })]
				}, i))
			})
		}) : null]
	});
}
function BrowserWindow({ media }) {
	const src = cv(media, "src");
	const alt = cv(media, "alt");
	const caption = cv(media, "caption") || "xamltech.com";
	const live = cv(media, "live");
	const width = cv(media, "width") || "1200";
	const height = cv(media, "height") || "700";
	if (!media.window) return /* @__PURE__ */ jsx("img", {
		src,
		alt,
		loading: "eager"
	});
	return /* @__PURE__ */ jsxs("figure", {
		className: "window",
		children: [/* @__PURE__ */ jsxs("div", {
			className: "window-bar",
			children: [
				/* @__PURE__ */ jsxs("span", {
					className: "window-dots",
					"aria-hidden": "true",
					children: [
						/* @__PURE__ */ jsx("i", {}),
						/* @__PURE__ */ jsx("i", {}),
						/* @__PURE__ */ jsx("i", {})
					]
				}),
				/* @__PURE__ */ jsx("span", {
					className: "window-url",
					children: caption
				}),
				/* @__PURE__ */ jsxs("span", {
					className: "window-live",
					children: [/* @__PURE__ */ jsx("b", {}), live]
				})
			]
		}), /* @__PURE__ */ jsx("img", {
			src,
			alt,
			width,
			height,
			loading: "eager"
		})]
	});
}
function Chips({ data }) {
	const items = arrOf(data.items).map(str);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: "section section-tight",
		children: /* @__PURE__ */ jsx("div", {
			className: "container chips",
			children: items.map((c, i) => /* @__PURE__ */ jsx("span", {
				className: "chip chip-accent",
				children: c
			}, i))
		})
	});
}
function Features({ data, lang }) {
	const items = arrOf(data.items);
	const cols = cv(data, "cols") || "3";
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		id: cv(data, "id") || void 0,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [
				/* @__PURE__ */ jsx(SectionHead, {
					eyebrow: cv(data, "eyebrow"),
					title: cv(data, "title"),
					sub: cv(data, "sub")
				}),
				/* @__PURE__ */ jsx("div", {
					className: `all-grid cols-${cols}`,
					children: items.map((it, i) => /* @__PURE__ */ jsxs("article", {
						className: "all-card",
						children: [
							it.img ? /* @__PURE__ */ jsx("img", {
								className: "all-img",
								src: cv(it, "img"),
								alt: cv(it, "alt"),
								width: 360,
								height: 200,
								loading: "lazy"
							}) : null,
							/* @__PURE__ */ jsxs("div", {
								className: "all-head",
								children: [it.icon ? /* @__PURE__ */ jsx("div", {
									className: "all-ico",
									children: str(it.icon)
								}) : null, /* @__PURE__ */ jsx("h3", { children: cv(it, "title") })]
							}),
							it.text ? /* @__PURE__ */ jsx("p", {
								className: "lead",
								children: cv(it, "text")
							}) : null,
							arrOf(it.chips).length ? /* @__PURE__ */ jsx("div", {
								className: "all-meta",
								children: arrOf(it.chips).map((c, j) => /* @__PURE__ */ jsx("span", {
									className: "chip",
									children: str(c)
								}, j))
							}) : null
						]
					}, i))
				}),
				data.cta ? /* @__PURE__ */ jsx("p", {
					className: "section-cta",
					children: /* @__PURE__ */ jsx(Btn, {
						href: localizeHref(lang, cv(data.cta, "href")),
						label: cv(data.cta, "label"),
						big: true
					})
				}) : null
			]
		})
	});
}
function Steps({ data }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsx("ol", {
				className: "steps",
				children: items.map((it, i) => /* @__PURE__ */ jsxs("li", { children: [/* @__PURE__ */ jsx("span", {
					className: "step-num",
					children: i + 1
				}), /* @__PURE__ */ jsxs("div", { children: [/* @__PURE__ */ jsx("h3", { children: cv(it, "title") }), /* @__PURE__ */ jsx("p", { children: cv(it, "text") })] })] }, i))
			})]
		})
	});
}
function Stats({ data }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: "section section-tight",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [data.title ? /* @__PURE__ */ jsx(SectionHead, { title: cv(data, "title") }) : null, /* @__PURE__ */ jsx("div", {
				className: "stats",
				children: items.map((s, i) => /* @__PURE__ */ jsxs("div", {
					className: "s",
					children: [/* @__PURE__ */ jsx("b", { children: cv(s, "v") }), /* @__PURE__ */ jsx("small", { children: cv(s, "l") })]
				}, i))
			})]
		})
	});
}
function Testimonials({ data }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsx("div", {
				className: "quote-grid",
				children: items.map((it, i) => /* @__PURE__ */ jsxs("figure", {
					className: "quote",
					children: [
						it.stars ? /* @__PURE__ */ jsx("div", {
							className: "quote-stars",
							children: str(it.stars)
						}) : null,
						/* @__PURE__ */ jsx("blockquote", { children: cv(it, "quote") }),
						/* @__PURE__ */ jsxs("figcaption", { children: [/* @__PURE__ */ jsx("div", {
							className: "quote-name",
							children: cv(it, "name")
						}), /* @__PURE__ */ jsx("div", {
							className: "quote-role",
							children: cv(it, "role")
						})] })
					]
				}, i))
			})]
		})
	});
}
function Faq({ data }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsx("div", {
				className: "faqs",
				children: items.map((it, i) => /* @__PURE__ */ jsxs("details", { children: [/* @__PURE__ */ jsx("summary", { children: cv(it, "q") }), /* @__PURE__ */ jsx("p", { children: cv(it, "a") })] }, i))
			})]
		})
	});
}
function Split({ data, lang }) {
	const points = arrOf(data.points).map(str);
	const media = data.media ?? null;
	const cta = data.cta ?? null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		children: /* @__PURE__ */ jsxs("div", {
			className: `container split${data.reverse ? " split-rev" : ""}`,
			children: [/* @__PURE__ */ jsxs("div", {
				className: "split-body",
				children: [
					/* @__PURE__ */ jsx(SectionHead, {
						eyebrow: cv(data, "eyebrow"),
						title: cv(data, "title"),
						sub: cv(data, "lead")
					}),
					/* @__PURE__ */ jsx(Ticks, { points }),
					cta ? /* @__PURE__ */ jsx("p", {
						className: "section-cta",
						children: /* @__PURE__ */ jsx(Btn, {
							href: localizeHref(lang, cv(cta, "href")),
							label: cv(cta, "label")
						})
					}) : null
				]
			}), media ? /* @__PURE__ */ jsx("div", {
				className: "split-media",
				children: /* @__PURE__ */ jsx("img", {
					src: cv(media, "src"),
					alt: cv(media, "alt"),
					loading: "lazy"
				})
			}) : null]
		})
	});
}
function RichText({ data }) {
	return /* @__PURE__ */ jsx("section", {
		className: "section",
		children: /* @__PURE__ */ jsx("div", {
			className: "container prose",
			id: cv(data, "id") || void 0,
			children: /* @__PURE__ */ jsx("div", { dangerouslySetInnerHTML: { __html: cv(data, "html") } })
		})
	});
}
function Pricing({ data, lang }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	const cols = cv(data, "cols") || "3";
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		id: cv(data, "id") || void 0,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsx("div", {
				className: `plans cols-${cols}`,
				children: items.map((it, i) => {
					const cta = it.cta ?? null;
					return /* @__PURE__ */ jsxs("article", {
						className: `plan${it.featured ? " featured" : ""}`,
						children: [
							it.tag ? /* @__PURE__ */ jsx("span", {
								className: "plan-tag",
								children: str(it.tag)
							}) : null,
							/* @__PURE__ */ jsx("h3", { children: cv(it, "title") }),
							it.text ? /* @__PURE__ */ jsx("p", {
								className: "desc",
								children: cv(it, "text")
							}) : null,
							it.price ? /* @__PURE__ */ jsxs("div", {
								className: "price",
								children: [it.currency ? /* @__PURE__ */ jsx("span", {
									className: "cur",
									children: str(it.currency)
								}) : null, /* @__PURE__ */ jsx("span", {
									className: "amount",
									children: str(it.price)
								})]
							}) : null,
							arrOf(it.features).length ? /* @__PURE__ */ jsx(Ticks, { points: arrOf(it.features).map(str) }) : null,
							cta ? /* @__PURE__ */ jsx("div", {
								className: "fill",
								children: /* @__PURE__ */ jsx(Btn, {
									href: localizeHref(lang, cv(cta, "href")),
									label: cv(cta, "label")
								})
							}) : null
						]
					}, i);
				})
			})]
		})
	});
}
function fmtMoney(n, currency) {
	return `${currency} ${Math.round(n).toLocaleString("en-US")}`;
}
function Roi({ data }) {
	const currency = cv(data, "currency") || "EUR";
	const investment0 = Number(cv(data, "investment")) || 0;
	const monthly0 = Number(cv(data, "monthlyDefault")) || 0;
	const [monthly, setMonthly] = useState(monthly0);
	const [investment, setInvestment] = useState(investment0);
	const m = Number.isFinite(monthly) && monthly >= 0 ? monthly : 0;
	const inv = Number.isFinite(investment) && investment >= 0 ? investment : 0;
	const payback = m > 0 ? `${(inv / m).toFixed(1)} mo` : "—";
	const net = m * 12 - inv;
	const annual = `${net >= 0 ? "+" : ""}${fmtMoney(net, currency)} (${inv > 0 ? Math.round(net / inv * 100) : 0}%)`;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		id: "roi",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsxs("div", {
				className: "roi",
				children: [/* @__PURE__ */ jsxs("div", {
					className: "roi-controls",
					children: [/* @__PURE__ */ jsxs("label", {
						className: "field",
						children: [/* @__PURE__ */ jsx("span", {
							className: "field-label",
							children: cv(data, "monthlyLabel")
						}), /* @__PURE__ */ jsx("input", {
							className: "input",
							type: "number",
							min: 0,
							step: 100,
							value: monthly,
							onChange: (e) => setMonthly(Number(e.target.value))
						})]
					}), /* @__PURE__ */ jsxs("label", {
						className: "field",
						children: [/* @__PURE__ */ jsx("span", {
							className: "field-label",
							children: cv(data, "investmentLabel")
						}), /* @__PURE__ */ jsx("input", {
							className: "input",
							type: "number",
							min: 0,
							step: 50,
							value: investment,
							onChange: (e) => setInvestment(Number(e.target.value))
						})]
					})]
				}), /* @__PURE__ */ jsxs("div", {
					className: "roi-out",
					children: [
						/* @__PURE__ */ jsxs("div", {
							className: "roi-cell",
							children: [/* @__PURE__ */ jsx("div", {
								className: "roi-v",
								children: fmtMoney(m, currency)
							}), /* @__PURE__ */ jsx("div", {
								className: "roi-l",
								children: cv(data, "savingsLabel")
							})]
						}),
						/* @__PURE__ */ jsxs("div", {
							className: "roi-cell",
							children: [/* @__PURE__ */ jsx("div", {
								className: "roi-v",
								children: payback
							}), /* @__PURE__ */ jsx("div", {
								className: "roi-l",
								children: cv(data, "paybackLabel")
							})]
						}),
						/* @__PURE__ */ jsxs("div", {
							className: "roi-cell",
							children: [/* @__PURE__ */ jsx("div", {
								className: "roi-v",
								children: annual
							}), /* @__PURE__ */ jsx("div", {
								className: "roi-l",
								children: cv(data, "annualLabel")
							})]
						})
					]
				})]
			})]
		})
	});
}
function Articles({ data, lang }) {
	const items = arrOf(data.items);
	const more = cv(data, "more");
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: `section${data.alt ? " section-alt" : ""}`,
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [/* @__PURE__ */ jsx(SectionHead, {
				eyebrow: cv(data, "eyebrow"),
				title: cv(data, "title"),
				sub: cv(data, "sub")
			}), /* @__PURE__ */ jsx("div", {
				className: "all-grid cols-3",
				children: items.map((it, i) => /* @__PURE__ */ jsxs("a", {
					className: "all-card link",
					href: localizeHref(lang, cv(it, "href")),
					children: [
						it.tag ? /* @__PURE__ */ jsx("span", {
							className: "chip chip-accent",
							children: str(it.tag)
						}) : null,
						/* @__PURE__ */ jsx("h3", { children: cv(it, "title") }),
						/* @__PURE__ */ jsx("p", {
							className: "lead",
							children: cv(it, "text")
						}),
						/* @__PURE__ */ jsx("span", {
							className: "card-more",
							children: more
						})
					]
				}, i))
			})]
		})
	});
}
function Cta({ data, lang }) {
	const actions = arrOf(data.actions);
	return /* @__PURE__ */ jsx("section", {
		className: "section",
		children: /* @__PURE__ */ jsx("div", {
			className: "container",
			children: /* @__PURE__ */ jsxs("div", {
				className: "cta-band",
				children: [
					/* @__PURE__ */ jsx("h2", { children: cv(data, "title") }),
					data.text ? /* @__PURE__ */ jsx("p", { children: str(data.text) }) : null,
					actions.length ? /* @__PURE__ */ jsx("div", {
						className: "cta-actions",
						children: actions.map((a, i) => /* @__PURE__ */ jsx(Btn, {
							href: localizeHref(lang, cv(a, "href")),
							label: cv(a, "label"),
							style: cv(a, "style"),
							big: true
						}, i))
					}) : null
				]
			})
		})
	});
}
function Logos({ data }) {
	const items = arrOf(data.items);
	if (!items.length) return null;
	return /* @__PURE__ */ jsx("section", {
		className: "section section-tight",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container",
			children: [data.title ? /* @__PURE__ */ jsx("p", {
				className: "section-sub center",
				children: str(data.title)
			}) : null, /* @__PURE__ */ jsx("div", {
				className: "logo-row",
				children: items.map((it, i) => /* @__PURE__ */ jsx("img", {
					src: cv(it, "src"),
					alt: cv(it, "alt"),
					loading: "lazy"
				}, i))
			})]
		})
	});
}
function Section({ partial, data, lang }) {
	switch (partial) {
		case "hero": return /* @__PURE__ */ jsx(Hero, {
			data,
			lang
		});
		case "chips": return /* @__PURE__ */ jsx(Chips, { data });
		case "features": return /* @__PURE__ */ jsx(Features, {
			data,
			lang
		});
		case "logos": return /* @__PURE__ */ jsx(Logos, { data });
		case "steps": return /* @__PURE__ */ jsx(Steps, { data });
		case "stats": return /* @__PURE__ */ jsx(Stats, { data });
		case "testimonials": return /* @__PURE__ */ jsx(Testimonials, { data });
		case "faq": return /* @__PURE__ */ jsx(Faq, { data });
		case "split": return /* @__PURE__ */ jsx(Split, {
			data,
			lang
		});
		case "richtext": return /* @__PURE__ */ jsx(RichText, { data });
		case "pricing": return /* @__PURE__ */ jsx(Pricing, {
			data,
			lang
		});
		case "roi": return /* @__PURE__ */ jsx(Roi, { data });
		case "articles": return /* @__PURE__ */ jsx(Articles, {
			data,
			lang
		});
		case "cta": return /* @__PURE__ */ jsx(Cta, {
			data,
			lang
		});
		default: return null;
	}
}
//#endregion
//#region src/components/form.tsx
function FormPanel({ lang, sent, action, title, sub, submit, privacyLabel, products, strings }) {
	return /* @__PURE__ */ jsx("section", {
		className: "section",
		children: /* @__PURE__ */ jsxs("div", {
			className: "container panel",
			children: [
				/* @__PURE__ */ jsxs("div", {
					className: "panel-head",
					children: [/* @__PURE__ */ jsx("h2", {
						className: "section-title",
						children: title
					}), sub ? /* @__PURE__ */ jsx("p", {
						className: "section-sub",
						children: sub
					}) : null]
				}),
				sent === "1" ? /* @__PURE__ */ jsx("div", {
					className: "notice notice-ok",
					children: cv(strings, "sent")
				}) : null,
				sent === "error" ? /* @__PURE__ */ jsx("div", {
					className: "notice notice-err",
					children: cv(strings, "error")
				}) : null,
				/* @__PURE__ */ jsxs("form", {
					className: "form",
					method: "post",
					action,
					children: [
						/* @__PURE__ */ jsx("input", {
							className: "hp",
							type: "text",
							name: "website",
							tabIndex: -1,
							autoComplete: "off",
							"aria-hidden": "true"
						}),
						/* @__PURE__ */ jsx("input", {
							type: "hidden",
							name: "lang",
							value: lang
						}),
						/* @__PURE__ */ jsxs("div", {
							className: "form-grid",
							children: [
								/* @__PURE__ */ jsxs("div", {
									className: "field",
									children: [/* @__PURE__ */ jsx("label", {
										className: "field-label",
										htmlFor: "f-name",
										children: cv(strings, "name")
									}), /* @__PURE__ */ jsx("input", {
										className: "input",
										id: "f-name",
										name: "name",
										required: true,
										maxLength: 120,
										autoComplete: "name"
									})]
								}),
								/* @__PURE__ */ jsxs("div", {
									className: "field",
									children: [/* @__PURE__ */ jsx("label", {
										className: "field-label",
										htmlFor: "f-email",
										children: cv(strings, "email")
									}), /* @__PURE__ */ jsx("input", {
										className: "input",
										id: "f-email",
										name: "email",
										type: "email",
										required: true,
										maxLength: 160,
										autoComplete: "email"
									})]
								}),
								/* @__PURE__ */ jsxs("div", {
									className: "field",
									children: [/* @__PURE__ */ jsx("label", {
										className: "field-label",
										htmlFor: "f-phone",
										children: cv(strings, "phone")
									}), /* @__PURE__ */ jsx("input", {
										className: "input",
										id: "f-phone",
										name: "phone",
										dir: "ltr",
										maxLength: 40,
										autoComplete: "tel"
									})]
								}),
								/* @__PURE__ */ jsxs("div", {
									className: "field",
									children: [/* @__PURE__ */ jsx("label", {
										className: "field-label",
										htmlFor: "f-company",
										children: cv(strings, "company")
									}), /* @__PURE__ */ jsx("input", {
										className: "input",
										id: "f-company",
										name: "company",
										maxLength: 160,
										autoComplete: "organization"
									})]
								})
							]
						}),
						products.length ? /* @__PURE__ */ jsxs("div", {
							className: "field",
							children: [/* @__PURE__ */ jsx("label", {
								className: "field-label",
								htmlFor: "f-product",
								children: cv(strings, "product")
							}), /* @__PURE__ */ jsxs("select", {
								className: "input",
								id: "f-product",
								name: "product",
								defaultValue: "",
								children: [/* @__PURE__ */ jsx("option", {
									value: "",
									children: cv(strings, "select")
								}), products.map((p) => /* @__PURE__ */ jsx("option", {
									value: p,
									children: p
								}, p))]
							})]
						}) : null,
						/* @__PURE__ */ jsxs("div", {
							className: "field",
							children: [/* @__PURE__ */ jsx("label", {
								className: "field-label",
								htmlFor: "f-msg",
								children: cv(strings, "message")
							}), /* @__PURE__ */ jsx("textarea", {
								className: "input",
								id: "f-msg",
								name: "message",
								rows: 5,
								required: true,
								maxLength: 4e3
							})]
						}),
						/* @__PURE__ */ jsx("button", {
							className: "btn btn-primary btn-lg",
							type: "submit",
							children: submit
						}),
						/* @__PURE__ */ jsxs("p", {
							className: "form-note",
							children: [
								cv(strings, "privacyNote"),
								" ",
								/* @__PURE__ */ jsx("a", {
									href: localizeHref(lang, "/privacy-policy"),
									children: privacyLabel
								})
							]
						})
					]
				})
			]
		})
	});
}
//#endregion
//#region src/components/page.tsx
var HOME_ORDER = [
	["hero", "hero"],
	["chips", "chips"],
	["why", "features"],
	["services", "features"],
	["odoo", "features"],
	["method", "steps"],
	["demo", "features"],
	["addons", "features"],
	["cta", "cta"]
];
var FORM_ACTIONS = {
	"/contact-us": "/api/contact",
	"/request-demo": "/api/demo",
	"/apk-download": "/api/apk"
};
function Page({ route, page, lang, sent }) {
	if (route.kind === "home") return /* @__PURE__ */ jsx(Fragment, { children: HOME_ORDER.map(([key, partial]) => {
		const data = page[key];
		if (!data) return null;
		return /* @__PURE__ */ jsx(Section, {
			partial,
			data,
			lang
		}, key);
	}) });
	if (route.kind === "form") {
		const strings = getStrings(lang);
		return /* @__PURE__ */ jsxs(Fragment, { children: [
			/* @__PURE__ */ jsx(Section, {
				partial: "hero",
				data: page.hero,
				lang
			}),
			/* @__PURE__ */ jsx(FormPanel, {
				lang,
				sent,
				action: FORM_ACTIONS[route.path] ?? "",
				title: cv(page, "formTitle"),
				sub: cv(page, "formSub"),
				submit: cv(page, "submit"),
				privacyLabel: cv(page, "privacyLabel"),
				products: arrOf(page.products).map(str),
				strings: strings.form
			}),
			/* @__PURE__ */ jsx(Section, {
				partial: "cta",
				data: page.cta,
				lang
			})
		] });
	}
	return /* @__PURE__ */ jsx(Fragment, { children: arrOf(page.sections).map((s, i) => /* @__PURE__ */ jsx(Section, {
		partial: cv(s, "partial"),
		data: s.data ?? {},
		lang
	}, i)) });
}
//#endregion
//#region src/app.tsx
function App({ path, lang, sent }) {
	const resolved = aliases[path] ?? path;
	const route = routeFor(resolved);
	if (!route) return notFound(lang);
	const page = getPage(resolved, lang);
	const strings = getStrings(lang);
	const whatsapp = String(contactOf(lang).whatsapp ?? "");
	return /* @__PURE__ */ jsxs(Fragment, { children: [
		/* @__PURE__ */ jsx("a", {
			className: "skip",
			href: "#main",
			children: strings.skip
		}),
		/* @__PURE__ */ jsx(Header, {
			lang,
			path: resolved
		}),
		/* @__PURE__ */ jsx("main", {
			id: "main",
			children: /* @__PURE__ */ jsx(Page, {
				route,
				page,
				lang,
				sent: sent ?? ""
			})
		}),
		/* @__PURE__ */ jsx(Footer, { lang }),
		whatsapp ? /* @__PURE__ */ jsx("a", {
			className: "wa-float",
			href: `https://wa.me/${whatsapp}`,
			target: "_blank",
			rel: "noopener",
			"aria-label": strings.ctaWhatsapp,
			children: /* @__PURE__ */ jsx("svg", {
				viewBox: "0 0 32 32",
				"aria-hidden": "true",
				children: /* @__PURE__ */ jsx("path", { d: "M16 3a13 13 0 0 0-11 20l-2 6 6-2A13 13 0 1 0 16 3zm7 18c-.3.9-1.8 1.7-2.5 1.7-.7 0-1.6.3-5.1-1.3-4.3-2-7-6.5-7.2-6.8-.2-.3-1.7-2.3-1.7-4.4s1.1-3.1 1.5-3.5c.4-.4.9-.5 1.2-.5h.9c.3 0 .7-.1 1 .7l1.3 3c.2.4.2.7 0 1l-.6.9c-.2.3-.5.5-.2 1 .3.5 1.3 2.1 2.8 3.4 1.9 1.7 2.7 1.9 3.1 2 .3.1.6.1.8-.1.3-.3 1-1.2 1.3-1.6.3-.4.5-.3.9-.2l3 1.4c.4.2.7.3.8.5.1.2.1 1-.2 1.9z" })
			})
		}) : null
	] });
}
function notFound(lang) {
	const strings = getStrings(lang);
	return /* @__PURE__ */ jsxs(Fragment, { children: [
		/* @__PURE__ */ jsx("a", {
			className: "skip",
			href: "#main",
			children: strings.skip
		}),
		/* @__PURE__ */ jsx(Header, {
			lang,
			path: "/"
		}),
		/* @__PURE__ */ jsx("main", {
			id: "main",
			children: /* @__PURE__ */ jsx("section", {
				className: "section",
				children: /* @__PURE__ */ jsxs("div", {
					className: "container page-head",
					children: [/* @__PURE__ */ jsx("h1", {
						className: "page-title",
						children: "404"
					}), /* @__PURE__ */ jsx("p", {
						className: "prose",
						children: lang === "ar" ? "الصفحة غير موجودة." : "Page not found."
					})]
				})
			})
		}),
		/* @__PURE__ */ jsx(Footer, { lang })
	] });
}
//#endregion
//#region src/seo.ts
var BASE_URL = "https://xamltech.com";
var CLARITY_ID = "y1l8d34cyq";
function canonicalOf(path, lang) {
	return lang === "en" ? `${BASE_URL}${path}?lang=en` : `${BASE_URL}${path}`;
}
function escapeAttr(s) {
	return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}
function jsonLd(path, _lang, title) {
	const base = BASE_URL;
	const graph = [{
		"@type": "Organization",
		"@id": `${base}/#org`,
		name: "XamlTech",
		url: base,
		logo: `${base}/static/img/logo.png`,
		email: "info@xamltech.com",
		telephone: "+201201410140",
		address: {
			"@type": "PostalAddress",
			addressLocality: "Aswan",
			addressCountry: "EG"
		},
		sameAs: ["https://github.com/XamlTech", "https://apps.odoo.com/apps/modules/browse?search=xamltech"]
	}, {
		"@type": "WebSite",
		"@id": `${base}/#website`,
		url: base,
		name: "XamlTech",
		publisher: { "@id": `${base}/#org` },
		inLanguage: ["ar", "en"]
	}];
	if (path !== "/") graph.push({
		"@type": "BreadcrumbList",
		itemListElement: [{
			"@type": "ListItem",
			position: 1,
			name: "XamlTech",
			item: `${base}/`
		}, {
			"@type": "ListItem",
			position: 2,
			name: title,
			item: `${base}${path}`
		}]
	});
	return JSON.stringify({
		"@context": "https://schema.org",
		"@graph": graph
	});
}
function buildHead({ boot, meta, assetTags, clarityId }) {
	const { path, lang, sent } = boot;
	const canonical = canonicalOf(path, lang);
	const arURL = `${BASE_URL}${path}`;
	const enURL = `${BASE_URL}${path}?lang=en`;
	const alternates = [
		`  <link rel="alternate" hreflang="ar" href="${arURL}">`,
		`  <link rel="alternate" hreflang="en" href="${enURL}">`,
		`  <link rel="alternate" hreflang="x-default" href="${arURL}">`
	].join("\n");
	const robots = meta.index ? "index,follow,max-image-preview:large" : "noindex,nofollow";
	const og = meta.ogImage || `https://xamltech.com/static/img/pos-light.png`;
	const lines = [
		"<meta charset=\"utf-8\">",
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
		`<title>${escapeAttr(meta.title)}</title>`,
		`<meta name="description" content="${escapeAttr(meta.description)}">`,
		`<link rel="canonical" href="${canonical}">`,
		alternates,
		`<meta name="robots" content="${robots}">`,
		"<meta name=\"author\" content=\"XamlTech\">",
		"<meta name=\"generator\" content=\"xamltech-site-react\">",
		"<meta name=\"theme-color\" content=\"#ffffff\">",
		"<meta name=\"geo.region\" content=\"EG-ASN\">",
		"<meta name=\"geo.placename\" content=\"Aswan\">",
		`<meta property="og:type" content="${escapeAttr(meta.type)}">`,
		"<meta property=\"og:site_name\" content=\"XamlTech\">",
		`<meta property="og:locale" content="${ogLocaleFor(lang)}">`,
		`<meta property="og:title" content="${escapeAttr(meta.title)}">`,
		`<meta property="og:description" content="${escapeAttr(meta.description)}">`,
		`<meta property="og:url" content="${canonical}">`,
		`<meta property="og:image" content="${og}">`,
		"<meta name=\"twitter:card\" content=\"summary_large_image\">",
		`<meta name="twitter:title" content="${escapeAttr(meta.title)}">`,
		`<meta name="twitter:description" content="${escapeAttr(meta.description)}">`,
		"<link rel=\"icon\" href=\"/static/img/favicon.ico\" sizes=\"any\">",
		"<link rel=\"icon\" type=\"image/png\" sizes=\"32x32\" href=\"/static/img/favicon-32x32.png\">",
		"<link rel=\"icon\" type=\"image/png\" sizes=\"16x16\" href=\"/static/img/favicon-16x16.png\">",
		"<link rel=\"apple-touch-icon\" href=\"/static/img/logo.png\">",
		"<link rel=\"manifest\" href=\"/manifest.webmanifest\">",
		`  <script type="application/ld+json">${jsonLd(path, lang, meta.title)}<\/script>`
	];
	for (const tag of assetTags) lines.push(`  ${tag}`);
	const clarity = clarityId ? `\n<script type="text/javascript">(function(c,l,a,r,i,t,y){c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y);})(window,document,"clarity","script","${clarityId}");<\/script>` : "";
	return `<head>\n${lines.join("\n")}${clarity}\n</head>`;
}
function metaFrom(page) {
	const m = page.meta ?? {};
	const index = typeof m["index"] === "boolean" ? m["index"] : true;
	const type = typeof m["type"] === "string" && m["type"] !== "" ? m["type"] : "website";
	return {
		title: cv(m, "title"),
		description: cv(m, "description"),
		ogImage: cv(m, "og_image"),
		type,
		index
	};
}
function pageMeta(path, _lang, page) {
	const meta = metaFrom(page);
	const rt = routeFor(path);
	if (rt && !rt.index) meta.index = false;
	return meta;
}
//#endregion
//#region src/ssr.tsx
var ALL_PAGES = routes.flatMap((r) => LANGS.map((lang) => ({
	path: r.path,
	lang
})));
function renderDocument(path, lang, sent, assetTags) {
	const boot = {
		path,
		lang,
		sent: sent || ""
	};
	const meta = routeFor(path) ? pageMeta(path, lang, getPage(path, lang)) : {
		title: "404 — XamlTech",
		description: "",
		ogImage: "",
		type: "website",
		index: false
	};
	const head = buildHead({
		boot,
		meta,
		assetTags,
		clarityId: CLARITY_ID
	});
	const body = renderToString(/* @__PURE__ */ jsx(App, { ...boot }));
	return {
		html: `<!doctype html>\n<html lang="${lang === "ar" ? "ar" : "en"}" dir="${lang === "ar" ? "rtl" : "ltr"}">\n${head}\n<body>\n<div id="root">${body}</div>\n<script>window.__XT__={"path":"${path}","lang":"${lang}","sent":"__SENT__"};<\/script>\n</body>\n</html>`,
		title: meta.title
	};
}
//#endregion
export { ALL_PAGES, App, renderDocument };
