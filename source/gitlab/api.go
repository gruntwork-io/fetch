package gitlab

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
)

var nextLinkRegex = regexp.MustCompile(`<(.+?)>;\s*rel="next"`)

// callGitLabApi performs HTTP request to GitLab API
func callGitLabApi(apiUrl, path, token string, customHeaders map[string]string) (*http.Response, error) {
	reqUrl := fmt.Sprintf("https://%s/api/v4/%s", apiUrl, path)
	return callGitLabApiRaw(reqUrl, "GET", token, customHeaders)
}

// callGitLabApiRaw performs raw HTTP request with GitLab auth
func callGitLabApiRaw(reqUrl, method, token string, customHeaders map[string]string) (*http.Response, error) {
	httpClient := &http.Client{}

	request, err := http.NewRequest(method, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	// GitLab uses PRIVATE-TOKEN header (different from GitHub)
	if token != "" {
		request.Header.Set("PRIVATE-TOKEN", token)
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
		resp.Body.Close()
		if goErr != nil {
			return nil, goErr
		}
		respBody := buf.String()
		return nil, fmt.Errorf("HTTP %d while fetching %s: %s", resp.StatusCode, reqUrl, respBody)
	}

	return resp, nil
}

// getNextUrl extracts next page URL from Link header (same format as GitHub)
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

// encodeProjectPath URL-encodes the project path for GitLab API
// GitLab requires owner/name to be URL-encoded (/ becomes %2F)
func encodeProjectPath(owner, name string) string {
	projectPath := owner + "/" + name
	return url.PathEscape(projectPath)
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

// writeResponseToTempFile writes HTTP response body to a temp file and returns the file path
func writeResponseToTempFile(resp *http.Response) (string, error) {
	tmpFile, err := os.CreateTemp("", "source-archive-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}

	return tmpFile.Name(), nil
}
