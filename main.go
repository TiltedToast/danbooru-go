package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
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

	opts := parseArgs(args)
	fmt.Println(opts)

	if len(opts.tags) == 0 {
		fmt.Println("No tags provided")
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(opts.tags))

	posts := []Post{}
	totalPages := []int{}

	for _, tag := range opts.tags {
		go func(tag string) {
			defer wg.Done()
			tagPosts := fetchPosts(tag, totalPages)
			posts = append(posts, tagPosts...)
		}(tag)
	}
	wg.Wait()

	wg2 := sync.WaitGroup{}
	wg2.Add(len(opts.tags))

	for _, tag := range opts.tags {
		go func(currentTag string) {
			defer wg2.Done()
			posts = append(posts, fetchPosts(currentTag, totalPages)...)
		}(tag)
	}
	wg2.Wait()

	newpath := filepath.Join(".", opts.outputDir)
	err := os.MkdirAll(newpath, os.ModePerm)

	if err != nil {
		fmt.Println("Error creating directory, exiting")
		return
	}

	fmt.Println(len(posts))

}

func fetchPosts(tag string, totalPages []int) []Post {
	posts := []Post{}

	wg := sync.WaitGroup{}
	wg.Add(len(totalPages))
	for _, currentPage := range totalPages {
		go func(currPage int) {
			defer wg.Done()
			posts = append(posts, fetchPostsFromPage(tag, currPage)...)
		}(currentPage)
	}
	wg.Wait()
	return posts
}

func fetchPostsFromPage(tag string, totalPageAmount int) []Post {
	posts := []Post{}

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)

	for i := 1; i <= totalPageAmount; i++ {
		go func(currentPage int) {
			defer wg.Done()
			url := fmt.Sprintf("https://danbooru.donmai.us/posts.json?page=%d&tags=%s", currentPage, url.QueryEscape(tag))

			response, err := http.Get(url)
			if err != nil {
				return
			}

			defer response.Body.Close()

			responseData, err := ioutil.ReadAll(response.Body)

			if err != nil {
				return
			}

			var result []Post
			if err := json.Unmarshal(responseData, &result); err != nil {
				return
			}

			posts = append(posts, result...)
		}(i)
	}
	wg.Wait()
	return posts
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
