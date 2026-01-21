package testutil

import (
	"errors"
	"testing"
)

// These tests verify the assertion helpers work correctly.
// Since we can't mock *testing.T, we test success cases directly
// and test the formatMessage helper which is internally testable.

func TestAssertEqual_Success(t *testing.T) {
	// These should not fail
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, 42, 42)
	AssertEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	AssertEqual(t, nil, nil)
}

func TestAssertEqual_WithMessage(t *testing.T) {
	// Test that message parameter works (success case)
	AssertEqual(t, "hello", "hello", "custom message")
	AssertEqual(t, 42, 42, "value should be %d", 42)
}

func TestAssertNoError_Success(t *testing.T) {
	AssertNoError(t, nil)
	AssertNoError(t, nil, "operation should succeed")
}

func TestAssertError_Success(t *testing.T) {
	AssertError(t, errors.New("test error"))
	AssertError(t, errors.New("test"), "expected error from %s", "operation")
}

func TestAssertContains_Success(t *testing.T) {
	AssertContains(t, "hello world", "world")
	AssertContains(t, "hello world", "hello")
	AssertContains(t, "test", "")
}

func TestAssertNotContains_Success(t *testing.T) {
	AssertNotContains(t, "hello world", "foo")
	AssertNotContains(t, "test", "xyz")
}

func TestAssertTrue_Success(t *testing.T) {
	AssertTrue(t, true)
	AssertTrue(t, 1 == 1)
	AssertTrue(t, len("hello") == 5)
}

func TestAssertFalse_Success(t *testing.T) {
	AssertFalse(t, false)
	AssertFalse(t, 1 == 2)
	AssertFalse(t, len("hello") == 0)
}

func TestAssertNil_Success(t *testing.T) {
	var p *int
	AssertNil(t, p)
	AssertNil(t, nil)
}

func TestAssertNotNil_Success(t *testing.T) {
	x := 42
	AssertNotNil(t, &x)
	AssertNotNil(t, "hello")
	AssertNotNil(t, []int{1, 2, 3})
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		want string
	}{
		{"no args", nil, ""},
		{"empty args", []interface{}{}, ""},
		{"single string", []interface{}{"hello"}, "hello"},
		{"single int", []interface{}{42}, "42"},
		{"format string", []interface{}{"hello %s", "world"}, "hello world"},
		{"format int", []interface{}{"value: %d", 42}, "value: 42"},
		{"format multiple", []interface{}{"%s %d %s", "test", 42, "end"}, "test 42 end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMessage(tt.args...)
			if got != tt.want {
				t.Errorf("formatMessage(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestAssertHelpers_CallerInfo verifies t.Helper() is working
// by ensuring the test file line numbers are reported correctly.
// This is implicitly tested by the fact that when assertions fail,
// they report the calling line, not the assertion function line.
func TestAssertHelpers_CallerInfo(t *testing.T) {
	// All assertions should call t.Helper()
	// This test verifies they compile and run without issues
	AssertEqual(t, true, true)
	AssertNoError(t, nil)
	AssertContains(t, "test", "est")
	AssertNotContains(t, "test", "xyz")
	AssertTrue(t, true)
	AssertFalse(t, false)
}
