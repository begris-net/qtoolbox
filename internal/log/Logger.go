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

package log

import (
	"github.com/pterm/pterm"
	"os"
)

var (
	DebugLevel int8
	Logger     = pterm.DefaultLogger.WithTime(false).WithLevel(pterm.LogLevelTrace).WithWriter(os.Stderr)
	//Logger      = slog.New(slog.NewTextHandler(os.Stdout))
	//ErrorLogger = slog.New(slog.NewTextHandler(os.Stderr, nil))
)

func SetLogLevel(level int8) {
	/*
		qtb:
			-1 == off
			0 == fatal, error
			1 == warn
			2 == info
		    3 == debug
			4 == trace

		pterm:
		const (
			// LogLevelDisabled does never print.
			LogLevelDisabled LogLevel = iota
			// LogLevelTrace is the log level for traces.
			LogLevelTrace
			// LogLevelDebug is the log level for debug.
			LogLevelDebug
			// LogLevelInfo is the log level for info.
			LogLevelInfo
			// LogLevelWarn is the log level for warnings.
			LogLevelWarn
			// LogLevelError is the log level for errors.
			LogLevelError
			// LogLevelFatal is the log level for fatal errors.
			LogLevelFatal
			// LogLevelPrint is the log level for printing.
			LogLevelPrint
		)
	*/

	var ptermLevel pterm.LogLevel
	var writer = os.Stderr
	switch level {
	case 1:
		ptermLevel = pterm.LogLevelWarn
	case 2:
		ptermLevel = pterm.LogLevelInfo
	case 3:
		ptermLevel = pterm.LogLevelDebug
	case 4:
		ptermLevel = pterm.LogLevelTrace
	case -1:
		ptermLevel = pterm.LogLevelDisabled
	case 0:
	default:
		ptermLevel = pterm.LogLevelError
	}
	Logger.Level = ptermLevel
	Logger.Writer = writer
}
