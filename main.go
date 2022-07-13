package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/ratelimit"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type inputOptions struct {
	tag       string
	outputDir string
	safe      bool
	risky     bool
	explicit  bool
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

	if contains(args, "-h") || contains(args, "--help") || len(args) == 0 {
		printHelpMessage()
		return
	}

	options := parseArgs(args)

	if options.tag == "" {
		fmt.Println("No tags provided")
		return
	}

	totalPages := getTotalPages(options.tag)
	posts := fetchPostsFromPage(options.tag, totalPages)

	newpath := filepath.Join(".", options.outputDir)
	err := os.MkdirAll(newpath, os.ModePerm)

	if err != nil {
		fmt.Println("Error creating directory, exiting")
		return
	}

	dl_bar := progressbar.NewOptions(len(posts),
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading posts")),
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

	for _, post := range posts {
		go func(post Post) {
			defer wg.Done()
			downloadPost(post, options)
			dl_bar.Add(1)
		}(post)
	}
	wg.Wait()

	fmt.Println("\nTime taken:", time.Since(start))

}

func downloadPost(post Post, options inputOptions) {
	url := post.FileURL

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading post:", post.ID)
		return
	}

	subfolder := "/"

	if post.Rating == "s" {
		subfolder = "/safe"
	} else if post.Rating == "q" {
		subfolder = "/risky"
	} else if post.Rating == "e" {
		subfolder = "/explicit"
	} else {
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

	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		fmt.Println("Error writing post:", post.ID)
		return
	}

}

func fetchPostsFromPage(tag string, totalPageAmount int) []Post {
	posts := []Post{}

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)
	rl := ratelimit.New(10)

	pages_bar := progressbar.NewOptions(totalPageAmount,
		progressbar.OptionSetDescription(fmt.Sprintf("Fetching posts")),
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
			url := fmt.Sprintf("https://danbooru.donmai.us/posts.json?page=%d&tags=%s", currentPage, url.QueryEscape(tag))

			response, err := http.Get(url)
			if err != nil {
				fmt.Println("Error fetching posts")
			}

			defer response.Body.Close()

			responseData, err := ioutil.ReadAll(response.Body)

			if err != nil {
				fmt.Println("Error reading response")
			}

			var result []Post
			if err := json.Unmarshal(responseData, &result); err != nil {
				fmt.Println("Error unmarshalling response")
			}

			posts = append(posts, result...)
			pages_bar.Add(1)
		}(i)
	}
	wg.Wait()
	return posts
}

func getTotalPages(tag string) int {
	rl := ratelimit.New(10)
	url := fmt.Sprintf("https://danbooru.donmai.us/posts?tags=%s", url.QueryEscape(tag))

	rl.Take()
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return 0
	}

	totalPages := doc.Find(".paginator-page.desktop-only").Last().Text()

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
	fmt.Println("  -t, --tag        the specific tag you want to download (required)")
	fmt.Println("  -s, --safe       add this flag for safe images")
	fmt.Println("  -r, --risky      add this flag for suggestive images")
	fmt.Println("  -e, --explicit   add this flag for clearly 18+ images")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-go")
}

func parseArgs(args []string) inputOptions {
	options := inputOptions{}

	options.outputDir = "output"
	options.explicit = false
	options.risky = false
	options.safe = false
	options.tag = ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if len(args) > i+1 {
				options.outputDir = args[i+1]
			}
		case "-t", "--tag":
			if len(args) > i+1 {
				options.tag = args[i+1]
			}
		case "-r", "--risky":
			options.risky = true
		case "-e", "--explicit":
			options.explicit = true
		case "-s", "--safe":
			options.safe = true
		}
	}

	return options

}
