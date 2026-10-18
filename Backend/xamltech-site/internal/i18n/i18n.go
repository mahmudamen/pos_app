// Package i18n holds language primitives for the bilingual (Arabic/English) site.
package i18n

import (
	"net/http"
	"strings"
)

// Lang is a supported language code.
type Lang string

const (
	// AR is Arabic (default, right-to-left).
	AR Lang = "ar"
	// EN is English.
	EN Lang = "en"
)

// Supported returns the languages the site ships.
func Supported() []Lang { return []Lang{AR, EN} }

// Valid reports whether code is a supported language.
func Valid(code string) bool {
	switch Lang(strings.ToLower(strings.TrimSpace(code))) {
	case AR, EN:
		return true
	}
	return false
}

// Normalize coerces any input to a supported language, falling back to def.
func Normalize(code string, def Lang) Lang {
	switch Lang(strings.ToLower(strings.TrimSpace(code))) {
	case AR:
		return AR
	case EN:
		return EN
	default:
		return def
	}
}

// Dir returns the text direction for the language.
func (l Lang) Dir() string {
	if l == AR {
		return "rtl"
	}
	return "ltr"
}

// Locale is the <html lang> value.
func (l Lang) Locale() string {
	if l == AR {
		return "ar"
	}
	return "en"
}

// Other returns the opposite language.
func (l Lang) Other() Lang {
	if l == AR {
		return EN
	}
	return AR
}

// Label is the language's own name, used by the language switcher.
func (l Lang) Label() string {
	if l == AR {
		return "العربية"
	}
	return "English"
}

// CookieName is the cookie that remembers an explicit language choice.
const CookieName = "xt_lang"

// FromRequest resolves the language for a request: an explicit ?lang= wins, then
// the remembered cookie, then Accept-Language, then def.
func FromRequest(r *http.Request, def Lang) Lang {
	if code := r.URL.Query().Get("lang"); code != "" {
		return Normalize(code, def)
	}
	if c, err := r.Cookie(CookieName); err == nil {
		return Normalize(c.Value, def)
	}
	return fromAcceptLanguage(r.Header.Get("Accept-Language"), def)
}

func fromAcceptLanguage(header string, def Lang) Lang {
	header = strings.ToLower(header)
	// Accept-Language looks like "ar-EG,ar;q=0.9,en;q=0.8".
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if tag == "" {
			continue
		}
		if strings.HasPrefix(tag, "ar") {
			return AR
		}
		if strings.HasPrefix(tag, "en") {
			return EN
		}
	}
	return def
}
