package opsconsole

import (
	"testing"

	"m31labs.dev/fluffyui/state"
)

func TestZZResourceComputed(t *testing.T) {
	res := state.NewResource(func() (string, error) { return "value", nil })
	computed := state.NewComputed(func() string {
		snap := res.Get()
		if snap.Error != nil {
			return "err"
		}
		return "data:" + snap.Data + ",loading:" + boolStr(snap.Loading)
	}, res)
	t.Logf("initial=%q", computed.Get())
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
