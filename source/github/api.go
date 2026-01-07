package github

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
)

var nextLinkRegex = regexp.MustCompile(`<(.+?)>;\s*rel="next"`)

// callGitHubApi performs HTTP request to GitHub API
func callGitHubApi(apiUrl, path, token string, customHeaders map[string]string) (*http.Response, error) {
	url := fmt.Sprintf("https://%s/%s", apiUrl, path)
	return callGitHubApiRaw(url, "GET", token, customHeaders)
}

// callGitHubApiRaw performs raw HTTP request
func callGitHubApiRaw(url, method, token string, customHeaders map[string]string) (*http.Response, error) {
	httpClient := &http.Client{}

	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	for headerName, headerValue := range customHeaders {
		request.Header.Set(headerName, headerValue)
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, goErr := buf.ReadFrom(resp.Body)
		if goErr != nil {
			return nil, goErr
		}
		respBody := buf.String()
		return nil, fmt.Errorf("HTTP %d while fetching %s: %s", resp.StatusCode, url, respBody)
	}

	return resp, nil
}

// getNextUrl extracts next page URL from Link header
func getNextUrl(links string) string {
	if len(links) == 0 {
		return ""
	}

	for _, link := range strings.Split(links, ",") {
		urlMatches := nextLinkRegex.FindStringSubmatch(link)
		if len(urlMatches) == 2 {
			return strings.TrimSpace(urlMatches[1])
		}
	}

	return ""
}

// writeCounter tracks download progress
type writeCounter struct {
	written uint64
	suffix  string
}

func newWriteCounter(total int64) *writeCounter {
	if total > 0 {
		return &writeCounter{
			suffix: fmt.Sprintf(" / %s", humanize.Bytes(uint64(total))),
		}
	}
	return &writeCounter{}
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.written += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc writeCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %s%s", humanize.Bytes(wc.written), wc.suffix)
}

// writeResponseToDisk writes HTTP response body to file
func writeResponseToDisk(resp *http.Response, destPath string, withProgress bool) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	defer out.Close()
	defer resp.Body.Close()

	var reader io.Reader
	if withProgress {
		reader = io.TeeReader(resp.Body, newWriteCounter(resp.ContentLength))
	} else {
		reader = resp.Body
	}
	_, err = io.Copy(out, reader)
	return err
}
