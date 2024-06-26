package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/joho/godotenv"
	pb "github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"

	"github.com/tiltedtoast/danbooru-go/internal"
	log "github.com/tiltedtoast/danbooru-go/internal/logger"
	"github.com/tiltedtoast/danbooru-go/internal/options"
)

var (
	opts   = options.GetOptions()
	logger = log.GetLogger()
)

func main() {
	args := os.Args[1:]

	exe, err := os.Executable()
	if err != nil {
		logger.Fatal("Error getting executable path")
	}

	exePath := filepath.Dir(exe)

	if _, err := os.Stat(fmt.Sprintf("%s/.env", exePath)); err == nil {
		envErr := godotenv.Load(fmt.Sprintf("%s/.env", exePath))
		if envErr != nil {
			logger.Fatal("Error loading .env file")
		}
	}

	if slices.Contains(args, "-h") || slices.Contains(args, "--help") || len(args) == 0 {
		internal.PrintHelpMessage()
		return
	}

	logger.Trace(fmt.Sprintf("Arguments: %v", opts))

	if len(opts.Tags) == 0 {
		logger.Fatal("No tags provided")
	}

	totalPages := internal.GetTotalPages(opts.Tags)

	logger.Trace(fmt.Sprintf("Total pages: %d", totalPages))

	if totalPages == 0 {
		logger.Fatal("No posts found")
	}

	client := fasthttp.Client{
		MaxConnsPerHost: 1000,
		Dial:            fasthttp.Dial,
	}

	posts := internal.FetchPostsFromPage(totalPages, &client)

	logger.Trace(fmt.Sprintf("Total posts: %d", len(posts)))

	newpath := filepath.Join(".", opts.OutputDir)
	if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
		logger.Fatal("Error creating directory, exiting")
	}

	dl_bar := pb.NewOptions(len(posts),
		pb.OptionSetDescription("Downloading posts"),
		pb.OptionEnableColorCodes(true),
		pb.OptionFullWidth(),
		pb.OptionShowCount(),
		pb.OptionSetPredictTime(true),
		pb.OptionShowElapsedTimeOnFinish(),
		pb.OptionSetTheme(pb.Theme{
			Saucer:        "[cyan]█[reset]",
			SaucerHead:    "[cyan]█[reset]",
			SaucerPadding: "[blue]░[reset]",
			BarStart:      "",
			BarEnd:        "",
		}))

	wg := sync.WaitGroup{}
	wg.Add(len(posts))

	indices := internal.SegmentSlice(posts, runtime.NumCPU())

	for _, index := range indices {
		go func(start, end int) {
			for _, post := range posts[start:end] {
				defer wg.Done()
				post.Download(&client)
				if err := dl_bar.Add(1); err != nil {
					continue
				}
			}
		}(index[0], index[1])
	}
	wg.Wait()
}
