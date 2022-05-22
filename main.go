package main

import (
	"fmt"
	"os"
)


type inputOptions struct {
	tag string
	outputDir string
	safe bool
	risky bool
	explicit bool
}


func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("No arguments provided")
		return
	}
	
	if contains(args, "-h") || contains(args, "--help") {
		printHelpMessage()
		return
	}
}



func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

func printHelpMessage() {
	fmt.Println("Usage:")
	fmt.Println("  danbooru-dl [options] <tags>")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help     print this help message and exit")
	fmt.Println("  -o, --output   output directory, defaults to 'output' subdirectory")
	fmt.Println("  -t, --tag      the specific tag(s) you want to download, separated by commas")
	fmt.Println("  -r, --risky    add this flag for suggestive images")
	fmt.Println("  -e, --explicit add this flag for clearly 18+ images")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-dl-go")
}


func parseArgs(args []string) {
	
}