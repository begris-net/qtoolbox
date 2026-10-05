package asar

import "errors"

// Package-level errors returned while reading an ASAR archive.
var (
	// ErrInvalidHeader is returned when the pickle header or JSON index is malformed.
	ErrInvalidHeader = errors.New("invalid asar header")
	// ErrTruncated is returned when the archive ends before the header or a file payload.
	ErrTruncated = errors.New("truncated asar archive")
	// ErrNotPacked is returned when File.Open is called on a directory, symlink, or unpacked entry.
	ErrNotPacked = errors.New("asar entry is not packed file data")
	// ErrHeaderTooLarge is returned when the header pickle exceeds maxHeaderPickle.
	ErrHeaderTooLarge = errors.New("asar header exceeds maximum size")
)
