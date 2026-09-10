package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeValidation, "name is required")
	if err.Code != CodeValidation {
		t.Errorf("Code: got %q, want %q", err.Code, CodeValidation)
	}
	if err.Message != "name is required" {
		t.Errorf("Message: got %q, want %q", err.Message, "name is required")
	}
	if err.Cause != nil {
		t.Errorf("Cause should be nil")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := Wrap(CodeDatabaseUnavailable, "unable to connect", cause)
	if err.Code != CodeDatabaseUnavailable {
		t.Errorf("Code: got %q, want %q", err.Code, CodeDatabaseUnavailable)
	}
	if err.Cause != cause {
		t.Errorf("Cause should be the original error")
	}
}

func TestAppErrorStringWithoutCause(t *testing.T) {
	err := New(CodeNotFound, "product not found")
	s := err.Error()
	if s != "not_found: product not found" {
		t.Errorf("Error(): got %q, want %q", s, "not_found: product not found")
	}
}

func TestAppErrorStringWithCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := Wrap(CodeDatabaseUnavailable, "unable to connect", cause)
	s := err.Error()
	if s != "database_unavailable: unable to connect: connection refused" {
		t.Errorf("Error(): got %q", s)
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying")
	err := Wrap(CodeInternal, "something failed", cause)
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should find the cause")
	}
}

func TestAppErrorUnwrapNil(t *testing.T) {
	err := New(CodeNotFound, "not found")
	if err.Unwrap() != nil {
		t.Fatal("Unwrap should return nil when Cause is nil")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []Code{
		CodeValidation, CodeUnauthorized, CodeForbidden,
		CodeNotFound, CodeConflict, CodeDatabaseUnavailable, CodeInternal,
	}
	for _, code := range codes {
		err := New(code, "test")
		if err.Code != code {
			t.Errorf("Code mismatch: got %q, want %q", err.Code, code)
		}
	}
}
