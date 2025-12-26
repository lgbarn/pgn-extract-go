package main

import (
	"testing"
)

func TestParseElo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		// Valid ratings
		{"typical elo", "2500", 2500},
		{"low rating", "1200", 1200},
		{"high rating", "2850", 2850},
		{"beginner", "600", 600},
		{"four digits", "1500", 1500},

		// Edge cases - return 0
		{"empty string", "", 0},
		{"dash", "-", 0},
		{"question mark", "?", 0},

		// Invalid formats - return 0
		{"letters", "abc", 0},
		{"mixed", "12a5", 0},
		{"float", "2500.5", 0},
		{"negative", "-100", 0}, // strconv.Atoi returns -100, but this is valid negative int
		{"spaces", " 2500", 0},
		{"trailing spaces", "2500 ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElo(tt.input)
			// Note: negative numbers are valid per strconv.Atoi
			if tt.input == "-100" {
				if got != -100 {
					t.Errorf("parseElo(%q) = %d; want -100", tt.input, got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("parseElo(%q) = %d; want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Normal strings remain unchanged
		{"simple text", "Hello World", "Hello World"},
		{"with numbers", "Player123", "Player123"},
		{"with punctuation", "Fischer, Robert J.", "Fischer, Robert J."},

		// Control characters removed
		{"null byte", "Hello\x00World", "HelloWorld"},
		{"tab character", "Hello\tWorld", "HelloWorld"}, // tab (0x09) is < 32, removed
		{"newline", "Hello\nWorld", "HelloWorld"},
		{"carriage return", "Hello\rWorld", "HelloWorld"},
		{"bell", "Hello\x07World", "HelloWorld"},

		// DEL character (127) removed
		{"del character", "Hello\x7fWorld", "HelloWorld"},

		// Unicode preserved
		{"accented chars", "Müller", "Müller"},
		{"cyrillic", "Карпов", "Карпов"},
		{"chinese", "象棋", "象棋"},
		{"emoji", "♔♕♖", "♔♕♖"},

		// Edge cases
		{"empty string", "", ""},
		{"only spaces", "   ", "   "},
		{"only control chars", "\x00\x01\x02", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanString(tt.input)
			if got != tt.want {
				t.Errorf("cleanString(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchedCountOperations(t *testing.T) {
	// Reset to zero state first (since tests run in order)
	// We can't reset matchedCount directly, so we test incremental behavior

	initialCount := GetMatchedCount()

	IncrementMatchedCount()
	afterFirst := GetMatchedCount()
	if afterFirst != initialCount+1 {
		t.Errorf("after first increment: GetMatchedCount() = %d; want %d", afterFirst, initialCount+1)
	}

	IncrementMatchedCount()
	IncrementMatchedCount()
	afterThree := GetMatchedCount()
	if afterThree != initialCount+3 {
		t.Errorf("after three increments: GetMatchedCount() = %d; want %d", afterThree, initialCount+3)
	}
}
