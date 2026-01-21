// Package testutil provides shared test utilities for the pgn-extract-go project.
package testutil

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// AssertEqual compares got and want using cmp.Diff and reports differences.
// The msgAndArgs are optional and provide additional context if the assertion fails.
func AssertEqual(t *testing.T, got, want interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: mismatch (-want +got):\n%s", msg, diff)
		} else {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

// AssertNoError fails if err is not nil.
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: unexpected error: %v", msg, err)
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

// AssertError fails if err is nil when an error was expected.
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected error but got nil", msg)
		} else {
			t.Error("expected error but got nil")
		}
	}
}

// AssertContains fails if substr is not found in got.
func AssertContains(t *testing.T, got, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(got, substr) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: %q does not contain %q", msg, got, substr)
		} else {
			t.Errorf("%q does not contain %q", got, substr)
		}
	}
}

// AssertNotContains fails if substr is found in got.
func AssertNotContains(t *testing.T, got, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if strings.Contains(got, substr) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: %q should not contain %q", msg, got, substr)
		} else {
			t.Errorf("%q should not contain %q", got, substr)
		}
	}
}

// AssertTrue fails if condition is false.
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected true but got false", msg)
		} else {
			t.Error("expected true but got false")
		}
	}
}

// AssertFalse fails if condition is true.
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected false but got true", msg)
		} else {
			t.Error("expected false but got true")
		}
	}
}

// AssertNil fails if got is not nil.
// It handles both untyped nil and typed nil (e.g., (*int)(nil)).
func AssertNil(t *testing.T, got interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isNil(got) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected nil but got %v", msg, got)
		} else {
			t.Errorf("expected nil but got %v", got)
		}
	}
}

// AssertNotNil fails if got is nil.
// It handles both untyped nil and typed nil (e.g., (*int)(nil)).
func AssertNotNil(t *testing.T, got interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if isNil(got) {
		msg := formatMessage(msgAndArgs...)
		if msg != "" {
			t.Errorf("%s: expected non-nil value but got nil", msg)
		} else {
			t.Error("expected non-nil value but got nil")
		}
	}
}

// isNil checks if a value is nil, handling both untyped and typed nils.
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// formatMessage formats optional message arguments into a string.
func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		if s, ok := msgAndArgs[0].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", msgAndArgs[0])
	}
	if s, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(s, msgAndArgs[1:]...)
	}
	return fmt.Sprintf("%v", msgAndArgs[0])
}
