# asar

[![Go Reference](https://pkg.go.dev/badge/golift.io/asar.svg)](https://pkg.go.dev/golift.io/asar)
[![Go Report Card](https://goreportcard.com/badge/golift.io/asar)](https://goreportcard.com/report/golift.io/asar)
[![MIT License](http://img.shields.io/:license-mit-blue.svg)](https://github.com/golift/asar/blob/main/LICENSE)
[![discord](https://badgen.net/badge/icon/Discord?color=0011ff&label&icon=https://simpleicons.now.sh/discord/eee "GoLift Discord")](https://golift.io/discord)

Read-only Go reader for [Electron ASAR](https://github.com/electron/asar) archives.

An ASAR file is an 8-byte Chromium pickle, a JSON index pickle, and concatenated
uncompressed file bytes. This package parses that layout and returns `io.SectionReader`
values over packed files. It does not pack archives, apply transforms, or verify
integrity hashes.

Unpacked entries (`unpacked: true`) live in a sibling `{archive}.unpacked` directory.
The reader records the flag; callers copy those files themselves.

```go
package main

import (
	"fmt"
	"io"
	"os"

	"golift.io/asar"
)

func main() {
	reader, err := asar.Open("app.asar")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	for _, file := range reader.Files {
		if !file.Packed() {
			continue
		}

		src, err := file.Open()
		if err != nil {
			panic(err)
		}

		fmt.Println(file.Name, file.Size)
		_, _ = io.Copy(os.Stdout, src)
	}
}
```
