package purchases

import "testing"

func TestShouldAdmitNoLimitsAlwaysAdmits(t *testing.T) {
	admit, borrow := ShouldAdmit(OCRWindows{}, Unlimited, 0)
	if !admit || borrow {
		t.Fatalf("unlimited limits: admit=%v borrow=%v, want admit=true borrow=false", admit, borrow)
	}
	// Even past arbitrary "usage" with no configured limits: admitted.
	admit, borrow = ShouldAdmit(OCRWindows{Day: 9999, Week: 9999, Month: 9999}, Unlimited, 0)
	if !admit || borrow {
		t.Fatalf("unlimited limits w/ usage: admit=%v borrow=%v, want admit=true", admit, borrow)
	}
}

func TestShouldAdmitUnderLimit(t *testing.T) {
	lim := OCRWindows{Day: 10, Week: 50, Month: 200}
	cases := []OCRWindows{
		{Day: 0, Week: 0, Month: 0},
		{Day: 5, Week: 30, Month: 199},
		{Day: 9, Week: 49, Month: 198},
	}
	for _, used := range cases {
		admit, borrow := ShouldAdmit(used, lim, 0)
		if !admit || borrow {
			t.Fatalf("used=%+v limit=%+v credits=0: admit=%v borrow=%v, want admit=true", used, lim, admit, borrow)
		}
	}
}

func TestShouldAdmitAtExactCapNeedsCredits(t *testing.T) {
	// used == limit on a window counts as past the cap (>= is the trip).
	lim := OCRWindows{Day: 10, Week: 50, Month: 200}
	used := OCRWindows{Day: 10, Week: 40, Month: 150}
	admit, borrow := ShouldAdmit(used, lim, 0)
	if admit || borrow {
		t.Fatalf("day exactly at cap, credits=0: admit=%v borrow=%v, want admit=false", admit, borrow)
	}
	admit, borrow = ShouldAdmit(used, lim, 1)
	if !admit || !borrow {
		t.Fatalf("day exactly at cap, credits=1: admit=%v borrow=%v, want admit=true borrow=true", admit, borrow)
	}
}

func TestShouldAdmitOverWindowNoCreditsRejects(t *testing.T) {
	lim := OCRWindows{Day: 10, Week: 50, Month: 200}
	used := OCRWindows{Day: 11, Week: 40, Month: 150} // day over
	admit, _ := ShouldAdmit(used, lim, 0)
	if admit {
		t.Fatalf("day over cap, credits=0: want reject, got admit (used=%+v)", used)
	}
	// Over on two windows but third open still rejects when credits are gone.
	admit, _ = ShouldAdmit(OCRWindows{Day: 11, Week: 60, Month: 100}, lim, 0)
	if admit {
		t.Fatal("day+week over, credits=0: want reject")
	}
}

func TestShouldAdmitOverWindowBorrowsCredit(t *testing.T) {
	lim := OCRWindows{Day: 10, Week: 50, Month: 200}
	used := OCRWindows{Day: 11, Week: 51, Month: 201}
	admit, borrow := ShouldAdmit(used, lim, 3)
	if !admit || !borrow {
		t.Fatalf("over caps, credits=3: admit=%v borrow=%v, want admit=true borrow=true", admit, borrow)
	}
	admit, borrow = ShouldAdmit(used, lim, 1)
	if !admit || !borrow {
		t.Fatalf("over caps, credits=1: admit=%v borrow=%v, want admit=true borrow=true", admit, borrow)
	}
}

func TestShouldAdmitZeroLimitWindowInactive(t *testing.T) {
	// A window with limit 0 is unlimited and never blocks; another capped
	// window being over still triggers the borrow/reject logic.
	lim := OCRWindows{Day: 5, Week: 0, Month: 0}
	admit, _ := ShouldAdmit(OCRWindows{Day: 6, Week: 0, Month: 0}, lim, 0)
	if admit {
		t.Fatal("day capped+over, credits=0: want reject even with week/month open")
	}
	admit, borrow := ShouldAdmit(OCRWindows{Day: 6, Week: 0, Month: 0}, lim, 2)
	if !admit || !borrow {
		t.Fatalf("day capped+over, credits=2: admit=%v borrow=%v, want admit=true borrow=true", admit, borrow)
	}
}

func TestUnlimitedZeroValueIsSwiftAdmission(t *testing.T) {
	if Unlimited != (OCRWindows{}) {
		t.Fatalf("Unlimited must be the zero value, got %+v", Unlimited)
	}
}
