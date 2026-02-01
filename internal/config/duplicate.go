package config

import "io"

// DuplicateConfig holds settings for duplicate game detection.
type DuplicateConfig struct {
	// Suppress enables duplicate suppression
	Suppress bool

	// SuppressOriginals suppresses original games when duplicates are found
	SuppressOriginals bool

	// FuzzyMatch enables fuzzy duplicate matching
	FuzzyMatch bool

	// FuzzyDepth is the depth for fuzzy matching comparison
	FuzzyDepth uint

	// UseVirtualHashTable uses virtual hash table for duplicate detection
	UseVirtualHashTable bool

	// DuplicateFile is the output stream for duplicate games
	DuplicateFile io.Writer

	// MaxCapacity is the maximum number of hash table entries for duplicate detection
	// 0 means unlimited capacity
	MaxCapacity int
}

// NewDuplicateConfig creates a DuplicateConfig with default values.
// All fields use Go zero values (false, 0, nil).
func NewDuplicateConfig() *DuplicateConfig {
	return &DuplicateConfig{}
}
