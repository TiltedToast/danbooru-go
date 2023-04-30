package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	. "github.com/tiltedtoast/danbooru-go/types"
	"github.com/valyala/fasthttp"
	"go.uber.org/ratelimit"
)

func main() {
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

	options := ParseArgs(args)

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

	dl_bar := progressbar.NewOptions(len(posts),
		progressbar.OptionSetDescription("Downloading posts"),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetTheme(progressbar.Theme{
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
			DownloadPost(post, options, &client)
			if err := dl_bar.Add(1); err != nil {
				return
			}
			<-guard
		}(post)
	}
	wg.Wait()
}

// Download a post and saves it to a subfolder based on its rating
func DownloadPost(post Post, options InputOptions, client *fasthttp.Client) {
	url := post.FileURL

	if post.FileExt == "zip" && strings.Contains(post.LargeFileURL, ".webm") {
		url = post.LargeFileURL
		post.FileExt = "webm"
	}

	_, body, err := client.Get(nil, url)
	if err != nil {
		return
	}

	var subfolder string

	switch post.Rating {
	case "s":
		subfolder = "/sensitive"
	case "q":
		subfolder = "/questionable"
	case "e":
		subfolder = "/explicit"
	case "g":
		subfolder = "/general"
	default:
		subfolder = "/unknown"
	}

	// Create subfolder if it doesn't exist
	if _, err := os.Stat(fmt.Sprint("./" + options.OutputDir + subfolder)); os.IsNotExist(err) {
		newpath := filepath.Join(options.OutputDir, subfolder)
		if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
			return
		}
	}

	filename := strconv.Itoa(post.Score) + "_" + strconv.Itoa(post.ID) + "." + post.FileExt
	filename = filepath.Join(fmt.Sprint(options.OutputDir+subfolder), filename)

	if _, err := os.Stat(filename); err == nil {
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		return
	}

	defer file.Close()
	w := bufio.NewWriter(file)
	defer w.Flush()
	if _, err := w.Write(body); err != nil {
		fmt.Println("Error writing post:", post.ID)
		return
	}
}

// Loops over all pages and returns a list of all posts
//
// Uses a Progress Bar to show the progress to the user
func FetchPostsFromPage(tags []string, totalPageAmount int, options InputOptions, client *fasthttp.Client) []Post {
	var posts []Post

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)

	// API rate limit is 10 requests per second, can go higher but will likely
	// result in a lot of errors
	rl_per_second := 10

	if IsGoldMember() {
		rl_per_second = 20
	}
	rl := ratelimit.New(rl_per_second)

	pagesBar := progressbar.NewOptions(totalPageAmount,
		progressbar.OptionSetDescription("Fetching posts"),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[cyan]█[reset]",
			SaucerHead:    "[cyan]█[reset]",
			SaucerPadding: "[blue]░[reset]",
			BarStart:      "",
			BarEnd:        "",
		}))

	// Loops over all pages and adds them together into a single list
	// For them to be downloaded later
	for i := 1; i <= totalPageAmount; i++ {
		go func(currentPage int) {
			defer wg.Done()
			rl.Take()
			tagString := ""
			for _, tag := range tags {
				tagString += url.QueryEscape(tag) + "+"
			}

			postsUrl := fmt.Sprintf(
				"https://danbooru.donmai.us/posts.json?page=%d&tags=%s&limit=200&only=rating,file_url,id,score,file_ext,large_file_url",
				currentPage, tagString)

			// Credentials to get access to extra features for Danbooru Gold users
			if os.Getenv("DANBOORU_LOGIN") != "" && os.Getenv("DANBOORU_API_KEY") != "" {
				postsUrl += "&login=" + os.Getenv("DANBOORU_LOGIN") + "&api_key=" + os.Getenv("DANBOORU_API_KEY")
			}

			statusCode, body, err := client.Get(nil, postsUrl)
			if err != nil {
				return
			}

			// Parse JSON Response into list of posts
			var result []Post
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println("Error reading response,", statusCode)
				return
			}

			// User can exclude ratings via CLI flags
			for _, post := range result {
				if post.Rating == "s" && !options.Sensitive ||
					post.Rating == "q" && !options.Questionable ||
					post.Rating == "e" && !options.Explicit ||
					post.Rating == "g" && !options.General {
					continue
				}
				posts = append(posts, post)
			}

			_ = pagesBar.Add(1)
		}(i)
	}
	wg.Wait()
	return posts
}

// Get the total amount of pages for the given tags
//
// This is used to determine how many pages worth of posts to fetch
func GetTotalPages(tags []string) int {
	tagString := ""
	for _, tag := range tags {
		tagString += url.QueryEscape(tag) + "+"
	}
	pageUrl := fmt.Sprintf("https://danbooru.donmai.us/posts?tags=%s&limit=200", tagString)

	// Credentials to get access to extra features for Danbooru Gold users
	if os.Getenv("DANBOORU_LOGIN") != "" && os.Getenv("DANBOORU_API_KEY") != "" {
		pageUrl += "&login=" + os.Getenv("DANBOORU_LOGIN") + "&api_key=" + os.Getenv("DANBOORU_API_KEY")
	}

	resp, err := http.Get(pageUrl)
	if err != nil {
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	// Don't want the program to think there's 1 page worth of posts
	// When there's not a single post on the page
	noPosts := doc.Find("#posts > div > p").Text()
	if noPosts == "No posts found." {
		return 0
	}

	// Total page count at the bottom of the page as part of the pagination
	totalPages := doc.Find(".paginator-page.desktop-only").Last().Text()

	if totalPages == "" {
		return 1
	}

	// Need to work with an int instead of a string
	totalAmount, err := strconv.Atoi(totalPages)
	if err != nil {
		return 0
	}

	return totalAmount
}

func IsGoldMember() bool {
	if os.Getenv("DANBOORU_LOGIN") == "" || os.Getenv("DANBOORU_API_KEY") == "" {
		return false
	}
	loginName := os.Getenv("DANBOORU_LOGIN")
	apiKey := os.Getenv("DANBOORU_API_KEY")

	userRes, err := http.Get(fmt.Sprintf("https://danbooru.donmai.us/profile.json?login=%s&api_key=%s", loginName, apiKey))
	if err != nil {
		return false
	}

	userResData, err := io.ReadAll(userRes.Body)
	if err != nil {
		return false
	}

	var userJson User
	if err := json.Unmarshal(userResData, &userJson); err != nil {
		return false
	}

	return userJson.LevelString != "Member"
}

// Returns true if the given string is inside the slice
func Contains(slice []string, element string) bool {
	for _, a := range slice {
		if a == element {
			return true
		}
	}
	return false
}

func PrintHelpMessage() {
	fmt.Println("Usage:")
	fmt.Println("  danbooru-go [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help          print this help message and exit")
	fmt.Println("  -o, --output        output directory, defaults to 'output' subdirectory")
	fmt.Println("  -t, --tags          the specific tags you want to search for, split by \"+\" or spaces (required)")
	fmt.Println("  -g, --general       add this flag to filter out general images (everything but the other 3 categories)")
	fmt.Println("  -s, --sensitive     add this flag to filter out sensitive images")
	fmt.Println("  -q, --questionable  add this flag to filter out questionable images")
	fmt.Println("  -e, --explicit      add this flag to filter out clearly 18+ images")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-go")
}

func ParseArgs(args []string) InputOptions {
	options := InputOptions{
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
