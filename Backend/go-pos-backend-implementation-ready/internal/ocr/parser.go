package ocr

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LineItem is one parsed purchase-invoice row. Money is integer minor units
// (price 25.50 -> UnitPriceMinor 2550) and Quantity is the raw invoice amount
// (may be fractional, e.g. 1.5 kg). Score quantifies how much structure the
// parser could extract; it is NOT an OCR confidence.
type LineItem struct {
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	UnitPriceMinor int64   `json:"unit_price_minor"`
	TotalMinor     int64   `json:"total_minor"`
	Score          int     `json:"score"`
	RowText        string  `json:"row_text"`
}

// ParseResult is the outcome of ParseInvoice: accepted items plus the rows the
// heuristics could not understand (returned verbatim so a human can review).
type ParseResult struct {
	Items     []LineItem `json:"items"`
	Discarded []string   `json:"discarded"`
}

var (
	reWord = regexp.MustCompile(`[\p{L}\p{M}](?:[\p{L}\p{M}.,]*[\p{L}\p{M}])?`)
)

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokWord
	tokMultiply // x X × *
	tokEquals   // =
)

type token struct {
	kind  tokenKind
	text  string
	value float64 // tokNumber only
}

func tokenize(s string) []token {
	var toks []token
	runes := []rune(s)
	for i := 0; i < len(runes); {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		switch r {
		case 'x', 'X', '×', '*':
			toks = append(toks, token{kind: tokMultiply, text: string(r)})
			i++
			continue
		case '=':
			toks = append(toks, token{kind: tokEquals, text: "="})
			i++
			continue
		}
		if unicode.IsDigit(r) {
			j := i
			for j < len(runes) && (unicode.IsDigit(runes[j]) || runes[j] == '.') {
				j++
			}
			v, _ := strconv.ParseFloat(string(runes[i:j]), 64)
			toks = append(toks, token{kind: tokNumber, text: string(runes[i:j]), value: v})
			i = j
			continue
		}
		if loc := reWord.FindStringIndex(string(runes[i:])); loc != nil && loc[0] == 0 {
			word := string(runes[i : i+utf8.RuneCountInString(string(runes[i:])[:loc[1]])])
			toks = append(toks, token{kind: tokWord, text: word})
			i += utf8.RuneCountInString(word)
			continue
		}
		i++
	}
	return toks
}

// splitDecimal normalizes digit strings produced by DigitsASCII: an embedded
// "500مل" is split into the number and the word so "ml" participates in unit
// detection, and a Latin comma between digits becomes a decimal point.
// DigitsASCII only maps glyphs; this pass re-joins structure.
func splitDecimal(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '.' || r == ',' {
			// Decimal comma only when flanked by digits (Egypt uses "25,50").
			if i > 0 && i+1 < len(runes) && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]) {
				sb.WriteRune('.')
			} else {
				sb.WriteRune(' ')
			}
			continue
		}
		if !unicode.IsDigit(r) {
			sb.WriteRune(r)
			continue
		}
		// Fold the maximal digit run; a trailing word like "500مل" becomes
		// "500" then "مل" is picked up by the next iteration.
		j := i
		for j < len(runes) && unicode.IsDigit(runes[j]) {
			j++
		}
		sb.WriteString(string(runes[i:j]))
		i = j - 1
	}
	return sb.String()
}

// ParseInvoice converts raw OCR text into structured line items. Rows that
// yield neither a product name nor a quantity are reported verbatim in
// Discarded for human review. The heuristics understand the dominant Egyptian
// layout: "name qty(unit) × unit_price = total" and "name qty unit price".
func ParseInvoice(text string) ParseResult {
	var res ParseResult
	rows := strings.Split(text, "\n")
	for _, raw := range rows {
		if item, ok := parseLine(raw); ok {
			res.Items = append(res.Items, item)
		} else if trimmed := strings.TrimSpace(raw); trimmed != "" {
			res.Discarded = append(res.Discarded, raw)
		}
	}
	return res
}

func parseLine(raw string) (LineItem, bool) {
	s := splitDecimal(DigitsASCII(raw))
	toks := tokenize(s)
	if len(toks) == 0 {
		return LineItem{}, false
	}

	nums := make([]int, 0, 4)
	for i, t := range toks {
		if t.kind == tokNumber {
			nums = append(nums, i)
		}
	}
	if len(nums) == 0 {
		return LineItem{}, false
	}

	// Locate a multiplier (x) if any. The number directly before it is usually
	// the quantity ("جبنه 5 × 45.00"), but in "كوكاكولا 500مل × 12 = 12.50" the
	// "500مل" before the x is a pack descriptor and 12 is the real quantity.
	mulPos := -1
	for i, t := range toks {
		if t.kind == tokMultiply {
			mulPos = i
			break
		}
	}

	// unitToken is the canonical unit resolved for the line; sizeOnly tracks a
	// unit that sits between a quantity candidate and the x (pack descriptor).
	var (
		qtyPos, pricePos, totalPos int
		excludeUnits               = map[int]bool{}
	)
	switch {
	case mulPos >= 0:
		before := numberBefore(toks, mulPos)
		hasSizeUnit := false
		if before >= 0 {
			for i := before + 1; i < mulPos; i++ {
				if toks[i].kind == tokWord && ResolveUnit(toks[i].text) != "" {
					hasSizeUnit = true
					excludeUnits[i] = true
				}
			}
		}
		if hasSizeUnit {
			qtyPos = numberAfter(toks, mulPos)
			pricePos = numberAfter(toks, qtyPos)
			if pricePos < 0 {
				pricePos = qtyPos
			}
			totalPos = nums[len(nums)-1]
		} else {
			qtyPos, pricePos = before, numberAfter(toks, mulPos)
			totalPos = nums[len(nums)-1]
		}
	case len(nums) >= 3:
		qtyPos, pricePos, totalPos = nums[0], nums[len(nums)-2], nums[len(nums)-1]
	case len(nums) == 2:
		qtyPos, pricePos, totalPos = nums[0], nums[1], -1
	default:
		qtyPos, pricePos, totalPos = nums[0], nums[0], -1
	}

	qty := 1.0
	if qtyPos >= 0 {
		qty = toks[qtyPos].value
		if qty <= 0 {
			qty = 1.0
		}
	}
	unitPrice := toks[pricePos].value
	unitPriceMinor := int64(math.Round(unitPrice * 100))
	_total := int64(0)
	if totalPos >= 0 {
		_total = int64(math.Round(toks[totalPos].value * 100))
	}
	totalMinor := int64(math.Round(qty * unitPrice * 100)) // server re-derives on apply

	// Units: prefer the alias nearest the quantity token.
	unit := "piece"
	best := len(toks) + 1
	for i, t := range toks {
		if t.kind != tokWord {
			continue
		}
		if u := ResolveUnit(t.text); u != "" && !excludeUnits[i] {
			dist := abs(i - qtyPos)
			if dist < best || (dist == best && u != "piece") {
				best = dist
				unit = u
			}
		}
	}

	// Name = remaining word tokens minus units, currencies and noise.
	nameParts := make([]string, 0, 8)
	for _, t := range toks {
		if t.kind != tokWord {
			continue
		}
		if ResolveUnit(t.text) != "" {
			continue
		}
		if isCurrencyToken(t.text) {
			continue
		}
		nameParts = append(nameParts, t.text)
	}
	name := strings.TrimSpace(strings.Join(nameParts, " "))

	score := 0
	switch {
	case name == "" && unitPriceMinor == 0:
		return LineItem{}, false
	case name == "":
		score = 30
	case mulPos >= 0 && unitPriceMinor > 0:
		score = 100
	case unitPriceMinor > 0 && qty > 0:
		score = 75
	default:
		score = 50
	}
	if _total > 0 && _total > totalMinor {
		totalMinor = _total
	}

	return LineItem{
		Name:           name,
		Quantity:       qty,
		Unit:           unit,
		UnitPriceMinor: unitPriceMinor,
		TotalMinor:     totalMinor,
		Score:          score,
		RowText:        strings.TrimSpace(raw),
	}, true
}

func numberBefore(toks []token, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if toks[i].kind == tokNumber {
			return i
		}
	}
	return -1
}

func numberAfter(toks []token, pos int) int {
	for i := pos + 1; i < len(toks); i++ {
		if toks[i].kind == tokNumber {
			return i
		}
	}
	return -1
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
