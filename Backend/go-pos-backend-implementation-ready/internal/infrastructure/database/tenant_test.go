package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWithTenantRejectsEmptyTenant(t *testing.T) {
	err := WithTenant(context.Background(), nil, uuid.Nil, func() error { return nil })
	if err == nil {
		t.Fatal("expected empty tenant to be rejected")
	}
}
