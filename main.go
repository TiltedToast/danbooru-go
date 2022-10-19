package main

import (
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
	"go.uber.org/ratelimit"
)

type inputOptions struct {
	tags         []string
	outputDir    string
	sensitive    bool
	questionable bool
	explicit     bool
	general      bool
}

type Post struct {
	ID      int    `json:"id"`
	Score   int    `json:"score"`
	Rating  string `json:"rating"`
	FileExt string `json:"file_ext"`
	FileURL string `json:"file_url"`
}

func main() {
	args := os.Args[1:]
	godotenv.Load()

	if contains(args, "-h") || contains(args, "--help") || len(args) == 0 {
		printHelpMessage()
		return
	}

	options := parseArgs(args)

	if len(options.tags) == 0 {
		fmt.Println("No tags provided")
		return
	}

	totalPages := getTotalPages(options.tags)

	if totalPages == 0 {
		fmt.Println("No posts found")
		return
	}
	

	posts := fetchPostsFromPage(options.tags, totalPages, options)

	newpath := filepath.Join(".", options.outputDir)
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
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
			Saucer:        "[cyan]#[reset]",
			SaucerHead:    "[cyan]=[reset]",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
		}))

	wg := sync.WaitGroup{}
	wg.Add(len(posts))

	maxGoroutines := runtime.NumCPU()
	guard := make(chan int, maxGoroutines)

	// Make sure there's not too many goroutines running at once
	// This would cause cause extremely high CPU usage / program crashes
	for _, post := range posts {
		guard <- 1
		go func(post Post) {
			defer wg.Done()
			downloadPost(post, options)
			err := dl_bar.Add(1)
			if err != nil {
				return
			}
			<-guard
		}(post)
	}
	wg.Wait()
}

// Download a post and saves it to a subfolder based on its rating
func downloadPost(post Post, options inputOptions) {
	resp, err := http.Get(post.FileURL)
	if err != nil {
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing body")
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading post:", post.ID)
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
	if _, err := os.Stat(fmt.Sprint("./" + options.outputDir + subfolder)); os.IsNotExist(err) {
		newpath := filepath.Join(options.outputDir, subfolder)
		err := os.MkdirAll(newpath, os.ModePerm)
		if err != nil {
			return
		}
	}

	filename := strconv.Itoa(post.Score) + "_" + strconv.Itoa(post.ID) + "." + post.FileExt
	filename = filepath.Join(fmt.Sprint(options.outputDir+subfolder), filename)

	if _, err := os.Stat(filename); err == nil {
		return
	}

	err = os.WriteFile(filename, body, 0o644)
	if err != nil {
		fmt.Println("Error writing post:", post.ID)
		return
	}
}

// Loops over all pages and returns a list of all posts
//
// Uses a Progress Bar to show the progress to the user
func fetchPostsFromPage(tags []string, totalPageAmount int, options inputOptions) []Post {
	posts := []Post{}

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)

	// API rate limit is 10 requests per second, can go higher but will likely
	// result in a lot of errors
	rl := ratelimit.New(10)

	pages_bar := progressbar.NewOptions(totalPageAmount,
		progressbar.OptionSetDescription("Fetching posts"),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[cyan]#[reset]",
			SaucerHead:    "[cyan]=[reset]",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
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
				"https://danbooru.donmai.us/posts.json?page=%d&tags=%s&limit=200&only=rating,file_url,id,score,file_ext",
				currentPage, tagString)

			// Credentials to get access to extra features for Danbooru Gold users
			postsUrl += "&login=" + os.Getenv("LOGIN_NAME") + "&api_key=" + os.Getenv("API_KEY")

			response, err := http.Get(postsUrl)
			if err != nil {
				fmt.Println("Error fetching posts")
			}

			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					return
				}
			}(response.Body)

			// Read JSON response into a byte list
			responseData, err := io.ReadAll(response.Body)
			if err != nil {
				fmt.Println("Error reading response")
			}

			// Parse JSON Response into list of posts
			var result []Post
			if err := json.Unmarshal(responseData, &result); err != nil {
				fmt.Println("Error reading response,", response.Status)
			}

			// User can exclude ratings via CLI flags
			for _, post := range result {
				if post.Rating == "s" && !options.sensitive ||
					post.Rating == "q" && !options.questionable ||
					post.Rating == "e" && !options.explicit ||
					post.Rating == "g" && !options.general {
					continue
				}
				posts = append(posts, post)
			}

			pages_bar.Add(1)
		}(i)
	}
	wg.Wait()
	return posts
}

// Get the total amount of pages for the given tags
//
// This is used to determine how many pages worth of posts to fetch
func getTotalPages(tags []string) int {
	tagString := ""
	for _, tag := range tags {
		tagString += url.QueryEscape(tag) + "+"
	}
	url := fmt.Sprintf("https://danbooru.donmai.us/posts?tags=%s&limit=200", tagString)

	// Credentials to get access to extra features for Danbooru Gold users
	url += "&login=" + os.Getenv("LOGIN_NAME") + "&api_key=" + os.Getenv("API_KEY")

	resp, err := http.Get(url)
	if err != nil {
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	// Don't want the program to think there's 1 page worth of posts
	// When there's not a single post on the page
	no_posts := doc.Find("#posts > div > p").Text()
	if no_posts == "No posts found." {
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

// Returns true if the given string is inside the slice
func contains(slice []string, element string) bool {
	for _, a := range slice {
		if a == element {
			return true
		}
	}
	return false
}

func printHelpMessage() {
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

func parseArgs(args []string) inputOptions {
	options := inputOptions{
		outputDir:    "output",
		tags:         []string{},
		sensitive:    true,
		general:      true,
		questionable: true,
		explicit:     true,
	}

	for i := range args {
		switch args[i] {
		case "-o", "--output":
			if len(args) > i+1 {
				options.outputDir = args[i+1]
			}
		case "-t", "--tag":
			if len(args) > i+1 {
				if strings.Contains(args[i+1], "+") {
					options.tags = strings.Split(args[i+1], "+")
				} else {
					// When manually selecting multiple tags on the website
					// they are separated by spaces
					options.tags = strings.Split(args[i+1], " ")
				}
			}
		case "-q", "--questionable":
			options.questionable = false
		case "-e", "--explicit":
			options.explicit = false
		case "-s", "--sensitive":
			options.sensitive = false
		case "-g", "--general":
			options.general = false
		}
	}

	return options
}
