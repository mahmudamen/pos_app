package ocr

import "strings"

// unitAliases maps Arabic/Egyptian invoice spelling (already run through
// NormalizeKey, so ة/alef variants are folded) to the canonical units.code
// values from migration 029. A purchase line like "5 كجم" resolves to kg.
var unitAliases = map[string]string{
	// piece / each
	"قطعه": "piece", "حبه": "piece", "قطع": "piece", "حبات": "piece",
	// dozen
	"دسته": "dozen", "دستات": "dozen",
	// box / carton
	"علبه": "box", "كرتونه": "box", "صندوق": "box", "علب": "box", "كراتين": "box",
	// pack / bag / bundle
	"عبوه": "pack", "باكيت": "pack", "كيس": "pack", "شوال": "pack",
	"رزمه": "pack", "ربطه": "pack", "اكياس": "pack", "عبوات": "pack",
	// kilogram
	"كيلو": "kg", "كجم": "kg", "كلغ": "kg", "كغم": "kg", "كج": "kg", "كيلوات": "kg",
	// gram
	"جرام": "g", "جم": "g", "غرام": "g",
	// litre
	"لتر": "liter", "لترات": "liter",
	// millilitre
	"مل": "ml", "مليلتر": "ml",
	// metre
	"متر": "m", "امتار": "m",
	// square metre
	"مترمربع": "qm",
}

// arabicCurrencyTokens are Arabic price markers stripped from a parsed name.
// All keys are in NormalizeKey form. Note "ج.م" (Egyptian pound) is deliberately
// absent: its folded form "جم" collides with the gram unit, and gram follows a
// quantity while the pound abbreviation appears near totals, so guessing wrong
// damages product names. Arabic totals are covered by "جنيه"/"الاجمالي" below.
var arabicCurrencyTokens = map[string]bool{
	"جنيه": true, "جنيهات": true,
	"الاجمالي": true, "الاجمالى": true, "اجمالي": true, "اجمالى": true, "الجمالي": true,
	"ريال": true, "دولار": true, "دينار": true,
}

var latinCurrencyTokens = map[string]bool{
	"egp": true, "le": true, "e": true, "usd": true, "sar": true,
}

// ResolveUnit returns the canonical units.code for a token (e.g. "كجم" -> "kg"
// and "قطعه" -> "piece"), or "" when the token is not a known unit alias.
func ResolveUnit(token string) string {
	return unitAliases[NormalizeKey(token)]
}

// isCurrencyToken reports whether token is a price marker (Egyptian-jargon
// Arabic words or a Latin currency code). Latin tokens are compared in ASCII
// casefold form; Arabic tokens go through NormalizeKey first.
func isCurrencyToken(token string) bool {
	if arabicCurrencyTokens[NormalizeKey(token)] {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(token))
	return latinCurrencyTokens[lower]
}
