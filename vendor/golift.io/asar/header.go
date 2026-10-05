package asar

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// JSON index walk. Matches Electron's validateHeader in src/disk.ts:
// entry names may not contain a slash or backslash, and may not be "." or "..".

func walkHeader(raw string, dataOffset, archiveSize int64) ([]*File, error) {
	var root jsonNode

	err := json.Unmarshal([]byte(raw), &root)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHeader, err)
	}

	if root.Files == nil {
		return nil, fmt.Errorf("%w: root header must be a directory with a files property", ErrInvalidHeader)
	}

	files := make([]*File, 0, len(root.Files))

	err = walkDir(&files, "", root.Files, dataOffset, archiveSize)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// jsonNode is one object in the ASAR files tree.
type jsonNode struct {
	Files      map[string]json.RawMessage `json:"files"`
	Offset     string                     `json:"offset"`
	Size       *json.Number               `json:"size"`
	Unpacked   *bool                      `json:"unpacked"`
	Executable *bool                      `json:"executable"`
	Link       string                     `json:"link"`
	Integrity  *Integrity                 `json:"integrity"`
}

func walkDir(out *[]*File, prefix string, children map[string]json.RawMessage, dataOffset, archiveSize int64) error {
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		err := validateEntryName(name)
		if err != nil {
			return err
		}

		path := name
		if prefix != "" {
			path = prefix + "/" + name
		}

		node, err := decodeNode(children[name], path)
		if err != nil {
			return err
		}

		file, nested, err := nodeToFile(path, node, dataOffset, archiveSize)
		if err != nil {
			return err
		}

		*out = append(*out, file)

		if nested != nil {
			err = walkDir(out, path, nested, dataOffset, archiveSize)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeNode(raw json.RawMessage, path string) (*jsonNode, error) {
	var node jsonNode

	err := json.Unmarshal(raw, &node)
	if err != nil {
		return nil, fmt.Errorf("%w: entry %q: %w", ErrInvalidHeader, path, err)
	}

	return &node, nil
}

func nodeToFile(path string, node *jsonNode, dataOffset, archiveSize int64) (*File, map[string]json.RawMessage, error) {
	switch {
	case node.Link != "":
		return &File{Name: path, Link: node.Link, Unpacked: boolVal(node.Unpacked)}, nil, nil
	case node.Files != nil:
		return &File{Name: path, Unpacked: boolVal(node.Unpacked), dir: true}, node.Files, nil
	case boolVal(node.Unpacked) && node.Size != nil:
		// Electron validates offset-bearing entries first, then honors unpacked
		// at read time. Treat unpacked before offset so Open does not return
		// archive bytes for a sibling that lives in {archive}.unpacked.
		return unpackedFile(path, node)
	case node.Offset != "":
		return packedFile(path, node, dataOffset, archiveSize)
	default:
		return nil, nil, fmt.Errorf("%w: entry %q is not a directory, file, or link", ErrInvalidHeader, path)
	}
}

func unpackedFile(path string, node *jsonNode) (*File, map[string]json.RawMessage, error) {
	size, err := parseSize(path, *node.Size)
	if err != nil {
		return nil, nil, err
	}

	return &File{
		Name:       path,
		Size:       size,
		Unpacked:   true,
		Executable: boolVal(node.Executable),
		Integrity:  node.Integrity,
	}, nil, nil
}

func packedFile(path string, node *jsonNode, dataOffset, archiveSize int64) (*File, map[string]json.RawMessage, error) {
	if !isDecimal(node.Offset) {
		return nil, nil, fmt.Errorf("%w: entry %q offset %q is not a numeric string", ErrInvalidHeader, path, node.Offset)
	}

	rel, err := strconv.ParseUint(node.Offset, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: entry %q offset: %w", ErrInvalidHeader, path, err)
	}

	if node.Size == nil {
		return nil, nil, fmt.Errorf("%w: entry %q is missing size", ErrInvalidHeader, path)
	}

	size, err := parseSize(path, *node.Size)
	if err != nil {
		return nil, nil, err
	}

	abs, fits := add64(uint64(dataOffset), rel)
	if !fits {
		return nil, nil, fmt.Errorf("%w: entry %q offset overflows", ErrInvalidHeader, path)
	}

	end, fits := add64(abs, uint64(size))
	if !fits || end > uint64(archiveSize) {
		return nil, nil, fmt.Errorf("%w: entry %q extends past the archive", ErrTruncated, path)
	}

	return &File{
		Name:       path,
		Offset:     int64(abs),
		Size:       size,
		Executable: boolVal(node.Executable),
		Integrity:  node.Integrity,
	}, nil, nil
}

func parseSize(path string, num json.Number) (int64, error) {
	size, err := num.Int64()
	if err != nil {
		return 0, fmt.Errorf("%w: entry %q size: %w", ErrInvalidHeader, path, err)
	}

	if size < 0 {
		return 0, fmt.Errorf("%w: entry %q size is negative", ErrInvalidHeader, path)
	}

	return size, nil
}

func validateEntryName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%w: invalid entry name %q", ErrInvalidHeader, name)
	}

	return nil
}

func boolVal(v *bool) bool {
	return v != nil && *v
}

func isDecimal(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func add64(a, b uint64) (uint64, bool) {
	sum := a + b
	if sum < a {
		return 0, false
	}

	return sum, true
}
