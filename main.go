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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/ratelimit"
)

type inputOptions struct {
	tags      []string
	outputDir string
	safe      bool
	risky     bool
	explicit  bool
	general   bool
}

type Post struct {
	ID                  int         `json:"id"`
	CreatedAt           string      `json:"created_at"`
	UploaderID          int         `json:"uploader_id"`
	Score               int         `json:"score"`
	Source              string      `json:"source"`
	Md5                 string      `json:"md5"`
	LastCommentBumpedAt interface{} `json:"last_comment_bumped_at"`
	Rating              string      `json:"rating"`
	ImageWidth          int         `json:"image_width"`
	ImageHeight         int         `json:"image_height"`
	TagString           string      `json:"tag_string"`
	FavCount            int         `json:"fav_count"`
	FileExt             string      `json:"file_ext"`
	LastNotedAt         string      `json:"last_noted_at"`
	ParentID            interface{} `json:"parent_id"`
	HasChildren         bool        `json:"has_children"`
	ApproverID          int         `json:"approver_id"`
	TagCountGeneral     int         `json:"tag_count_general"`
	TagCountArtist      int         `json:"tag_count_artist"`
	TagCountCharacter   int         `json:"tag_count_character"`
	TagCountCopyright   int         `json:"tag_count_copyright"`
	FileSize            int         `json:"file_size"`
	UpScore             int         `json:"up_score"`
	DownScore           int         `json:"down_score"`
	IsPending           bool        `json:"is_pending"`
	IsFlagged           bool        `json:"is_flagged"`
	IsDeleted           bool        `json:"is_deleted"`
	TagCount            int         `json:"tag_count"`
	UpdatedAt           string      `json:"updated_at"`
	IsBanned            bool        `json:"is_banned"`
	PixivID             int         `json:"pixiv_id"`
	LastCommentedAt     interface{} `json:"last_commented_at"`
	HasActiveChildren   bool        `json:"has_active_children"`
	BitFlags            int         `json:"bit_flags"`
	TagCountMeta        int         `json:"tag_count_meta"`
	HasLarge            bool        `json:"has_large"`
	HasVisibleChildren  bool        `json:"has_visible_children"`
	TagStringGeneral    string      `json:"tag_string_general"`
	TagStringCharacter  string      `json:"tag_string_character"`
	TagStringCopyright  string      `json:"tag_string_copyright"`
	TagStringArtist     string      `json:"tag_string_artist"`
	TagStringMeta       string      `json:"tag_string_meta"`
	FileURL             string      `json:"file_url"`
	LargeFileURL        string      `json:"large_file_url"`
	PreviewFileURL      string      `json:"preview_file_url"`
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
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[cyan]#[reset]",
			SaucerHead:    "[cyan]=[reset]",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
		}))

	wg := sync.WaitGroup{}
	wg.Add(len(posts))
	start := time.Now()

	maxGoroutines := runtime.NumCPU()
	guard := make(chan int, maxGoroutines)

	for _, post := range posts {
		guard <- 1
		go func(post Post) {
			defer wg.Done()
			downloadPost(post, options)
			dl_bar.Add(1)
			<-guard
		}(post)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println("\nTime taken:", elapsed.Round(time.Second))
}

func downloadPost(post Post, options inputOptions) {
	url := post.FileURL

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading post:", post.ID)
		return
	}

	var subfolder string

	switch post.Rating {
	case "s":
		subfolder = "/safe"
	case "q":
		subfolder = "/risky"
	case "e":
		subfolder = "/explicit"
	case "g":
		subfolder = "/general"
	default:
		subfolder = "/unknown"
	}

	if _, err := os.Stat(fmt.Sprint("./" + options.outputDir + subfolder)); os.IsNotExist(err) {
		newpath := filepath.Join(options.outputDir, subfolder)
		os.MkdirAll(newpath, os.ModePerm)
	}

	filename := strconv.Itoa(post.ID) + "." + post.FileExt
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

func fetchPostsFromPage(tags []string, totalPageAmount int, options inputOptions) []Post {
	posts := []Post{}

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)
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

	for i := 1; i <= totalPageAmount; i++ {
		go func(currentPage int) {
			defer wg.Done()
			rl.Take()
			tagString := ""
			for _, tag := range tags {
				tagString += url.QueryEscape(tag) + "+"
			}

			url := fmt.Sprintf("https://danbooru.donmai.us/posts.json?page=%d&tags=%s", currentPage, tagString)

			url += "&login=" + os.Getenv("LOGIN_NAME") + "&api_key=" + os.Getenv("API_KEY")

			response, err := http.Get(url)
			if err != nil {
				fmt.Println("Error fetching posts")
			}

			defer response.Body.Close()

			responseData, err := io.ReadAll(response.Body)
			if err != nil {
				fmt.Println("Error reading response")
			}

			var result []Post
			if err := json.Unmarshal(responseData, &result); err != nil {
				fmt.Println("Error unmarshalling response")
			}

			for _, post := range result {
				if post.Rating == "s" && !options.safe ||
					post.Rating == "q" && !options.risky ||
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

func getTotalPages(tags []string) int {
	tagString := ""
	for _, tag := range tags {
		tagString += url.QueryEscape(tag) + "+"
	}
	url := fmt.Sprintf("https://danbooru.donmai.us/posts?tags=%s", tagString)

	url += "&login=" + os.Getenv("LOGIN_NAME") + "&api_key=" + os.Getenv("API_KEY")

	resp, err := http.Get(url)
	if err != nil {
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0
	}

	no_posts := doc.Find("#posts > div > p").Text()

	if no_posts == "No posts found." {
		return 0
	}

	totalPages := doc.Find(".paginator-page.desktop-only").Last().Text()

	if totalPages == "" {
		return 1
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
	fmt.Println("  danbooru-go [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help       print this help message and exit")
	fmt.Println("  -o, --output     output directory, defaults to 'output' subdirectory")
	fmt.Println("  -t, --tags       the specific tags you want to search for, split by \"-\" or spaces (required)")
	fmt.Println("  -s, --safe       add this flag for filter out safe images")
	fmt.Println("  -g, --general    add this flag for filter out general images (everything but the other 3 categories)")
	fmt.Println("  -r, --risky      add this flag for filter out suggestive images")
	fmt.Println("  -e, --explicit   add this flag for filter out clearly 18+ images")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-go")
}

func parseArgs(args []string) inputOptions {
	options := inputOptions{
		outputDir: "output",
		tags:      []string{},
		safe:      true,
		general:   true,
		risky:     true,
		explicit:  true,
	}

	for i := range args {
		switch args[i] {
		case "-o", "--output":
			if len(args) > i+1 {
				options.outputDir = args[i+1]
			}
		case "-t", "--tag":
			if len(args) > i+1 {
				if strings.Contains(args[i+1], "-") {
					options.tags = strings.Split(args[i+1], "-")
				} else {
					options.tags = strings.Split(args[i+1], " ")
				}
			}
		case "-r", "--risky":
			options.risky = false
		case "-e", "--explicit":
			options.explicit = false
		case "-s", "--safe":
			options.safe = false
		case "-g", "--general":
			options.general = false
		}
	}

	return options
}
