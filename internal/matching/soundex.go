package matching

import (
	"strings"
	"unicode"
)

// Soundex generates a soundex code for a name.
// This implementation is tailored for chess player names, including
// Slavic transliterations (Nimzovich = Nimsowitsch, Tal = Talj, etc.)
func Soundex(name string) string {
	if name == "" {
		return ""
	}

	// Convert to uppercase and clean
	name = strings.ToUpper(strings.TrimSpace(name))

	// Keep only letters
	var cleaned strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) {
			cleaned.WriteRune(r)
		}
	}

	s := cleaned.String()
	if s == "" {
		return ""
	}

	// Get the first letter
	result := string(s[0])

	// Soundex codes - modified for chess names
	// Group similar sounding consonants
	getCode := func(c byte) byte {
		switch c {
		case 'B', 'F', 'P', 'V', 'W':
			return '1'
		case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
			return '2'
		case 'D', 'T':
			return '3'
		case 'L':
			return '4'
		case 'M', 'N':
			return '5'
		case 'R':
			return '6'
		default:
			return '0' // vowels and others
		}
	}

	// Process remaining characters
	lastCode := getCode(s[0])
	for i := 1; i < len(s) && len(result) < 6; i++ {
		code := getCode(s[i])
		// Skip vowels (0) and consecutive same codes
		if code != '0' && code != lastCode {
			result += string(code)
		}
		if code != '0' {
			lastCode = code
		}
	}

	// Pad with zeros to length 6
	for len(result) < 6 {
		result += "0"
	}

	return result
}

// SoundexMatch checks if two names match via soundex.
func SoundexMatch(name1, name2 string) bool {
	return Soundex(name1) == Soundex(name2)
}
