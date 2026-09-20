package ocr

import "strings"

// MatchScore ranks how likely a scanned (already normalized) product name is
// the same item as a catalog name. Range 0..100:
//   - 100: the scanned name contains the catalog name (or vice versa) as a
//     whole normalized string;
//   - token overlap: 70 + 10 per shared significant word beyond the first
//     (so "سكر كيس" vs "سكر" lands at 75);
//   - 0: nothing in common.
//
// The result gates auto-suggest in the OCR response and is stored on the
// purchase line so a human can audit what matched with what. It is heuristic,
// not a learned classifier.
func MatchScore(scannedKey, catalogKey string) int {
	s := strings.TrimSpace(scannedKey)
	c := strings.TrimSpace(catalogKey)
	if s == "" || c == "" {
		return 0
	}
	if s == c || strings.Contains(s, c) || strings.Contains(c, s) {
		return 100
	}
	swords := splitWords(s)
	cwords := splitWords(c)
	if len(swords) == 0 || len(cwords) == 0 {
		return 0
	}
	shared := 0
	for _, cs := range cwords {
		cs = strings.TrimSpace(cs)
		if cs == "" {
			continue
		}
		for _, ss := range swords {
			if strings.TrimSpace(ss) == cs {
				shared++
				break
			}
		}
	}
	if shared == 0 {
		return 0
	}
	score := 70 + 10*(shared-1)
	if score > 99 {
		score = 99
	}
	return score
}

func splitWords(s string) []string {
	return strings.Fields(s)
}
