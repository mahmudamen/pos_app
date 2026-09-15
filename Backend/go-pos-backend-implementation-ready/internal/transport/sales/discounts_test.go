package sales

import (
	"errors"
	"testing"

	"github.com/example/pos-api/internal/transport/access"
)

func cashierPerms() access.Permissions {
	return access.Resolve("cashier", nil)
}

func managerPerms() access.Permissions {
	return access.Resolve("manager", nil)
}

func adminPerms() access.Permissions {
	return access.Resolve("owner", nil)
}

func TestEvaluateDiscountRejectsInvalidInputs(t *testing.T) {
	perms := cashierPerms()
	cases := []struct {
		name     string
		subtotal int64
		discount int64
		wantErr  error
	}{
		{"negative discount", 1000, -1, ErrNegativeDiscount},
		{"discount exceeds subtotal", 1000, 1001, ErrDiscountExceedsSubtotal},
	}
	for _, tc := range cases {
		if _, err := EvaluateDiscount(perms, DiscountModeCap, 0, tc.subtotal, tc.discount); !errors.Is(err, tc.wantErr) {
			t.Errorf("%s: got err %v, want %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestEvaluateDiscountNoDiscountPermission(t *testing.T) {
	perms := access.Resolve("owner", &access.Row{AccessLevel: access.LevelNone})
	if _, err := EvaluateDiscount(perms, DiscountModeCap, 0, 1000, 100); !errors.Is(err, ErrDiscountForbidden) {
		t.Fatalf("none-level user should be forbidden, got %v", err)
	}
}

func TestEvaluateDiscountZeroKeepsTotal(t *testing.T) {
	d, err := EvaluateDiscount(cashierPerms(), DiscountModeBlock, 0, 1000, 0)
	if err != nil || !d.Allow || d.NeedsManager || d.TotalAfter != 1000 {
		t.Fatalf("zero discount must pass through: %+v err %v", d, err)
	}
}

func TestEvaluateDiscountAdminAlwaysAllowed(t *testing.T) {
	d, err := EvaluateDiscount(adminPerms(), DiscountModeBlock, 10, 1000, 500)
	if err != nil || !d.Allow || d.NeedsManager || d.TotalAfter != 500 {
		t.Fatalf("admin must bypass policy: %+v err %v", d, err)
	}
}

func TestEvaluateDiscountWithinLimit(t *testing.T) {
	d, err := EvaluateDiscount(cashierPerms(), DiscountModeBlock, 0, 1000, 50)
	if err != nil || !d.Allow || d.NeedsManager || d.TotalAfter != 950 {
		t.Fatalf("within-limit discount must pass: %+v err %v", d, err)
	}
}

func TestEvaluateDiscountCapModeClamps(t *testing.T) {
	d, err := EvaluateDiscount(cashierPerms(), DiscountModeCap, 0, 1000, 100)
	if err != nil {
		t.Fatalf("cap mode should clamp, not error: %v", err)
	}
	if !d.Allow || d.NeedsManager || !d.Capped {
		t.Fatalf("expected a capped allow: %+v", d)
	}
	if d.TotalAfter != 950 { // 5% is the cashier ceiling
		t.Errorf("capped total = %d, want 950", d.TotalAfter)
	}
	if d.AppliedDiscount != 50 {
		t.Errorf("applied discount = %d, want 50", d.AppliedDiscount)
	}
}

func TestEvaluateDiscountGlobalCapWinsAgainstLevel(t *testing.T) {
	// Manager level allows 50%; a tenant-wide 10% ceiling narrows it.
	d, err := EvaluateDiscount(managerPerms(), DiscountModeCap, 10, 1000, 300)
	if err != nil || !d.Capped {
		t.Fatalf("expected capped decision: %+v err %v", d, err)
	}
	if d.TotalAfter != 900 {
		t.Errorf("capped total = %d, want 900 (10%% global)", d.TotalAfter)
	}
}

func TestEvaluateDiscountBlockModeNeedsManager(t *testing.T) {
	d, err := EvaluateDiscount(cashierPerms(), DiscountModeBlock, 0, 1000, 200)
	if err != nil {
		t.Fatalf("block mode should defer to manager, not error: %v", err)
	}
	if d.Allow || !d.NeedsManager {
		t.Fatalf("expected NeedsManager, got %+v", d)
	}
	if d.TotalAfter != 800 {
		t.Errorf("manager-approved total = %d, want 800", d.TotalAfter)
	}
}

func TestEvaluateDiscountWarnModeAllowsWithNotice(t *testing.T) {
	d, err := EvaluateDiscount(cashierPerms(), DiscountModeWarn, 0, 1000, 200)
	if err != nil || !d.Allow || d.NeedsManager {
		t.Fatalf("warn mode must allow: %+v err %v", d, err)
	}
	if d.Warning == "" {
		t.Fatal("warn mode should carry a warning")
	}
	if d.TotalAfter != 800 || d.AppliedDiscount != 200 {
		t.Errorf("warn total/applied = %d/%d, want 800/200", d.TotalAfter, d.AppliedDiscount)
	}
}

func TestParseDiscountMode(t *testing.T) {
	if got := ParseDiscountMode("block"); got != DiscountModeBlock {
		t.Errorf("block parsed as %q", got)
	}
	if got := ParseDiscountMode("warn"); got != DiscountModeWarn {
		t.Errorf("warn parsed as %q", got)
	}
	for _, v := range []string{"cap", "", "spam"} {
		if got := ParseDiscountMode(v); got != DiscountModeCap {
			t.Errorf("%q parsed as %q, want cap", v, got)
		}
	}
}
