package mainsite

// Brand constants. Kept in one place so the contact details, links and demo
// credentials used across pages cannot drift apart.
const (
	brandName    = "XamlTech"
	brandNameAr  = "زامل تك"
	brandSub     = "Odoo · POS · IT Infrastructure"
	brandSubAr   = "أودو · نقاط بيع · بنية تحتية"
	brandTagline = "More than technology — we build digital growth."
	brandTagAr   = "أكثر من تقنية — نبني نمو رقمي حقيقي."

	founderName = "Mahmoud Amen"
	founderRole = "Founder & Lead Developer"

	phoneA = "+20 100 798 2468"
	phoneB = "+20 120 141 0140"
	// WhatsApp deep link (international number without '+', with spaces removed).
	whatsapp = "https://wa.me/201007982468"

	emailMain = "info@xamltech.com"

	addrEn  = "Sahary, Aswan, Egypt"
	addrAr  = "الصحاري، أسوان، مصر"
	hours   = "Sun–Thu 9AM–6PM"
	hoursAr = "الأحد–الخميس ٩ص–٦م"

	githubURL    = "https://github.com/XamlTech"
	githubRepo   = "https://github.com/mahmudamen/pos-go"
	odooAppsURL  = "https://apps.odoo.com/apps/modules/browse?search=xamltech"
	playStoreURL = "https://play.google.com/store/apps/details?id=com.xamltech.pos_go"
	posGoURL     = "https://posgo.xamltech.com"
	consoleURL   = "https://api.xamltech.com/admin/"

	// Public Odoo demo credentials (already published on the old site).
	demoURL  = "demo18.xamltech.com"
	demoUser = "demo"
	demoPass = "demo"

	companyYear = "2026"
)

// whatsappWith builds a WhatsApp deep link with a prefilled message.
func whatsappWith(msg string) string {
	if msg == "" {
		return whatsapp
	}
	return whatsapp + "?text=" + urlEncode(msg)
}

// mailto builds a mailto link, optionally with a subject.
func mailto(subject string) string {
	if subject == "" {
		return "mailto:" + emailMain
	}
	return "mailto:" + emailMain + "?subject=" + urlEncode(subject)
}
