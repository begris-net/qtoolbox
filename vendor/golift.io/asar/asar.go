// Package asar reads Electron ASAR archives.
//
// An ASAR file is an 8-byte size pickle, a JSON index pickle, and concatenated
// uncompressed file bytes. Offsets in the index are relative to the start of those
// bytes. Files with identical contents may share one offset. Entries marked unpacked
// live in a sibling directory named {archive}.unpacked; this package only records
// that flag. Integrity hashes are parsed and not verified.
package asar

import (
	"fmt"
	"io"
	"os"
)

// Integrity is the optional SHA-256 checksum Electron stores on a file entry.
type Integrity struct {
	Algorithm string   `json:"algorithm"`
	Hash      string   `json:"hash"`
	BlockSize int      `json:"blockSize"`
	Blocks    []string `json:"blocks"`
}

// File is one directory, symlink, packed file, or unpacked file in the index.
// Directories are listed before their children. Name uses forward slashes and
// has no leading slash.
type File struct {
	Name       string
	Offset     int64 // Absolute archive offset of packed data. Zero when Unpacked, a directory, or a link.
	Size       int64
	Executable bool
	Unpacked   bool
	Link       string // Package-relative symlink target when this entry is a link.
	Integrity  *Integrity

	dir bool
	ra  io.ReaderAt
}

// IsDir reports whether f is a directory node.
func (f *File) IsDir() bool {
	return f != nil && f.dir
}

// IsLink reports whether f is a symlink.
func (f *File) IsLink() bool {
	return f != nil && f.Link != ""
}

// Packed reports whether f is stored in the archive body.
func (f *File) Packed() bool {
	return f != nil && !f.dir && f.Link == "" && !f.Unpacked
}

// Open returns a SectionReader over the packed bytes. Directories, symlinks, and
// unpacked entries return ErrNotPacked.
func (f *File) Open() (*io.SectionReader, error) {
	if f == nil || f.ra == nil || !f.Packed() {
		return nil, ErrNotPacked
	}

	return io.NewSectionReader(f.ra, f.Offset, f.Size), nil
}

// Reader is a parsed ASAR archive. Files is the depth-first index.
type Reader struct {
	Files      []*File
	HeaderSize int64 // Length of the header pickle, not including the 8-byte size pickle.
	DataOffset int64 // Absolute offset of the first packed byte (8 + HeaderSize).

	file *os.File
}

// Open opens path as an ASAR archive. The caller must Close the Reader.
func Open(path string) (*Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening asar: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stat asar: %w", err)
	}

	reader, err := NewReader(file, stat.Size())
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	reader.file = file

	return reader, nil
}

// NewReader parses an ASAR image from reader, which must cover size bytes.
func NewReader(reader io.ReaderAt, size int64) (*Reader, error) {
	if size < sizePickleBytes {
		return nil, fmt.Errorf("%w: archive is %d bytes", ErrTruncated, size)
	}

	pickleLen, err := readSizePickle(reader)
	if err != nil {
		return nil, err
	}

	jsonIndex, err := readHeaderJSON(reader, pickleLen, size)
	if err != nil {
		return nil, err
	}

	dataOffset := int64(sizePickleBytes) + int64(pickleLen)

	files, err := walkHeader(jsonIndex, dataOffset, size)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		file.ra = reader
	}

	return &Reader{
		Files:      files,
		HeaderSize: int64(pickleLen),
		DataOffset: dataOffset,
	}, nil
}

// Close closes the file opened by Open. It is a no-op for Readers from NewReader.
func (r *Reader) Close() error {
	if r == nil || r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil

	if err != nil {
		return fmt.Errorf("closing asar: %w", err)
	}

	return nil
}
