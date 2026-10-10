package mainsite

import "net/url"

func urlEncode(s string) string { return url.QueryEscape(s) }

// link is a call-to-action, navigation entry or footer link.
type link struct {
	Label bi
	Href  string
	// Kind selects the button style: "primary", "ghost", "wa" (WhatsApp) or "".
	Kind string
	// Ext renders target="_blank" rel="noopener".
	Ext bool
}

// navGroup is a top-level navigation entry, optionally with a dropdown.
type navGroup struct {
	Key   string
	Label bi
	Href  string
	Items []link
}

// feature is one card of a feature grid: an emoji/SVG icon, a title and body.
type feature struct {
	Icon  string
	Title bi
	Body  bi
}

// card is a richer tile: icon, optional eyebrow/badge, title, body, meta chips
// and a trailing link.
type card struct {
	Icon    string
	Eyebrow bi
	Badge   bi
	Title   bi
	Body    bi
	Meta    []bi
	Price   bi
	Rating  string
	Href    string
	CTA     bi
}

// stat is a headline number with a label.
type stat struct {
	Value bi
	Label bi
}

// step is one node of an ordered process.
type step struct {
	Title bi
	Body  bi
}

// faq is a question/answer pair.
type faq struct {
	Q bi
	A bi
}

// plan is a pricing/package card.
type plan struct {
	Name     bi
	Desc     bi
	Price    bi
	Period   bi
	Currency string
	Badge    bi
	Features []bi
	Featured bool
	Href     string
	CTA      bi
}

// block is a prose block for text pages (privacy, terms, about).
type block struct {
	Heading bi
	Body    []bi
	List    []bi
}

// sectionKind enumerates the layout used for a page section. Templates switch
// on Kind, so a page is just an ordered list of sections.
type sectionKind string

const (
	secFeatures sectionKind = "features" // feature grid
	secCards    sectionKind = "cards"    // card grid (addons, projects, posts)
	secSteps    sectionKind = "steps"    // ordered process
	secPlans    sectionKind = "plans"    // pricing packages
	secStats    sectionKind = "stats"    // headline metrics
	secSplit    sectionKind = "split"    // two-column prose (why/without/with)
	secProse    sectionKind = "prose"    // long-form text
	secFAQ      sectionKind = "faq"      // accordion
	secCTA      sectionKind = "cta"      // closing call to action
)

// section is one band of a page.
type section struct {
	Kind     sectionKind
	Label    bi
	Title    bi
	Lead     bi
	Note     bi
	Features []feature
	Cards    []card
	Steps    []step
	Plans    []plan
	Stats    []stat
	Blocks   []block
	FAQ      []faq
	Links    []link
	// Emojis for compact icon strips (e.g. industries, crops).
	Tags []string
}

// page is everything a route needs to render.
type page struct {
	Slug string // canonical path, no language prefix, e.g. "/solutions/erp"
	// Nav is the section id used to highlight the active menu entry.
	Nav       string
	Title     bi // <title>, without the brand suffix
	Meta      bi // meta description
	Eyebrow   bi
	H1        bi
	Lead      bi
	HeroCTAs  []link
	HeroNote  bi
	HeroTiles []feature // optional small pills under the hero
	Sections  []section
	FAQ       []faq
	Final     section // closing CTA band
	Robots    string  // "" means index,follow; "noindex" for legal pages
}

// nav returns the primary navigation, shared by every page.
func nav() []navGroup {
	return []navGroup{
		{Key: "home", Label: more("Home", "الرئيسية"), Href: "/"},
		{Key: "solutions", Label: more("Solutions", "الحلول"), Href: "/solutions", Items: []link{
			{Label: more("ERP & Odoo", "أنظمة أودو و ERP"), Href: "/solutions/erp"},
			{Label: more("IT Infrastructure", "البنية التحتية للشبكات"), Href: "/solutions/it-infrastructure"},
			{Label: more("Web Development", "تطوير المواقع"), Href: "/solutions/web"},
			{Label: more("AI Copilots", "مساعدو الذكاء الاصطناعي"), Href: "/ai-solutions"},
			{Label: more("Agriculture ERP", "ERP للزراعة"), Href: "/erp-agriculture-egypt"},
		}},
		{Key: "pos", Label: more("Odoo POS", "نقاط بيع Odoo"), Href: "/odoo-pos-egypt"},
		{Key: "addons", Label: more("Addons Store", "متجر الإضافات"), Href: "/addons-store"},
		{Key: "pricing", Label: more("Pricing", "الأسعار"), Href: "/pricing"},
		{Key: "portfolio", Label: more("Portfolio", "أعمالنا"), Href: "/portfolio"},
		{Key: "insights", Label: more("Insights", "المدونة"), Href: "/insights"},
		{Key: "contact", Label: more("Contact", "تواصل معنا"), Href: "/contact-us"},
	}
}
