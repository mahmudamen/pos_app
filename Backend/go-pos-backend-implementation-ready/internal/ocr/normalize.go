// Package ocr turns a scanned Arabic purchase invoice into structured line
// items. The heavy lifting (Arabic-aware text normalization, digit folding and
// line-item heuristics) is pure Go and fully unit-tested; only engine.go shells
// out to a local Tesseract binary so the rest of the stack runs without CGO or
// cloud services. The output is advisory: totals are re-derived server-side on
// apply and a human reviews the parsed lines before they hit inventory.
package ocr

import (
	"strings"
	"unicode"
)

var arabicToASCII = map[rune]rune{
	// Arabic-Indic digits ٠..٩
	'\u0660': '0', '\u0661': '1', '\u0662': '2', '\u0663': '3', '\u0664': '4',
	'\u0665': '5', '\u0666': '6', '\u0667': '7', '\u0668': '8', '\u0669': '9',
	// Extended (Persian) digits ۰..۹
	'\u06f0': '0', '\u06f1': '1', '\u06f2': '2', '\u06f3': '3', '\u06f4': '4',
	'\u06f5': '5', '\u06f6': '6', '\u06f7': '7', '\u06f8': '8', '\u06f9': '9',
	// Arabic decimal separator ٫ -> full stop so "٥٫٥" parses as 5.5.
	'\u066b': '.',
}

const (
	arTatweel = '\u0640' // ـ
	arAlef    = '\u0627' // ا
	arHeh     = '\u0647' // ه
	arYeh     = '\u064a' // ي
)

// isTashkeel reports whether r is a short-vowel / diacritic mark that print
// publishers add but OCR engines drop unpredictably (syntax marks U+064B-0652
// plus the superscript alef U+0670). Matching on text without them is far more
// forgiving.
func isTashkeel(r rune) bool {
	return (r >= '\u064b' && r <= '\u0652') || r == '\u0670'
}

// toASCII maps one decimal-family character to its ASCII equivalent.
func toASCII(r rune) (rune, bool) {
	if d, ok := arabicToASCII[r]; ok {
		return d, true
	}
	return r, false
}

// DigitsASCII folds every digit glyph (Arabic-Indic and Persian) to ASCII and
// turns the Arabic decimal separator into a period. Everything else — letters,
// punctuation, separators — is untouched so the line parser can still see
// structure. This is the lightest touch an invoice line needs before parsing.
func DigitsASCII(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if d, ok := toASCII(r); ok {
			runes[i] = d
		}
	}
	return string(runes)
}

// NormalizeKey strips the spelling variance that makes OCR text differ from a
// catalog name even when they mean the same word: tashkeel and tatweel are
// removed, alef/hamza and teh-marbuta spellings are unifed (ء/أ/إ/آ -> ا,
// ة -> ه, ى -> ي) and digits are folded to ASCII. The result is the canonical
// key used for product matching and unit lookup; it drops punctuation and
// collapses runs of whitespace. It is intentionally NOT used to display text.
func NormalizeKey(s string) string {
	var sb []rune
	prevSpace := true
	for _, r := range []rune(s) {
		if isTashkeel(r) || r == arTatweel {
			continue
		}
		switch r {
		case '\u0621', '\u0622', '\u0623', '\u0625':
			r = arAlef
		case '\u0629':
			r = arHeh
		case '\u0649':
			r = arYeh
		}
		if d, ok := toASCII(r); ok {
			r = d
		}
		if unicode.IsSpace(r) {
			if prevSpace {
				continue
			}
			prevSpace = true
			sb = append(sb, ' ')
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			// Punctuation is dropped, but it still separates words: "مدمس--الى"
			// yields "مدمس الى", not "مدمسالى".
			if !prevSpace {
				sb = append(sb, ' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		sb = append(sb, r)
	}
	return strings.TrimSpace(string(sb))
}
