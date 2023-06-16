package app

import (
	"os"

	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var logger = log.Logger{
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
	if Contains(args, "--debug") {
		return log.DebugLevel
	} else if Contains(args, "--verbose") || Contains(args, "-v") {
		return log.TraceLevel
	} else {
		return log.ErrorLevel
	}
}
