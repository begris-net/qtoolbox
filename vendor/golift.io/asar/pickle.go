package asar

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf8"
)

// Chromium pickle used by Electron's ASAR header.
//
// A pickle is a uint32 payload-size prefix followed by that many payload bytes.
// Integers are little-endian. Each value is padded to a 4-byte boundary.
// The ASAR size pickle is always 8 bytes (payload size 4 plus one uint32).
// The header pickle holds one string: the JSON index.

const (
	pickleAlign      = 4
	pickleHeaderSize = 4
	sizePickleBytes  = 8
	maxHeaderPickle  = 64 << 20 // 64 MiB; the pickle length is also bounded by the file size.
)

// alignPickle rounds n up to the next multiple of 4.
func alignPickle(n int) int {
	if rem := n % pickleAlign; rem != 0 {
		return n + pickleAlign - rem
	}

	return n
}

// readFullAt reads len(buf) bytes at off from r.
func readFullAt(reader io.ReaderAt, off int64, buf []byte) error {
	_, err := io.ReadFull(io.NewSectionReader(reader, off, int64(len(buf))), buf)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTruncated, err)
	}

	return nil
}

// readSizePickle reads the leading 8-byte pickle and returns the header pickle length.
func readSizePickle(reader io.ReaderAt) (uint32, error) {
	var buf [sizePickleBytes]byte

	err := readFullAt(reader, 0, buf[:])
	if err != nil {
		return 0, err
	}

	payload := binary.LittleEndian.Uint32(buf[0:4])
	if payload != pickleHeaderSize {
		return 0, fmt.Errorf("%w: size pickle payload is %d, want 4", ErrInvalidHeader, payload)
	}

	return binary.LittleEndian.Uint32(buf[4:8]), nil
}

// readHeaderJSON reads the header pickle at offset 8 and returns its JSON string.
func readHeaderJSON(reader io.ReaderAt, pickleLen uint32, archiveSize int64) (string, error) {
	if pickleLen < pickleHeaderSize {
		return "", fmt.Errorf("%w: header pickle is %d bytes", ErrInvalidHeader, pickleLen)
	}

	if uint64(pickleLen) > maxHeaderPickle {
		return "", fmt.Errorf("%w: %d bytes", ErrHeaderTooLarge, pickleLen)
	}

	if int64(pickleLen) > archiveSize-sizePickleBytes {
		return "", fmt.Errorf("%w: header pickle %d exceeds archive size %d",
			ErrTruncated, pickleLen, archiveSize)
	}

	buf := make([]byte, pickleLen)

	err := readFullAt(reader, sizePickleBytes, buf)
	if err != nil {
		return "", err
	}

	payloadSize := binary.LittleEndian.Uint32(buf[0:4])
	headerSize := int(pickleLen) - int(payloadSize)

	if headerSize < pickleHeaderSize || headerSize%pickleAlign != 0 || headerSize > int(pickleLen) {
		return "", fmt.Errorf("%w: header pickle layout (header %d, payload %d)",
			ErrInvalidHeader, headerSize, payloadSize)
	}

	return readPickleString(buf[headerSize:])
}

// readPickleString reads one pickle string from payload: uint32 length, UTF-8 bytes, 4-byte padding.
func readPickleString(payload []byte) (string, error) {
	if len(payload) < pickleHeaderSize {
		return "", fmt.Errorf("%w: header string is truncated", ErrInvalidHeader)
	}

	strLen := binary.LittleEndian.Uint32(payload[0:4])
	if uint64(pickleHeaderSize)+uint64(strLen) > uint64(len(payload)) {
		return "", fmt.Errorf("%w: header string length %d", ErrInvalidHeader, strLen)
	}

	text := payload[pickleHeaderSize : pickleHeaderSize+strLen]
	if !utf8.Valid(text) {
		return "", fmt.Errorf("%w: header JSON is not valid UTF-8", ErrInvalidHeader)
	}

	return string(text), nil
}
