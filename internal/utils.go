package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
	pb "github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"
	"go.uber.org/ratelimit"

	log "github.com/tiltedtoast/danbooru-go/internal/logger"
	"github.com/tiltedtoast/danbooru-go/internal/types"
	"github.com/tiltedtoast/danbooru-go/internal/options"
)

var (
	BASE_URL       = "https://danbooru.donmai.us"
	POSTS_PER_PAGE = 200
	LOGIN_NAME     = os.Getenv("DANBOORU_LOGIN")
	API_KEY        = os.Getenv("DANBOORU_API_KEY")
	logger         = log.GetLogger()
	opts           = options.GetOptions()
)

// Loops over all pages and returns a list of all posts
//
// Uses a Progress Bar to show the progress to the user
func FetchPostsFromPage(totalPageAmount int, client *fasthttp.Client) []types.Post {
	var posts []types.Post

	wg := sync.WaitGroup{}
	wg.Add(totalPageAmount)

	// API rate limit is 10 requests per second, can go higher but will likely
	// result in a lot of errors
	rl_per_second := 10

	if IsGoldMember() {
		rl_per_second = 20
	}
	rl := ratelimit.New(rl_per_second)

	pagesBar := pb.NewOptions(totalPageAmount,
		pb.OptionSetDescription("Fetching posts"),
		pb.OptionEnableColorCodes(true),
		pb.OptionFullWidth(),
		pb.OptionShowCount(),
		pb.OptionSetPredictTime(false),
		pb.OptionSetTheme(pb.Theme{
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
			for _, tag := range opts.Tags {
				tagString += url.QueryEscape(tag) + "+"
			}

			postsUrl := fmt.Sprintf(
				"%s/posts.json?page=%d&tags=%s&limit=%d&only=rating,file_url,id,score,file_ext,large_file_url",
				BASE_URL, currentPage, tagString, POSTS_PER_PAGE,
			)

			logger.Trace(postsUrl)

			// Credentials to get access to extra features for Danbooru Gold users
			if LOGIN_NAME != "" && API_KEY != "" {
				postsUrl += "&login=" + LOGIN_NAME + "&api_key=" + API_KEY
			}

			statusCode, body, err := client.Get(nil, postsUrl)
			if err != nil {
				logger.Warn("Error fetching posts,", err)
				return
			}
			logger.Debug("Status code:", statusCode)

			// Parse JSON Response into list of posts
			var result []types.Post
			if err := json.Unmarshal(body, &result); err != nil {
				logger.Error("Error reading response,", statusCode)
				return
			}

			// User can exclude ratings via CLI flags
			for _, post := range result {
				if post.Rating == "s" && !opts.Sensitive ||
					post.Rating == "q" && !opts.Questionable ||
					post.Rating == "e" && !opts.Explicit ||
					post.Rating == "g" && !opts.General {
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
	pageUrl := fmt.Sprintf("%s/posts?tags=%s&limit=%d", BASE_URL, tagString, POSTS_PER_PAGE)
	logger.Debug(pageUrl)

	// Credentials to get access to extra features for Danbooru Gold users
	if LOGIN_NAME != "" && API_KEY != "" {
		pageUrl += "&login=" + LOGIN_NAME + "&api_key=" + API_KEY
	}

	resp, err := http.Get(pageUrl)
	if err != nil {
		logger.Warn("Error fetching page,", err)
		return 0
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Warn("Error reading page,", err)
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
		logger.Warn("Error converting total page amount to int,", err)
		return 0
	}

	return totalAmount
}

func IsGoldMember() bool {
	if LOGIN_NAME == "" || API_KEY == "" {
		return false
	}

	userRes, err := http.Get(fmt.Sprintf("%s/profile.json?login=%s&api_key=%s", BASE_URL, LOGIN_NAME, API_KEY))
	if err != nil {
		return false
	}

	userResData, err := io.ReadAll(userRes.Body)
	if err != nil {
		return false
	}

	var userJson types.User
	if err := json.Unmarshal(userResData, &userJson); err != nil {
		return false
	}

	return userJson.LevelString != "Member"
}

// Chunks a slice into n sub-slices of roughly equal size (last one may be smaller)
//
// These sub-slices do not contain elements from the original slice,
// but rather the start and end indexes within the original slice
func SegmentSlice[T any](slice []T, n int) [][]int {
	length := len(slice)
	if n > length {
		n = length
	}
	subSliceSize := int(math.Ceil(float64(length) / float64(n)))
	partitions := make([][]int, 0)

	for startIdx := 0; startIdx < length; startIdx += subSliceSize {
		endIdx := startIdx + subSliceSize
		if endIdx > length {
			endIdx = length
		}
		partitions = append(partitions, []int{startIdx, endIdx})
	}

	return partitions
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
	fmt.Println("  -v  --verbose       add this flag to enable verbose output (prints everything)")
	fmt.Println("      --debug         add this flag to enable debug information")
	fmt.Println("")
	fmt.Println("For more information, see https://github.com/TiltedToast/danbooru-go")
}
