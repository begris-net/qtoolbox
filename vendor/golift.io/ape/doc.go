// Package ape reads and writes Monkey's Audio files.
//
// The container written here is the 3.99 descriptor layout. Compression
// levels match Monkey's Audio: 1000 fast, 2000 normal, 3000 high, 4000 extra
// high, 5000 insane.
//
// Encode turns interleaved little-endian PCM into a version 3990 APE file.
// NewEncoder writes that file from PCM chunks when the sample count is known.
// Decode reads version 3990 and file versions 3.93 through 3.99. NewDecoder
// yields one frame at a time. DecodeFile also follows an APL image link and
// returns that sample range.
package ape
