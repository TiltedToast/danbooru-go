package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/joho/godotenv"
	pb "github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"

	. "github.com/tiltedtoast/danbooru-go/types"
)

func RunApp() {
	args := os.Args[1:]

	exe, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path")
		return
	}

	exePath := filepath.Dir(exe)

	if _, err := os.Stat(fmt.Sprintf("%s/.env", exePath)); err == nil {
		envErr := godotenv.Load(fmt.Sprintf("%s/.env", exePath))
		if envErr != nil {
			fmt.Println("Error loading .env file")
			return
		}
	}

	if Contains(args, "-h") || Contains(args, "--help") || len(args) == 0 {
		PrintHelpMessage()
		return
	}

	options := NewArgs()

	if len(options.Tags) == 0 {
		fmt.Println("No tags provided")
		return
	}

	totalPages := GetTotalPages(options.Tags)

	if totalPages == 0 {
		fmt.Println("No posts found")
		return
	}

	client := fasthttp.Client{
		MaxConnsPerHost: 1000,
		Dial:            fasthttp.Dial,
	}

	posts := FetchPostsFromPage(options.Tags, totalPages, options, &client)

	newpath := filepath.Join(".", options.OutputDir)
	if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
		fmt.Println("Error creating directory, exiting")
		return
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

	maxGoroutines := runtime.NumCPU() * 3
	guard := make(chan int, maxGoroutines)

	// Make sure there's not too many goroutines running at once
	// This would cause cause extremely high CPU usage / program crashes
	for _, post := range posts {
		guard <- 1
		go func(post Post) {
			defer wg.Done()
			post.Download(options, &client)
			if err := dl_bar.Add(1); err != nil {
				return
			}
			<-guard
		}(post)
	}
	wg.Wait()
}

func NewArgs(args []string) {
	panic("unimplemented")
}
