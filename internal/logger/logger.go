package logger

import (
	"os"
	"slices"

	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func GetLogger() *log.Logger {
	return &Logger
}

var Logger = log.Logger{
	Out: os.Stderr,
	Formatter: &prefixed.TextFormatter{
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	},
	Level: determine_level(os.Args[1:]),
}

func determine_level(args []string) log.Level {
	if slices.Contains(args, "--debug") {
		return log.DebugLevel
	} else if slices.Contains(args, "--verbose") || slices.Contains(args, "-v") {
		return log.TraceLevel
	} else {
		return log.ErrorLevel
	}
}
