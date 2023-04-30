package types

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// Download a post and saves it to a subfolder based on its rating
func (post *Post) Download(options Args, client *fasthttp.Client) {
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
