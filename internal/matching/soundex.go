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

	// Convert to uppercase and keep only letters
	var cleaned strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) {
			cleaned.WriteRune(r)
		}
	}

	s := cleaned.String()
	if s == "" {
		return ""
	}

	// Start with the first letter
	var result strings.Builder
	result.WriteByte(s[0])

	// Process remaining characters
	lastCode := soundexCode(s[0])
	for i := 1; i < len(s) && result.Len() < 6; i++ {
		code := soundexCode(s[i])
		// Skip vowels (0) and consecutive same codes
		if code != '0' && code != lastCode {
			result.WriteByte(code)
		}
		if code != '0' {
			lastCode = code
		}
	}

	// Pad with zeros to length 6
	for result.Len() < 6 {
		result.WriteByte('0')
	}

	return result.String()
}

// soundexCode returns the soundex code for a character.
// Groups similar sounding consonants together.
func soundexCode(c byte) byte {
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

// SoundexMatch checks if two names match via soundex.
func SoundexMatch(name1, name2 string) bool {
	return Soundex(name1) == Soundex(name2)
}
