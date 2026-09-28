package opsconsole

import (
	"strings"
	"testing"

	"m31labs.dev/fluffyui/runtime"
	"m31labs.dev/fluffyui/state"
)

// countingWidget records how often it was rendered, so a test can tell which
// branch of a Switcher is actually on screen.
type countingWidget struct {
	mark     string
	renders  int
	measured int
}

func (w *countingWidget) Measure(runtime.Constraints) runtime.Size {
	w.measured++
	return runtime.Size{Width: 10, Height: 1}
}

func (w *countingWidget) Layout(runtime.Rect) {}
func (w *countingWidget) HandleMessage(runtime.Message) runtime.HandleResult {
	return runtime.Unhandled()
}

func (w *countingWidget) Render(runtime.RenderContext) { w.renders++ }

func TestSwitcherShowsTheFalseBranchByDefault(t *testing.T) {
	condition := state.NewSignal(false)
	whenTrue, whenFalse := &countingWidget{mark: "true"}, &countingWidget{mark: "false"}
	s := NewSwitcher(condition, whenTrue, whenFalse)

	if got := s.active(); got != runtime.Widget(whenFalse) {
		t.Fatalf("a false condition should show whenFalse, got %T", got)
	}
}

func TestSwitcherFollowsTheSignal(t *testing.T) {
	condition := state.NewSignal(false)
	whenTrue, whenFalse := &countingWidget{mark: "true"}, &countingWidget{mark: "false"}
	s := NewSwitcher(condition, whenTrue, whenFalse)

	condition.Set(true)
	if got := s.active(); got != runtime.Widget(whenTrue) {
		t.Fatalf("after Set(true) the switcher should show whenTrue, got %T", got)
	}
	condition.Set(false)
	if got := s.active(); got != runtime.Widget(whenFalse) {
		t.Fatalf("after Set(false) the switcher should show whenFalse, got %T", got)
	}
}

// TestSwitcherOnlyExposesTheVisibleChild is the property that keeps a hidden
// confirmation strip out of the tab order: focus registration and mounting both
// walk ChildWidgets, so a hidden branch that reported itself would still be
// focusable and would still be mounted.
func TestSwitcherOnlyExposesTheVisibleChild(t *testing.T) {
	condition := state.NewSignal(false)
	whenTrue, whenFalse := &countingWidget{mark: "true"}, &countingWidget{mark: "false"}
	s := NewSwitcher(condition, whenTrue, whenFalse)

	children := s.ChildWidgets()
	if len(children) != 1 {
		t.Fatalf("expected exactly one child, got %d", len(children))
	}
	if children[0] != runtime.Widget(whenFalse) {
		t.Errorf("the hidden branch must not be exposed")
	}

	condition.Set(true)
	if s.ChildWidgets()[0] != runtime.Widget(whenTrue) {
		t.Errorf("the visible branch should be exposed once the condition flips")
	}
}

func TestSwitcherDelegatesToTheActiveChild(t *testing.T) {
	condition := state.NewSignal(false)
	whenTrue, whenFalse := &countingWidget{mark: "true"}, &countingWidget{mark: "false"}
	s := NewSwitcher(condition, whenTrue, whenFalse)

	constraints := runtime.Constraints{MaxWidth: 80, MaxHeight: 24}
	_ = s.Measure(constraints)
	if whenFalse.measured != 1 {
		t.Errorf("Measure should reach the visible child, got %d calls", whenFalse.measured)
	}
	if whenTrue.measured != 0 {
		t.Errorf("Measure must not reach the hidden child, got %d calls", whenTrue.measured)
	}

	s.Render(runtime.RenderContext{})
	if whenFalse.renders != 1 {
		t.Errorf("Render should reach the visible child, got %d calls", whenFalse.renders)
	}
	if whenTrue.renders != 0 {
		t.Errorf("Render must not reach the hidden child, got %d calls", whenTrue.renders)
	}
}

// TestSwitcherIgnoresSignalsAfterUnmount matters because the app unmounts the
// tree on quit while fetches may still be landing; without the mounted guard a
// late signal would invalidate a dead widget.
func TestSwitcherIgnoresSignalsAfterUnmount(t *testing.T) {
	condition := state.NewSignal(false)
	whenTrue, whenFalse := &countingWidget{mark: "true"}, &countingWidget{mark: "false"}
	s := NewSwitcher(condition, whenTrue, whenFalse)

	s.Mount()
	s.Unmount()
	renders := whenTrue.renders
	condition.Set(true)

	// active() reads the signal, so it flips regardless; what must not happen is
	// a repaint being scheduled against the torn-down tree.
	if s.active() != runtime.Widget(whenTrue) {
		t.Errorf("active() should still reflect the signal")
	}
	if whenTrue.renders != renders {
		t.Errorf("a signal after Unmount must not drive a render, got %d", whenTrue.renders-renders)
	}
}

func TestSwitcherSurvivesUnmountBeforeMount(t *testing.T) {
	s := NewSwitcher(state.NewSignal(false), &countingWidget{}, &countingWidget{})
	// Unmounting a tree that was never mounted must not panic; the app calls it
	// defensively during teardown.
	s.Unmount()
	s.Mount()
	s.Unmount()
}

func TestSwitcherHasALabelForScreenReaders(t *testing.T) {
	s := NewSwitcher(state.NewSignal(false), &countingWidget{}, &countingWidget{})
	if strings.TrimSpace(s.Label) == "" {
		t.Errorf("a switcher should carry a label for assistive technology")
	}
}
