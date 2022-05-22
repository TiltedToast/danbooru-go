package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type inputOptions struct {
	tags      []string
	outputDir string
	safe      bool
	risky     bool
	explicit  bool
}

func main() {
	args := os.Args[1:]

	if contains(args, "-h") || contains(args, "--help") || len(args) == 0 {
		printHelpMessage()
		return
	}

	opts := parseArgs(args)
	fmt.Println(opts)

	if len(opts.tags) == 0 {
		fmt.Println("No tags provided")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(opts.tags))

	for _, tag := range opts.tags {
		go func(currentTag string) {
			defer wg.Done()
			fmt.Println(currentTag, ":", getTotalPages(currentTag))
		}(tag)
	}
	wg.Wait()

	newpath := filepath.Join(".", opts.outputDir)
	err := os.MkdirAll(newpath, os.ModePerm)

	if err != nil {
		fmt.Println("Error creating directory, exiting")
		return
	}

}

func getTotalPages(tag string) int {
	url := fmt.Sprintf("https://danbooru.donmai.us/posts?tags=%s", url.QueryEscape(tag))

	resp, err := http.Get(url)
	if err != nil {
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return 0
	}

	totalPages := doc.Find(".paginator-page.desktop-only").Text()

	if totalPages != "" {
		totalPages = totalPages[4:]
	}

	totalAmount, err := strconv.Atoi(totalPages)

	if err != nil {
		return 0
	}

	return totalAmount

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
	fmt.Println("  -h, --help       print this help message and exit")
	fmt.Println("  -o, --output     output directory, defaults to 'output' subdirectory")
	fmt.Println("  -t, --tag        the specific tag(s) you want to download, separated by commas")
	fmt.Println("  -s, --safe       add this flag for safe images")
	fmt.Println("  -r, --risky      add this flag for suggestive images")
	fmt.Println("  -e, --explicit   add this flag for clearly 18+ images")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-dl-go")
}

func parseArgs(args []string) inputOptions {
	opts := inputOptions{}

	opts.outputDir = "output"
	opts.explicit = false
	opts.risky = false
	opts.safe = false
	opts.tags = []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if len(args) > i+1 {
				opts.outputDir = args[i+1]
			}
		case "-t", "--tag":
			if len(args) > i+1 {
				opts.tags = strings.Split(args[i+1], ",")
			}
		case "-r", "--risky":
			opts.risky = true
		case "-e", "--explicit":
			opts.explicit = true
		case "-s", "--safe":
			opts.safe = true
		}
	}

	return opts

}
