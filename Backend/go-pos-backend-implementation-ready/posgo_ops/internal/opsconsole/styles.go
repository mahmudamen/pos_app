package opsconsole

import (
	"m31labs.dev/fluffyui/style"
)

// Palette for the console. The values are deliberately restrained: colour is
// reserved for conveying state (a stopped tenant is the only red in the
// interface), never for decoration.
var (
	colInk     = style.RGB(232, 232, 236)
	colSurface = style.RGB(24, 24, 28)
	colMuted   = style.RGB(140, 140, 150)
	colAccent  = style.RGB(120, 170, 255)
	colGood    = style.RGB(126, 211, 135)
	colWarn    = style.RGB(240, 200, 100)
	colBad     = style.RGB(240, 120, 120)
)

// ConsoleStylesheet is the FSS rulesheet for the whole app.
//
// It is built in Go rather than parsed from a .fss file so the console stays a
// single binary: an operator SSHes into the VPS and runs one command, and a
// missing sidecar stylesheet would otherwise fail at the worst moment. The
// syntax is the same, so a designer can lift a block into app.fss later if the
// theming ever needs to be hot-reloadable.
func ConsoleStylesheet() *style.Stylesheet {
	return style.NewStylesheet().
		// Base text. The console is dense, so the default is a slightly
		// dimmed ink rather than pure white to reduce halation on dark
		// terminals.
		Add(style.Select("*"), style.Style{
			Foreground: style.RGB(232, 232, 236),
			Background: style.RGB(18, 18, 22),
		}).
		// Headings and tile captions read as structure, not as shouting.
		Add(style.Select("Label").Class("heading"), style.Style{
			Foreground: style.RGB(245, 245, 248),
			Bold:       style.Bool(true),
		}).
		Add(style.Select("Label").Class("muted"), style.Style{
			Foreground: style.RGB(140, 140, 150),
		}).
		// Buttons: one primary weight for the affirmative action in view,
		// everything else secondary so the eye lands on one target.
		Add(style.Select("Button").Class("primary"), style.Style{
			Foreground: style.RGB(12, 12, 16),
			Background: colAccent,
			Bold:       style.Bool(true),
		}).
		Add(style.Select("Button").Class("secondary"), style.Style{
			Foreground: colInk,
			Background: colSurface,
		}).
		// The destructive action is the only red button in the interface.
		// Stopping a merchant locks them out, so it must never be reachable
		// by muscle memory from a neighbouring button.
		Add(style.Select("Button").Class("danger"), style.Style{
			Foreground: style.RGB(255, 240, 240),
			Background: colBad,
			Bold:       style.Bool(true),
		}).
		// Focus is inverted rather than outlined: a reverse-video cell is
		// legible on every terminal, including ones that ignore colour.
		Add(style.Select("Button").Pseudo(style.PseudoFocus), style.Style{
			Reverse: style.Bool(true),
		}).
		Add(style.Select("Input").Pseudo(style.PseudoFocus), style.Style{
			Reverse: style.Bool(true),
		}).
		// Tables get a subtle surface so the data area reads as a panel
		// against the app background.
		Add(style.Select("Table"), style.Style{
			Foreground: colInk,
			Background: colSurface,
		}).
		Add(style.Select("Card"), style.Style{
			Foreground: colInk,
			Background: colSurface,
		}).
		// The tab strip is navigation chrome, so it sits quieter than the
		// content it switches between. The selected-tab affordance is left to
		// the widget: Tabs renders it in reverse video, which reads correctly
		// on terminals that ignore colour entirely, so there is no class to
		// hook here and no rule to add.
		Add(style.Select("Tabs"), style.Style{
			Foreground: colMuted,
			Background: colSurface,
		}).
		// Semantic accents, referenced by AddClass from the view so the state
		// colours are defined once.
		Add(style.Select("*").Class("tone-good"), style.Style{
			Foreground: colGood,
		}).
		Add(style.Select("*").Class("tone-warn"), style.Style{
			Foreground: colWarn,
		}).
		Add(style.Select("*").Class("tone-bad"), style.Style{
			Foreground: colBad,
		})
}
