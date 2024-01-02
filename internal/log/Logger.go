package log

import (
	"github.com/pterm/pterm"
	"os"
)

var (
	DebugLevel int8
	Logger     = pterm.DefaultLogger.WithTime(false).WithLevel(pterm.LogLevelTrace)
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
