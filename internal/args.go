package internal

import (
	"os"
	"strings"
)

type Args struct {
	Tags         []string
	OutputDir    string
	Sensitive    bool
	Questionable bool
	Explicit     bool
	General      bool
}

var OPTIONS = NewArgs()

// Parses the command line arguments and returns an Args struct
func NewArgs() Args {
	args := os.Args[1:]

	options := Args{
		OutputDir:    "output",
		Tags:         []string{},
		Sensitive:    true,
		General:      true,
		Questionable: true,
		Explicit:     true,
	}

	for i := range args {
		switch args[i] {
		case "-o", "--output":
			if len(args) > i+1 {
				options.OutputDir = args[i+1]
			}
		case "-t", "--tag":
			if len(args) > i+1 {
				if strings.Contains(args[i+1], "+") {
					options.Tags = strings.Split(args[i+1], "+")
				} else {
					// When manually selecting multiple tags on the website
					// they are separated by spaces
					options.Tags = strings.Split(args[i+1], " ")
				}
			}
		case "-q", "--questionable":
			options.Questionable = false
		case "-e", "--explicit":
			options.Explicit = false
		case "-s", "--sensitive":
			options.Sensitive = false
		case "-g", "--general":
			options.General = false
		}
	}

	return options
}
