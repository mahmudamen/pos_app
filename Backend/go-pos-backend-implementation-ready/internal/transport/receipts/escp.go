package receipts

import (
	"fmt"
	"strings"
)

// escposEnc is the core ESC/POS byte encoder. It is deliberately pure Go with
// no hardware dependency so the printer-agnostic receipt pipeline can be unit
// tested offline: the output is written to a byte buffer and any ESC/POS
// compatible thermal printer (USB, Bluetooth, or network) can consume it.

const (
	esc = byte(0x1B)
	gs  = byte(0x1D)

	// cols80 is the 80mm-column width (also the QR/layout default) used for text
	// wrapping and dashed separators. cols58 is the narrower 58mm layout.
	cols80 = 32
	cols58 = 24
)

// receiptCols validates a requested column width, falling back to the 80mm
// default for anything that is not the supported 58mm narrow width.
func receiptCols(cols int) int {
	if cols == cols58 {
		return cols58
	}
	return cols80
}

// ReceiptLine is a single itemized row on the receipt.
type ReceiptLine struct {
	Name  string `json:"name"`
	Sku   string `json:"sku,omitempty"`
	Qty   int64  `json:"quantity"`
	Price int64  `json:"unit_price_minor"`
	Total int64  `json:"total_minor"`
}

// ReceiptPayment shows how the sale was tendered.
type ReceiptPayment struct {
	Method string `json:"method"`
	Amount int64  `json:"amount_minor"`
	Tip    int64  `json:"tip_minor"`
}

// Receipt is the full printable payload for one sale. The same struct is both
// the JSON body of GET /sales/:id/receipt and the input to BuildBytes.
type Receipt struct {
	TenantName     string           `json:"tenant_name"`
	TenantAddress  string           `json:"tenant_address,omitempty"`
	SaleID         string           `json:"sale_id"`
	Status         string           `json:"status"`
	CreatedAt      string           `json:"created_at"`
	Cashier        string           `json:"cashier"`
	Device         string           `json:"device,omitempty"`
	TableName      string           `json:"table_name,omitempty"`
	FloorName      string           `json:"floor_name,omitempty"`
	CustomerName   string           `json:"customer_name,omitempty"`
	Currency       string           `json:"currency"`
	Lines          []ReceiptLine    `json:"items"`
	SubtotalMinor  int64            `json:"subtotal_minor"`
	DiscountMinor  int64            `json:"discount_minor"`
	TipsMinor      int64            `json:"tips_minor"`
	TotalMinor     int64            `json:"total_minor"`
	Payments       []ReceiptPayment `json:"payments"`
	LoyaltyPoints  int64            `json:"loyalty_points_earned"`
	IdempotencyKey string           `json:"idempotency_key"`
}

// ReceiptOptions tunes how BuildBytes lays out the receipt stream. The zero
// value is the standard 80mm layout with a trailing cut.
type ReceiptOptions struct {
	// Cols is the printable width (cols80 default, cols58 narrow). Zero uses
	// the 80mm layout.
	Cols int
	// Cut appends GS V 0 (full paper cut). Zero omits the cut command.
	Cut bool
	// Compact skips the inter-line feeds for a denser 58mm receipt.
	Compact bool
}

// printer wraps a growing byte slice with the ES/POS command stream.
type printer struct {
	b []byte
}

func (p *printer) raw(data ...byte) {
	p.b = append(p.b, data...)
}

func (p *printer) text(s string) {
	p.b = append(p.b, []byte(s)...)
}

func (p *printer) init() {
	p.raw(esc, 0x40) // ESC @ reset
}

func (p *printer) align(n byte) {
	p.raw(esc, 0x61, n) // ESC a n: 0 left, 1 center, 2 right
}

func (p *printer) emphasize(bold bool) {
	mode := byte(0x00)
	if bold {
		mode = 0x01 // double height
	}
	p.raw(gs, 0x21, mode) // GS ! n
}

func (p *printer) feed(n byte) {
	p.raw(esc, 0x64, n) // ESC d n
}

func (p *printer) qr(url string) {
	if url == "" {
		return
	}
	// GS ( k pL pH fn ...
	p.raw(gs, 0x28, 0x6B, 0x04, 0x00, 0x31, 0x51, 0x30) // function 51 model 0
	p.raw(gs, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x43, 0x04) // function 67 size 4
	p.raw(gs, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x45, 0x30) // function 69 error correction L
	p.raw(gs, 0x28, 0x6B,
		byte(len(url)&0xFF), byte((len(url)>>8)&0xFF), 0x31, 0x50,
	) // function 80 store data (pL/pH then fn header)
	p.b = append(p.b, []byte(url)...)
	p.raw(gs, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x51, 0x30) // function 81 print
}

// BuildBytes renders a Receipt to a byte slice usable by an ESC/POS printer.
// opts selects the print width (cols80 = 32, default, for 80mm paper; cols58 =
// 24 for 58mm — anything else falls back to 32), whether the GS V 0 paper-cut
// command is appended (multi-copy jobs send one cut after the last copy rather
// than cutting between copies), and a compact layout that skips the inter-line
// feeds for denser printing on narrow paper.
func BuildBytes(r Receipt, opts ReceiptOptions) []byte {
	width := receiptCols(opts.Cols)
	p := &printer{}
	p.init()
	if !opts.Compact {
		p.feed(1)
	}
	p.align(1)
	p.emphasize(true)
	p.text(center(r.TenantName, width))
	p.emphasize(false)
	p.align(0)
	if r.TenantAddress != "" {
		p.text(r.TenantAddress)
		if !opts.Compact {
			p.feed(1)
		}
	}
	p.text(dashes(width))
	id := r.SaleID
	if runes := []rune(id); len(runes) > 26 {
		id = string(runes[:26])
	}
	p.text(fmt.Sprintf("Sale     %s", id))
	p.text(fmt.Sprintf("Date     %s", r.CreatedAt))
	if r.Cashier != "" {
		p.text(fmt.Sprintf("Cashier  %s", r.Cashier))
	}
	if r.Device != "" {
		p.text(fmt.Sprintf("Device   %s", r.Device))
	}
	if r.FloorName != "" || r.TableName != "" {
		p.text(fmt.Sprintf("Table    %s/%s", r.FloorName, r.TableName))
	}
	if r.CustomerName != "" {
		p.text(fmt.Sprintf("Customer %s", r.CustomerName))
	}
	p.text(dashes(width))
	for _, line := range r.Lines {
		name := line.Name
		if line.Sku != "" {
			name = name + " [" + line.Sku + "]"
		}
		p.text(truncate(name, width))
		p.text(fmt.Sprintf("%d x %s", line.Qty, money(line.Price, r.Currency)) +
			" ... " + money(line.Total, r.Currency))
	}
	p.text(dashes(width))
	p.text(pair("Subtotal", money(r.SubtotalMinor, r.Currency), width))
	if r.DiscountMinor > 0 {
		p.text(pair("Discount", "-"+money(r.DiscountMinor, r.Currency), width))
	}
	if r.TipsMinor > 0 {
		p.text(pair("Tip", money(r.TipsMinor, r.Currency), width))
	}
	if len(r.Payments) > 0 {
		for _, pay := range r.Payments {
			p.text(pair("Paid("+pay.Method+")", money(pay.Amount, r.Currency), width))
		}
	}
	p.text(dashes(width))
	p.align(1)
	p.emphasize(true)
	p.text(center("TOTAL "+money(r.TotalMinor, r.Currency), width))
	p.emphasize(false)
	p.align(0)
	if r.LoyaltyPoints > 0 {
		p.feed(1)
		p.text(fmt.Sprintf("Loyalty points earned: %d", r.LoyaltyPoints))
	}
	p.feed(1)
	p.qr("posgo:sale:" + r.SaleID)
	p.feed(3)
	if opts.Cut {
		p.raw(gs, 0x56, 0x00) // GS V 0 full cut
	}
	return p.b
}

func money(minor int64, currency string) string {
	abs := minor
	sign := ""
	if minor < 0 {
		sign = "-"
		abs = -minor
	}
	whole := abs / 100
	frac := abs % 100
	return fmt.Sprintf("%s%s%d.%02d", sign, currencySymbol(currency), whole, frac)
}

func currencySymbol(currency string) string {
	switch currency {
	case "EGP":
		return "E£"
	default:
		return currencyCode(currency)
	}
}

func currencyCode(currency string) string {
	if currency == "" {
		return ""
	}
	return currency + " "
}

// center pads a line so it is centered within the printable width.
func center(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	pad := (width - len(runes)) / 2
	return spaces(pad) + s + spaces(width-len(runes)-pad)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	// n is bounded by the widest layout (cols80, 32)
	const block = 32
	var buf [block]byte
	for i := range buf {
		buf[i] = ' '
	}
	return string(buf[:n])
}

func dashes(width int) string {
	return strings.Repeat("-", width)
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}

func pair(label, value string, width int) string {
	labelWidth := 14
	if width < cols80 {
		labelWidth = 10
	}
	pad := labelWidth - len([]rune(label))
	if pad < 1 {
		pad = 1
	}
	return label + spaces(pad) + value
}
