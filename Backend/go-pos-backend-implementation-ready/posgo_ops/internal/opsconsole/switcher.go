package opsconsole

import (
	"m31labs.dev/fluffyui/accessibility"
	"m31labs.dev/fluffyui/runtime"
	"m31labs.dev/fluffyui/state"
	"m31labs.dev/fluffyui/widgets"
)

// Switcher shows one of two children based on a boolean signal.
//
// It exists because the framework has no reactive conditional: fluffy.Conditional
// picks its child once, when it is constructed, so a screen built out of it
// cannot change. Re-evaluating it would mean rebuilding the whole tree on every
// keystroke, which throws away focus, selection and scroll position.
//
// Only the active child is mounted, exposed to focus scanning and sent messages,
// so the confirmation strip cannot be tabbed into while it is hidden and the
// action buttons cannot receive a keystroke while the strip is up.
type Switcher struct {
	widgets.Base

	whenTrue  runtime.Widget
	whenFalse runtime.Widget
	condition *state.Signal[bool]

	mounted bool
	unsubs  state.Subscriptions
}

// NewSwitcher creates a switcher over whenTrue and whenFalse.
func NewSwitcher(condition *state.Signal[bool], whenTrue, whenFalse runtime.Widget) *Switcher {
	w := &Switcher{
		whenTrue:  whenTrue,
		whenFalse: whenFalse,
		condition: condition,
	}
	w.Base.Role = accessibility.RoleGroup
	w.SetLabel("content")
	return w
}

// Mount starts watching the condition. The framework's MountTree calls this.
func (s *Switcher) Mount() {
	if s == nil {
		return
	}
	s.mounted = true
	s.unsubs.Observe(s.condition, s.onChange)
	s.onChange()
}

// Unmount stops watching, so a signal that fires after teardown cannot touch a
// dead tree.
func (s *Switcher) Unmount() {
	if s == nil {
		return
	}
	s.mounted = false
	s.unsubs.Clear()
}

func (s *Switcher) onChange() {
	if s == nil || !s.mounted {
		return
	}
	s.Invalidate()
}

// active returns the child currently on screen.
func (s *Switcher) active() runtime.Widget {
	if s.condition.Get() {
		return s.whenTrue
	}
	return s.whenFalse
}

func (s *Switcher) Measure(constraints runtime.Constraints) runtime.Size {
	child := s.active()
	if child == nil {
		return runtime.Size{}
	}
	return child.Measure(constraints)
}

func (s *Switcher) Layout(bounds runtime.Rect) {
	s.Base.Layout(bounds)
	if child := s.active(); child != nil {
		child.Layout(bounds)
	}
}

func (s *Switcher) Render(ctx runtime.RenderContext) {
	if child := s.active(); child != nil {
		runtime.RenderChild(ctx, child)
	}
}

func (s *Switcher) HandleMessage(msg runtime.Message) runtime.HandleResult {
	child := s.active()
	if child == nil {
		return runtime.Unhandled()
	}
	return child.HandleMessage(msg)
}

// ChildWidgets exposes only the visible child, which is what focus registration
// and mounting walk.
func (s *Switcher) ChildWidgets() []runtime.Widget {
	child := s.active()
	if child == nil {
		return nil
	}
	return []runtime.Widget{child}
}
