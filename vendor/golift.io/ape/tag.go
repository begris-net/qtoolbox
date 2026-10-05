package ape

import (
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	tagVersion     = 2000
	tagFooterBytes = 32
)

func writeTag(dst io.Writer, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}

	slices.Sort(names)

	var fields []byte

	for _, name := range names {
		if strings.ContainsRune(name, 0) {
			return errBadTag
		}

		fields = append(fields, tagField(name, tags[name])...)
	}

	footer := make([]byte, tagFooterBytes)
	copy(footer, "APETAGEX")
	binary.LittleEndian.PutUint32(footer[8:], tagVersion)
	binary.LittleEndian.PutUint32(footer[12:], uint32(len(fields)+tagFooterBytes))
	binary.LittleEndian.PutUint32(footer[16:], uint32(len(names)))

	_, err := dst.Write(append(fields, footer...))
	if err != nil {
		return fmt.Errorf("ape: writing tag: %w", err)
	}

	return nil
}

func tagField(name, value string) []byte {
	field := make([]byte, 8+len(name)+1+len(value))
	binary.LittleEndian.PutUint32(field[0:], uint32(len(value)))
	copy(field[8:], name)
	copy(field[8+len(name)+1:], value)

	return field
}
