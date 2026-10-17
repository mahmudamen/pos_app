package server

import (
	"context"
	"testing"

	"github.com/example/pos-api/internal/testutil"
)

func TestPublicPlanCodeFiltersInternalTrial(t *testing.T) {
	if publicPlanCode("trial") {
		t.Error("the internal trial entitlement must not be shown on the pricing page")
	}
	for _, code := range []string{"starter", "business", "enterprise", "standard", ""} {
		if !publicPlanCode(code) {
			t.Errorf("publicPlanCode(%q) = false, want true", code)
		}
	}
}

// TestLoadPlanCardsExcludesTrialPlan proves the seeded trial entitlement never
// reaches the public plan catalog when the plans table is populated.
func TestLoadPlanCardsExcludesTrialPlan(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))

	cards := loadPlanCards(context.Background(), pool, "en")
	if len(cards) == 0 {
		t.Fatal("expected the seeded plan catalog to render")
	}
	for _, c := range cards {
		if c.Code == "trial" {
			t.Error("trial entitlement leaked into the public pricing page")
		}
	}
}
