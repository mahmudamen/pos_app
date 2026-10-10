// Package mainsite renders the XamlTech company website (xamltech.com) as a
// set of server-rendered, bilingual (English / Arabic RTL) pages.
//
// The package is deliberately self-contained: it depends only on the standard
// library and gin, needs no database, and is served by the small cmd/site
// binary on its own port. That mirrors how the Odoo website this replaces was
// served (a single origin on port 8069) without pulling the whole pos-api
// runtime along.
//
// Every page is data-driven: copy lives in content.go as bilingual (bi) values
// and templates render them through the t helper, so the two languages can
// never drift out of sync at the key level.
package mainsite

// Lang is the active page language.
type Lang string

const (
	EN Lang = "en"
	AR Lang = "ar"
)

// Dir returns the text direction for the language.
func (l Lang) Dir() string {
	if l == AR {
		return "rtl"
	}
	return "ltr"
}

// Other returns the opposite language, for the language-switch link.
func (l Lang) Other() Lang {
	if l == AR {
		return EN
	}
	return AR
}

// bi is a bilingual string. En is always required; Ar may be empty, in which
// case the English text is used as a graceful fallback.
type bi struct {
	En string
	Ar string
}

// L resolves the value for a language.
func (b bi) L(l Lang) string {
	if l == AR && b.Ar != "" {
		return b.Ar
	}
	return b.En
}

// more builds a bilingual value from an English/Arabic pair.
func more(en, ar string) bi { return bi{En: en, Ar: ar} }
