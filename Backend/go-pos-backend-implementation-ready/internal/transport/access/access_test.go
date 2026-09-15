package access

import "testing"

func TestDefaultLevelForRole(t *testing.T) {
	cases := map[string]string{
		"owner":      LevelAdmin,
		"manager":    LevelManager,
		"cashier":    LevelCashier,
		"saas_admin": LevelAdmin,
		"guest":      LevelNone,
		"":           LevelNone,
	}
	for role, want := range cases {
		if got := DefaultLevelForRole(role); got != want {
			t.Errorf("DefaultLevelForRole(%q) = %q, want %q", role, got, want)
		}
	}
}

func TestValidLevel(t *testing.T) {
	for _, level := range []string{LevelNone, LevelCashier, LevelAdvanced, LevelManager, LevelAdmin} {
		if !ValidLevel(level) {
			t.Errorf("ValidLevel(%q) = false, want true", level)
		}
	}
	for _, level := range []string{"", "owner", "superuser", "cashiers"} {
		if ValidLevel(level) {
			t.Errorf("ValidLevel(%q) = true, want false", level)
		}
	}
}

func TestLevelDefaults(t *testing.T) {
	t.Run("none has no permissions", func(t *testing.T) {
		p := levelDefaults(LevelNone)
		if p.MaxDiscountPct != 0 || p.Discount || p.OpenSession || p.Refund || p.ChangeQty {
			t.Fatalf("none level unexpectedly permissive: %+v", p)
		}
	})
	t.Run("cashier can sell with 5% discount", func(t *testing.T) {
		p := levelDefaults(LevelCashier)
		if p.MaxDiscountPct != 5 || !p.ChangeQty || !p.Discount || !p.OpenSession || !p.CloseSession {
			t.Fatalf("cashier defaults wrong: %+v", p)
		}
		if p.Refund || p.NegativeStock || p.DeleteOrder || p.PriceChange {
			t.Fatalf("cashier should not refund/delete/price-change: %+v", p)
		}
	})
	t.Run("advanced adds line delete and payments", func(t *testing.T) {
		p := levelDefaults(LevelAdvanced)
		if p.MaxDiscountPct != 10 || !p.DeleteLine || !p.PaymentModification {
			t.Fatalf("advanced defaults wrong: %+v", p)
		}
		if p.Refund || p.DeleteOrder || p.NegativeQty {
			t.Fatalf("advanced should not refund/delete-order/negative-qty: %+v", p)
		}
	})
	t.Run("manager has refund and 50% discount", func(t *testing.T) {
		p := levelDefaults(LevelManager)
		if p.MaxDiscountPct != 50 || !p.Refund || !p.DeleteOrder || !p.NegativeQty || !p.PriceChange {
			t.Fatalf("manager defaults wrong: %+v", p)
		}
		if p.NegativeStock {
			t.Fatalf("manager should not sell negative stock by default: %+v", p)
		}
	})
	t.Run("admin is fully permissive", func(t *testing.T) {
		p := levelDefaults(LevelAdmin)
		if p.MaxDiscountPct != 100 || !p.Refund || !p.NegativeStock || !p.PriceChange ||
			!p.PaymentModification || !p.CloseSession || !p.OpenSession {
			t.Fatalf("admin defaults wrong: %+v", p)
		}
	})
}

func TestResolveWithoutRowFallsBackToRole(t *testing.T) {
	p := Resolve("cashier", nil)
	if p.AccessLevel != LevelCashier || p.MaxDiscountPct != 5 {
		t.Fatalf("cashier fallback wrong: %+v", p)
	}
	p = Resolve("owner", nil)
	if p.AccessLevel != LevelAdmin || !p.NegativeStock {
		t.Fatalf("owner fallback wrong: %+v", p)
	}
	p = Resolve("saas_admin", nil)
	if p.AccessLevel != LevelAdmin {
		t.Fatalf("saas_admin fallback wrong: %+v", p)
	}
}

func TestResolveLevelOverridesRole(t *testing.T) {
	// A cashier explicitly demoted to "none" cannot do anything.
	row := ScanRow(LevelNone, nil, false, false, false, false, false, false, false, false, false, false, false, false)
	p := Resolve("cashier", row)
	if p.AccessLevel != LevelNone || p.Discount || p.OpenSession || p.ChangeQty {
		t.Fatalf("none override wrong: %+v", p)
	}
}

func TestResolveCustomOverrides(t *testing.T) {
	row := ScanRow(LevelCashier, ptr(100), true, true, false, true, false, true, true, true, true, true, true, true)
	p := Resolve("cashier", row)
	if p.AccessLevel != LevelCashier {
		t.Fatalf("level wrong: %s", p.AccessLevel)
	}
	if !p.DeleteOrder || !p.Refund || !p.NegativeStock {
		t.Fatalf("custom flags not respected: %+v", p)
	}
	if p.DeleteLine {
		t.Fatalf("custom flag DeleteLine should stay false: %+v", p)
	}
	if p.NegativeQty {
		t.Fatalf("custom flag NegativeQty should stay false: %+v", p)
	}
}

func TestResolveCustomKeepsLevelFlagsWhenDisabled(t *testing.T) {
	// use_custom_permissions=false should resolve to level defaults even if
	// the stored row has stale/empty flags.
	row := ScanRow(LevelManager, nil, false, false, false, false, false, false, false, false, false, false, false, false)
	p := Resolve("manager", row)
	if !p.Refund || p.MaxDiscountPct != 50 || !p.DeleteOrder || !p.NegativeQty {
		t.Fatalf("level flags not applied when custom disabled: %+v", p)
	}
}

func TestResolvePerUserDiscountCapWithoutCustom(t *testing.T) {
	// A per-user max_discount_pct can narrow the level default even without
	// enabling custom permission flags.
	row := ScanRow(LevelCashier, ptr(2), false, false, false, false, false, false, false, false, false, false, false, false)
	p := Resolve("cashier", row)
	if p.MaxDiscountPct != 2 {
		t.Fatalf("per-user cap ignored: %+v", p)
	}
	if !p.Discount || !p.OpenSession {
		t.Fatalf("level flags lost: %+v", p)
	}
}

func TestEffectiveDiscountPct(t *testing.T) {
	cases := []struct {
		name   string
		perms  Permissions
		global int
		want   int
	}{
		{"no global limit", Permissions{AccessLevel: LevelManager, MaxDiscountPct: 50}, 0, 50},
		{"global caps user", Permissions{AccessLevel: LevelManager, MaxDiscountPct: 50}, 30, 30},
		{"global above user keeps user", Permissions{AccessLevel: LevelManager, MaxDiscountPct: 20}, 50, 20},
		{"user zero stays zero", Permissions{AccessLevel: LevelCashier, MaxDiscountPct: 0}, 100, 0},
		{"none level zero", Permissions{AccessLevel: LevelNone, MaxDiscountPct: 5}, 0, 5},
		{"admin always uncapped", Permissions{AccessLevel: LevelAdmin, MaxDiscountPct: 20}, 10, 100},
	}
	for _, c := range cases {
		if got := EffectiveDiscountPct(c.perms, c.global); got != c.want {
			t.Errorf("%s: EffectiveDiscountPct(%+v, %d) = %d, want %d", c.name, c.perms, c.global, got, c.want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	if got, err := ParseLevel(" manager "); err != nil || got != LevelManager {
		t.Fatalf("ParseLevel(manager) = %q, %v", got, err)
	}
	if _, err := ParseLevel("owner"); err == nil {
		t.Fatal("ParseLevel(owner) should be rejected")
	}
	if _, err := ParseLevel(""); err == nil {
		t.Fatal("ParseLevel(empty) should be rejected")
	}
}

func ptr(v int) *int { return &v }
