package types

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/tiltedtoast/danbooru-go/internal/logger"
	"github.com/tiltedtoast/danbooru-go/internal/options"
	"github.com/valyala/fasthttp"
)

type Post struct {
	ID           int    `json:"id"`
	Score        int    `json:"score"`
	Rating       string `json:"rating"`
	FileExt      string `json:"file_ext"`
	FileURL      string `json:"file_url"`
	LargeFileURL string `json:"large_file_url"`
}

var (
	logger = log.GetLogger()
	opts   = options.GetOptions()
)

// Download a post and saves it to a subfolder based on its rating
func (post *Post) Download(client *fasthttp.Client) {
	url := post.FileURL

	if post.FileExt == "zip" && strings.Contains(post.LargeFileURL, ".webm") {
		url = post.LargeFileURL
		post.FileExt = "webm"
	}

	logger.Debug(fmt.Sprintf("Downloading %s", url))

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
	if _, err := os.Stat(fmt.Sprint("./" + opts.OutputDir + subfolder)); os.IsNotExist(err) {
		newpath := filepath.Join(opts.OutputDir, subfolder)
		if err := os.MkdirAll(newpath, os.ModePerm); err != nil {
			logger.Warn(fmt.Sprintf("Error creating subfolder: %v", err))
			return
		}
	}

	filename := strconv.Itoa(post.Score) + "_" + strconv.Itoa(post.ID) + "." + post.FileExt
	filename = filepath.Join(fmt.Sprint(opts.OutputDir+subfolder), filename)

	if _, err := os.Stat(filename); err == nil {
		logger.Trace(fmt.Sprintf("File already exists: %s", filename))
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		logger.Warn(fmt.Sprintf("Error creating file: %v", err))
		return
	}

	code, body, err := client.Get(nil, url)
	if err != nil || code != fasthttp.StatusOK {
		logger.Warn(fmt.Sprintf("[%d] Error downloading post: %s", code, err))
		return
	}

	defer file.Close()
	w := bufio.NewWriter(file)
	defer w.Flush()
	if _, err := w.Write(body); err != nil {
		logger.Warn("Error writing post:", post.ID)
		return
	}
}
