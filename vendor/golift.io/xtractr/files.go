package xtractr

/* Code to find, write, move and delete files. */

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// XFile defines the data needed to extract an archive.
type XFile struct {
	// Path to archive being extracted.
	FilePath string
	// Folder to extract archive into.
	OutputDir string
	// Write files with this mode.
	FileMode os.FileMode
	// Write folders with this mode.
	DirMode os.FileMode
	// (RAR/7z) Archive password. Blank for none. Gets prepended to Passwords, below.
	Password string
	// (RAR/7z) Archive passwords (to try multiple).
	Passwords []string
}

// Filter is the input to find compressed files.
type Filter struct {
	// This is the path to search in for archives.
	Path string
	// Any files with this suffix are ignored. ie. ".7z" or ."iso"
	ExcludeSuffix Exclude
}

// Exclude represents an exclusion list.
type Exclude []string

// GetFileList returns all the files in a path.
// This is non-resursive and only returns files _in_ the base path provided.
// This is a helper method and only exposed for convenience. You do not have to call this.
func (x *Xtractr) GetFileList(path string) ([]string, error) {
	fileList, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading path %s: %w", path, err)
	}

	files := make([]string, len(fileList))
	for idx, file := range fileList {
		files[idx] = filepath.Join(path, file.Name())
	}

	return files, nil
}

// Difference returns all the strings that are in slice2 but not in slice1.
// Used to find new files in a file list from a path. ie. those we extracted.
// This is a helper method and only exposed for convenience. You do not have to call this.
func Difference(slice1 []string, slice2 []string) []string {
	diff := []string{}

	for _, s2p := range slice2 {
		var found bool

		for _, s1 := range slice1 {
			if s1 == s2p {
				found = true

				break
			}
		}

		if !found { // String not found, so it's a new string, add it to the diff.
			diff = append(diff, s2p)
		}
	}

	return diff
}

// Has returns true if the test has an excluded suffix.
func (e Exclude) Has(test string) bool {
	for _, exclude := range e {
		if strings.HasSuffix(test, strings.ToLower(exclude)) {
			return true
		}
	}

	return false
}

// FindCompressedFiles returns all the rar and zip files in a path. This attempts to grab
// only the first file in a multi-part archive. Sometimes there are multiple archives, so
// if the archive does not have "part" followed by a number in the name, then it will be
// considered an independent archive. Some packagers seem to use different naming schemes,
// so this will need to be updated as time progresses. So far it's working well.
// This is a helper method and only exposed for convenience. You do not have to call this.
func FindCompressedFiles(filter Filter) map[string][]string {
	dir, err := os.Open(filter.Path)
	if err != nil {
		return nil
	}
	defer dir.Close()

	if info, err := dir.Stat(); err != nil {
		return nil // unreadable folder?
	} else if l := strings.ToLower(filter.Path); !info.IsDir() &&
		(strings.HasSuffix(l, ".zip") || strings.HasSuffix(l, ".rar") || strings.HasSuffix(l, ".r00")) {
		return map[string][]string{filter.Path: {filter.Path}} // passed in an archive file; send it back out.
	}

	fileList, err := dir.Readdir(-1)
	if err != nil {
		return nil
	}

	// Check (save) if the current path has any rar files.
	// So we can ignore r00 if it does.
	r, _ := filepath.Glob(filepath.Join(filter.Path, "*.rar"))

	return getCompressedFiles(len(r) > 0, filter, fileList)
}

// getCompressedFiles checks file suffixes to find archives to decompress.
// This pays special attention to the widely accepted variance of rar formats.
func getCompressedFiles(hasrar bool, filter Filter, fileList []os.FileInfo) map[string][]string { //nolint:cyclop
	files := map[string][]string{}
	path := filter.Path

	for _, file := range fileList {
		switch lowerName := strings.ToLower(file.Name()); {
		case !file.IsDir() && filter.ExcludeSuffix.Has(lowerName):
			continue // file suffix is excluded.
		case lowerName == "" || lowerName[0] == '.':
			continue // ignore empty names and dot files/folders.
		case file.IsDir(): // Recurse.
			for k, v := range FindCompressedFiles(Filter{
				Path:          filepath.Join(path, file.Name()),
				ExcludeSuffix: filter.ExcludeSuffix,
			}) {
				files[k] = v
			}
		case strings.HasSuffix(lowerName, ".zip") || strings.HasSuffix(lowerName, ".tar") ||
			strings.HasSuffix(lowerName, ".tgz") || strings.HasSuffix(lowerName, ".gz") ||
			strings.HasSuffix(lowerName, ".bz2") || strings.HasSuffix(lowerName, ".7z") ||
			strings.HasSuffix(lowerName, ".7z.001") || strings.HasSuffix(lowerName, ".iso"):
			files[path] = append(files[path], filepath.Join(path, file.Name()))
		case strings.HasSuffix(lowerName, ".rar"):
			hasParts := regexp.MustCompile(`.*\.part[0-9]+\.rar$`)
			partOne := regexp.MustCompile(`.*\.part0*1\.rar$`)
			// Some archives are named poorly. Only return part01 or part001, not all.
			if !hasParts.Match([]byte(lowerName)) || partOne.Match([]byte(lowerName)) {
				files[path] = append(files[path], filepath.Join(path, file.Name()))
			}

		case !hasrar && strings.HasSuffix(lowerName, ".r00"):
			// Accept .r00 as the first archive file if no .rar files are present in the path.
			files[path] = append(files[path], filepath.Join(path, file.Name()))
		}
	}

	return files
}

// Extract calls the correct procedure for the type of file being extracted.
// Returns size of extracted data, list of extracted files, and/or error.
func (x *XFile) Extract() (int64, []string, []string, error) {
	return ExtractFile(x)
}

// ExtractFile calls the correct procedure for the type of file being extracted.
// Returns size of extracted data, list of extracted files, list of archives processed, and/or error.
func ExtractFile(xFile *XFile) (int64, []string, []string, error) { //nolint:cyclop
	var (
		size  int64
		files []string
		err   error
	)

	switch sName := strings.ToLower(xFile.FilePath); {
	case strings.HasSuffix(sName, ".rar"), strings.HasSuffix(sName, ".r00"):
		return ExtractRAR(xFile)
	case strings.HasSuffix(sName, ".7z"), strings.HasSuffix(sName, ".7z.001"):
		return Extract7z(xFile)
	case strings.HasSuffix(sName, ".zip"):
		size, files, err = ExtractZIP(xFile)
	case strings.HasSuffix(sName, ".tar.gz"), strings.HasSuffix(sName, ".tgz"):
		size, files, err = ExtractTarGzip(xFile)
	case strings.HasSuffix(sName, ".tar.bz2"), strings.HasSuffix(sName, ".tbz2"),
		strings.HasSuffix(sName, ".tbz"), strings.HasSuffix(sName, ".tar.bz"):
		size, files, err = ExtractTarBzip(xFile)
	case strings.HasSuffix(sName, ".bz"), strings.HasSuffix(sName, ".bz2"):
		size, files, err = ExtractBzip(xFile)
	case strings.HasSuffix(sName, ".gz"):
		size, files, err = ExtractGzip(xFile)
	case strings.HasSuffix(sName, ".iso"):
		size, files, err = ExtractISO(xFile)
	case strings.HasSuffix(sName, ".tar"):
		size, files, err = ExtractTar(xFile)
	default:
		return 0, nil, nil, fmt.Errorf("%w: %s", ErrUnknownArchiveType, xFile.FilePath)
	}

	return size, files, []string{xFile.FilePath}, err
}

// MoveFiles relocates files then removes the folder they were in.
// Returns the new file paths.
// This is a helper method and only exposed for convenience. You do not have to call this.
func (x *Xtractr) MoveFiles(fromPath string, toPath string, overwrite bool) ([]string, error) {
	var (
		files, err = x.GetFileList(fromPath)
		newFiles   = []string{}
		keepErr    error
	)

	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(toPath, x.config.DirMode); err != nil {
		return nil, fmt.Errorf("os.MkDirAll: %w", err)
	}

	for _, file := range files {
		var (
			newFile = filepath.Join(toPath, filepath.Base(file))
			_, err  = os.Stat(newFile)
			exists  = !os.IsNotExist(err)
		)

		if exists && !overwrite {
			x.config.Printf("Error: Renaming Temp File: %v to %v: (refusing to overwrite existing file)", file, newFile)
			// keep trying.
			continue
		}

		switch err = x.Rename(file, newFile); {
		case err != nil:
			keepErr = err
			x.config.Printf("Error: Renaming Temp File: %v to %v: %v", file, newFile, err)
		case exists:
			newFiles = append(newFiles, newFile)
			x.config.Debugf("Renamed Temp File: %v -> %v (overwrote existing file)", file, newFile)
		default:
			newFiles = append(newFiles, newFile)
			x.config.Debugf("Renamed Temp File: %v -> %v", file, newFile)
		}
	}

	x.DeleteFiles(fromPath)
	// Since this is the last step, we tried to rename all the files, bubble the
	// os.Rename error up, so it gets flagged as failed. It may have worked, but
	// it should get attention.
	return newFiles, keepErr
}

// DeleteFiles obliterates things and logs. Use with caution.
func (x *Xtractr) DeleteFiles(files ...string) {
	for _, file := range files {
		if err := os.RemoveAll(file); err != nil {
			x.config.Printf("Error: Deleting %v: %v", file, err)

			continue
		}

		x.config.Printf("Deleted (recursively): %s", file)
	}
}

// writeFile writes a file from an io reader, making sure all parent directories exist.
func writeFile(fpath string, fdata io.Reader, fMode, dMode os.FileMode) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(fpath), dMode); err != nil {
		return 0, fmt.Errorf("os.MkdirAll: %w", err)
	}

	fout, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fMode)
	if err != nil {
		return 0, fmt.Errorf("os.OpenFile: %w", err)
	}
	defer fout.Close()

	s, err := io.Copy(fout, fdata)
	if err != nil {
		return s, fmt.Errorf("copying io: %w", err)
	}

	return s, nil
}

// Rename is an attempt to deal with "invalid cross link device" on weird file systems.
func (x *Xtractr) Rename(oldpath, newpath string) error {
	if err := os.Rename(oldpath, newpath); err == nil {
		return nil
	}

	/* Rename failed, try copy. */

	oldFile, err := os.Open(oldpath) // do not forget to close this!
	if err != nil {
		return fmt.Errorf("os.Open(): %w", err)
	}

	newFile, err := os.OpenFile(newpath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, x.config.FileMode)
	if err != nil {
		oldFile.Close()
		return fmt.Errorf("os.OpenFile(): %w", err)
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, oldFile)
	oldFile.Close()

	if err != nil {
		return fmt.Errorf("io.Copy(): %w", err)
	}

	// The copy was successful, so now delete the original file
	_ = os.Remove(oldpath)

	return nil
}

// clean returns an absolute path for a file inside the OutputDir.
// If trim length is > 0, then the suffixes are trimmed, and filepath removed.
func (x *XFile) clean(filePath string, trim ...string) string {
	if len(trim) != 0 {
		filePath = filepath.Base(filePath)
		for _, suffix := range trim {
			filePath = strings.TrimSuffix(filePath, suffix)
		}
	}

	return filepath.Clean(filepath.Join(x.OutputDir, filePath))
}
