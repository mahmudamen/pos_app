// Package opsconsole renders the POS.Go control plane as a terminal
// application: a login gate, the platform overview, a filterable tenant table
// with the platform actions, and a per-tenant drill-down.
//
// The console is driven by signals and resources from the framework's state
// package. Every fetch goes through state.Resource, which runs the fetcher off
// the UI goroutine and discards a stale response when the operator changes a
// filter mid-flight, so no view code ever blocks the event loop.
package opsconsole

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
	"m31labs.dev/fluffyui/accessibility"
	"m31labs.dev/fluffyui/backend"
	"m31labs.dev/fluffyui/runtime"
	"m31labs.dev/fluffyui/state"
	"m31labs.dev/fluffyui/widgets"
)

// baseStyle is the console's default text style. Custom widgets start from it
// so a colour tweak lands everywhere at once.
func baseStyle() backend.Style { return backend.DefaultStyle() }

// textWidth measures a string in terminal cells. runewidth is used rather than
// a rune count because tenant and product names are free text: a CJK name
// occupies two cells per rune and would otherwise overflow its column.
func textWidth(text string) int { return runewidth.StringWidth(text) }

// StatTile is a single headline number with a caption, used across the top of
// the overview tab. It is a custom widget, so it carries accessibility
// metadata: a screen reader should announce "Total revenue, 986,030 Egyptian
// pounds" as one phrase rather than two disconnected fragments of text.
type StatTile struct {
	widgets.Base

	caption string
	value   string
	hint    string
	urgent  bool
}

// NewStatTile builds a tile. hint is the accessible description and stays off
// screen unless the operator asks for detail; urgent tints the value for
// states an operator must act on, such as a nonzero stopped-tenant count.
func NewStatTile(caption, value, hint string, urgent bool) *StatTile {
	tile := &StatTile{caption: caption, value: value, hint: hint, urgent: urgent}
	tile.Role = accessibility.RoleGroup
	tile.Label = caption
	tile.Description = hint
	return tile
}

// SetValue updates the rendered number. The announced description is derived
// from the same fields on every read, so the two cannot drift apart.
func (t *StatTile) SetValue(value string) {
	t.value = value
	t.Invalidate()
}

// SetUrgent tints the value, for states an operator has to act on. It exists so
// the urgency flag can be cleared as well as set: a repaint that only ever set
// it would leave a tile permanently red after the first bad news.
func (t *StatTile) SetUrgent(urgent bool) {
	t.urgent = urgent
	t.Invalidate()
}

// Text is the tile's announced form.
func (t *StatTile) Text() string {
	if t.hint == "" {
		return fmt.Sprintf("%s %s", t.caption, t.value)
	}
	return fmt.Sprintf("%s %s, %s", t.caption, t.value, t.hint)
}

// Render draws the caption above the value inside the tile's own bounds.
//
// The coordinates come from ContentBounds, never from the buffer origin. A tile
// is five across in an HStack, and drawing at (0,0) painted every one of them
// on top of each other in the screen's top-left corner, straight over the nav
// bar - which is what a live run of the console showed.
func (t *StatTile) Render(ctx runtime.RenderContext) {
	style := baseStyle()
	if t.urgent {
		style = style.Foreground(backend.ColorRed).Bold(true)
	}
	bounds := t.ContentBounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}
	ctx.Buffer.SetString(bounds.X, bounds.Y, truncate(t.caption, bounds.Width), style)
	if bounds.Height > 1 {
		ctx.Buffer.SetString(bounds.X, bounds.Y+1, truncate(t.value, bounds.Width), style)
	}
}

// Measure reports a two-line tile as wide as its widest line.
func (t *StatTile) Measure(constraints runtime.Constraints) runtime.Size {
	w := textWidth(t.caption)
	if vw := textWidth(t.value); vw > w {
		w = vw
	}
	return runtime.Size{Width: w, Height: 2}
}

// HandleMessage ignores input; a tile is display-only.
func (t *StatTile) HandleMessage(msg runtime.Message) runtime.HandleResult {
	return runtime.Unhandled()
}

// tone is the severity of a status, kept separate from the label so the same
// word can be re-coloured if the palette changes.
type tone int

const (
	toneNeutral tone = iota
	toneGood
	toneWarn
	toneBad
)

// toneFor maps a tenant status onto its severity. A stopped tenant is the only
// state that locks a merchant out of signing in, so it is the only one that
// earns the alarm colour; suspended is a warning, everything else is neutral.
func toneFor(status string) tone {
	switch status {
	case statusActive:
		return toneGood
	case statusSuspended:
		return toneWarn
	case statusDisabled:
		return toneBad
	default:
		return toneNeutral
	}
}

// StatusPill renders a tenant status as a bracketed tag so status cells stay
// legible when a column of them is scanned.
type StatusPill struct {
	widgets.Base

	label string
	tone  tone
}

// NewStatusPill builds a status tag from a raw tenant status.
func NewStatusPill(status string) *StatusPill {
	pill := &StatusPill{label: statusPill(status), tone: toneFor(status)}
	pill.Role = accessibility.RoleStatus
	pill.Label = "Tenant status"
	pill.Description = pill.label
	return pill
}

// Text is the on-screen tag, e.g. "[STOPPED]".
func (p *StatusPill) Text() string { return "[" + p.label + "]" }

// Render draws the tag in its severity colour.
func (p *StatusPill) Render(ctx runtime.RenderContext) {
	style := baseStyle().Bold(true)
	switch p.tone {
	case toneGood:
		style = style.Foreground(backend.ColorGreen)
	case toneWarn:
		style = style.Foreground(backend.ColorYellow)
	case toneBad:
		style = style.Foreground(backend.ColorRed)
	}
	ctx.Buffer.SetString(0, 0, p.Text(), style)
}

// Measure reports the tag's own width.
func (p *StatusPill) Measure(constraints runtime.Constraints) runtime.Size {
	return runtime.Size{Width: textWidth(p.Text()), Height: 1}
}

// HandleMessage ignores input; the pill is display-only.
func (p *StatusPill) HandleMessage(msg runtime.Message) runtime.HandleResult {
	return runtime.Unhandled()
}

// KeyHint is the footer explaining the console's key bindings. It is a note
// landmark, so assistive tech can jump to it rather than read past it.
type KeyHint struct {
	widgets.Base

	keys string
}

// NewKeyHint builds the footer.
func NewKeyHint(keys string) *KeyHint {
	hint := &KeyHint{keys: keys}
	hint.Role = accessibility.RoleNote
	hint.Label = "Key bindings"
	hint.Description = keys
	return hint
}

// Text is the footer line.
func (h *KeyHint) Text() string { return h.keys }

// Render draws the bindings dimmed.
func (h *KeyHint) Render(ctx runtime.RenderContext) {
	ctx.Buffer.SetString(0, 0, h.keys, baseStyle().Foreground(backend.ColorBrightBlack))
}

// Measure reports the footer's width.
func (h *KeyHint) Measure(constraints runtime.Constraints) runtime.Size {
	return runtime.Size{Width: textWidth(h.keys), Height: 1}
}

// HandleMessage ignores input.
func (h *KeyHint) HandleMessage(msg runtime.Message) runtime.HandleResult {
	return runtime.Unhandled()
}

// pad right-pads a cell to width, truncating with an ellipsis when it does not
// fit. Table columns are fixed width, so every cell goes through this to keep
// the grid aligned regardless of content length.
func pad(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) > width {
		if width == 1 {
			return "…"
		}
		return string(runes[:width-1]) + "…"
	}
	return text + strings.Repeat(" ", width-len(runes))
}

// truncate shortens text to width without padding, for the one-line status bar
// where trailing spaces would be noise.
func truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

// vGap is a vertical gap of n rows.
//
// It exists because fluffy.Spacer is not a gap: its measure returns the full
// MaxSize, so inside a VStack it claims the entire height budget and every
// widget after it is laid out with zero height and paints nothing. The console
// shipped with Spacer between every section and rendered an empty screen for
// exactly that reason. FixedSpace is the right helper but returns a FlexChild,
// which only fits the flex-typed fluffy.Fixed builders, not the plain
// VStack/HStack used here.
//
// The gap measures zero across so it never influences the column's width, and
// vGap's counterpart hGap measures zero tall: in a row flex the child's height
// is the cross axis, so a full-height gap would set the whole row's height.
func vGap(n int) runtime.Widget {
	w := widgets.NewSimpleWidget()
	w.MeasureFunc = func(c runtime.Constraints) runtime.Size {
		return c.Constrain(runtime.Size{Width: 0, Height: n})
	}
	return w
}

// hGap is the horizontal counterpart of vGap: a gap of n columns.
func hGap(n int) runtime.Widget {
	w := widgets.NewSimpleWidget()
	w.MeasureFunc = func(c runtime.Constraints) runtime.Size {
		return c.Constrain(runtime.Size{Width: n, Height: 0})
	}
	return w
}

// multilineLabel renders a block of text one row per line, and measures as tall
// as the text actually is.
//
// It exists because widgets.SignalLabel measures every string as a single row,
// even one containing newlines. The overview breakdown and the tenant drill-down
// are both multi-line blocks inside a titled card, so the card sized itself to
// one row of text and clipped everything below it: the breakdown lost every
// business type past the first, and the drill-down lost the counts, the revenue
// trend and the top products. The label then painted the overflow past the card
// border, or off the bottom of the pane.
//
// The widget follows the framework's SignalLabel contract - it reads its source
// when it mounts and then follows it by subscription - so it has to be mounted
// with the rest of the pane to come alive.
type multilineLabel struct {
	widgets.Base

	source  state.Readable[string]
	lines   []string
	style   backend.Style
	mounted bool
	subs    state.Subscriptions
}

// multilineText builds a reactive multi-line block. It is the counterpart to
// fluffy.ReactiveText, which cannot be used for anything taller than one row.
func multilineText(render func() string, deps ...state.Subscribable) runtime.Widget {
	label := &multilineLabel{style: baseStyle()}
	label.Base.Role = accessibility.RoleText
	label.source = state.NewComputed(render, deps...)
	label.lines = splitLines(label.source.Get())
	label.Base.Label = strings.Join(label.lines, " ")
	return label
}

// splitLines splits on newlines and drops a trailing blank line, so a value that
// ends in "\n" does not render as an extra empty row.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Measure reports the widest line by n rows.
func (m *multilineLabel) Measure(constraints runtime.Constraints) runtime.Size {
	width := 0
	for _, line := range m.lines {
		if w := textWidth(line); w > width {
			width = w
		}
	}
	rows := len(m.lines)
	if rows == 0 {
		rows = 1
	}
	return constraints.Constrain(runtime.Size{Width: width, Height: rows})
}

// Render draws one line per row inside the label's own bounds.
func (m *multilineLabel) Render(ctx runtime.RenderContext) {
	bounds := m.ContentBounds()
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}
	for i, line := range m.lines {
		if i >= bounds.Height {
			return
		}
		ctx.Buffer.SetString(bounds.X, bounds.Y+i, truncate(line, bounds.Width), m.style)
	}
}

// HandleMessage ignores input; a text block is display-only.
func (m *multilineLabel) HandleMessage(msg runtime.Message) runtime.HandleResult {
	return runtime.Unhandled()
}

// Mount starts following the source.
func (m *multilineLabel) Mount() {
	if m == nil {
		return
	}
	m.mounted = true
	m.refresh()
	m.subs.Observe(m.source, m.onChange)
}

// Unmount stops following, so a fetch landing during teardown cannot touch a
// dead widget.
func (m *multilineLabel) Unmount() {
	if m == nil {
		return
	}
	m.mounted = false
	m.subs.Clear()
}

// onChange re-reads the source when one of its dependencies moves.
func (m *multilineLabel) onChange() {
	if m == nil || !m.mounted {
		return
	}
	m.refresh()
	m.Invalidate()
}

// refresh pulls the current text and keeps the accessible label in step with it.
func (m *multilineLabel) refresh() {
	m.lines = splitLines(m.source.Get())
	m.Base.Label = strings.Join(m.lines, " ")
}

var _ runtime.Widget = (*multilineLabel)(nil)
