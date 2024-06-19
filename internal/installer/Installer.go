/*
 * Copyright (c) 2024 Bjoern Beier.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package installer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/begris-net/qtoolbox/internal/log"
	extract "github.com/codeclysm/extract/v3"
	"github.com/imroc/req/v3"
	"golift.io/xtractr"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

func (d *CandidateDownload) ensureDownloadPathExists(downloadPath string) {
	err := os.MkdirAll(downloadPath, 0750)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error creating cache path %s.", downloadPath), log.Logger.Args("err", err))
	}
}

func (d *CandidateDownload) CheckedDownload(destination string) (*req.Response, error) {
	d.ensureDownloadPathExists(path.Dir(destination))

	client := req.C().SetOutputDirectory(destination)

	callback := func(info req.DownloadInfo) {
		if info.Response.Response != nil {
			fmt.Fprintf(os.Stderr, "\rDownloading %s %s: (%d Kb/%d Kb) %.2f%%...",
				d.Candidate.Provider.Product, d.Candidate.DisplayName,
				info.DownloadedSize/1024, info.Response.ContentLength/1024,
				float64(info.DownloadedSize)/float64(info.Response.ContentLength)*100.0)
		}
	}
	candidateArchive := filepath.Base(d.DownloadUrl.String())
	get, err := client.R().SetOutputFile(candidateArchive).
		SetDownloadCallback(callback).SetRetryCount(2).
		OnAfterResponse(func(client *req.Client, resp *req.Response) error {
			fmt.Fprintf(os.Stderr, "Download finished\n")
			return nil
		}).Get(d.DownloadUrl.String())

	hasPreviousInstallation := false
	var previousInstallation string
	if stat, err := os.Stat(d.InstallPath); err == nil && stat.IsDir() {
		log.Logger.Info("Renaming previous installation.")
		previousInstallation = d.InstallPath + ".bac"
		os.Rename(d.InstallPath, previousInstallation)
		hasPreviousInstallation = true
	}

	candidateArchivePath := filepath.Join(destination, candidateArchive)

	//x := &xtractr.XFile{
	//	FilePath:  candidateArchivePath,
	//	OutputDir: filepath.FromSlash(d.InstallPath),
	//	DirMode:   0750,
	//}
	//
	//// size is how many bytes were written.
	//// files may be nil, but will contain any files written (even with an error).
	//extractSize, extractedFiles, processedArchives, err := x.Extract()

	data, err := os.Open(candidateArchivePath)
	if err != nil {
		log.Logger.Fatal("Error extracting candidate archive", log.Logger.Args("err", err))
	}
	defer data.Close()
	reader := bufio.NewReader(data)
	err = extract.Archive(context.TODO(), reader, filepath.FromSlash(d.InstallPath), nil)
	var extractedFiles []string
	var processedArchives []string
	var extractSize int
	if err == nil {
		extractedFiles = make([]string, 1)
		processedArchives = make([]string, 1)
		extractSize = 0
	}

	if err != nil && !errors.Is(err, xtractr.ErrUnknownArchiveType) {
		log.Logger.Warn("Error during extraction", log.Logger.Args("archive", candidateArchivePath, "installPath", d.InstallPath, "downloadPath", d.DownloadPath, "err", err))
	}

	if errors.Is(err, xtractr.ErrUnknownArchiveType) || err != nil {
		log.Logger.Debug("Assuming the download was not compressed and is an executable")
		// create candidate directory, as it was not created by the extraction process
		os.MkdirAll(d.InstallPath, 0750)

		candidateExecutable := path.Join(d.InstallPath, candidateArchive)

		copyBinary := func(src string, dst string) error {
			log.Logger.Info("Copying binary.", log.Logger.Args("src", src, "dst", dst))
			in, err := os.Open(src)
			if err != nil {
				return err
			}
			defer in.Close()

			out, err := os.Create(dst)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, in)
			if err != nil {
				return err
			}
			return nil
		}

		for retries := 0; retries < 3; retries++ {
			err = copyBinary(candidateArchivePath, candidateExecutable)
			if err == nil {
				break
			}
			log.Logger.Debug("Move binary", log.Logger.Args("archive-path", candidateArchivePath, "executeable", candidateExecutable, "try", retries, "err", err))
			time.Sleep(500 * time.Millisecond)
		}
		if d.FileMode != "" {
			mode, _ := strconv.ParseUint(d.FileMode, 10, 32)
			log.Logger.Debug("Setting file mode for executable", log.Logger.Args("file-mode", d.FileMode, "mode", mode, "err", err))
			os.Chmod(candidateExecutable, os.FileMode(mode))
		}
		// reset error, as this one was expected
		err = nil
	} else if err != nil && errors.Is(err, xtractr.ErrUnknownArchiveType) || extractedFiles == nil {
		log.Logger.Error(fmt.Sprintf("Installation of candidate %s failed.", d.Candidate.DisplayName), log.Logger.Args("err", err.Error()))
		if hasPreviousInstallation {
			log.Logger.Warn(fmt.Sprintf("Restoring previous installation of candidate %s.", d.Candidate.DisplayName))
			os.RemoveAll(d.InstallPath)
			os.Rename(previousInstallation, d.InstallPath)
			hasPreviousInstallation = false
		}
	} else {
		log.Logger.Info(fmt.Sprintf("Extraced %d files with %d bytes from archive %s.", len(extractedFiles), extractSize, processedArchives[0]))
		// checking if we got an unnecessary intermediate folder
		dirEntry, err := os.ReadDir(d.InstallPath)
		if err != nil {
			log.Logger.Fatal("Error during archive cleanup check.", log.Logger.Args("err", err.Error()))
		}
		if len(dirEntry) == 1 {
			entry := dirEntry[0]
			if entry.IsDir() {
				log.Logger.Info("Removing intermediate archive folder.")
				intermediateFolder := path.Join(d.InstallPath, entry.Name())

				subDirEntries, err := os.ReadDir(intermediateFolder)
				if err != nil {
					log.Logger.Warn("Could not read content of intermediate archive folder.", log.Logger.Args("err", err))
				}
				for _, subDirEntry := range subDirEntries {
					log.Logger.Debug("Moving sub entry of intermediate archive folder.", log.Logger.Args("subEntry", subDirEntry))
					err = os.Rename(path.Join(intermediateFolder, subDirEntry.Name()), path.Join(d.InstallPath, subDirEntry.Name()))
					if err != nil {
						log.Logger.Error("Error during move of sub entry.", log.Logger.Args("err", err))
					}
				}

				if err == nil {
					log.Logger.Debug("Deleting intermediate archive folder.")
					os.RemoveAll(intermediateFolder)
				}
			}
		}
	}
	if hasPreviousInstallation {
		log.Logger.Info("Removing previous installation.")
		os.RemoveAll(previousInstallation)
	}

	return get, err
}
