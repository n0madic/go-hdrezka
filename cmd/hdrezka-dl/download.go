package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// siteClient is the shared HTTP client used for both site API requests and
// stream/subtitle downloads. It points at hdrezka.HDRezka.Client so the same
// cookie jar carries the authenticated session through every request.
var siteClient *http.Client

func downloadHLSPlaylist(playlistURL, output string) error {
	fileInfo, err := os.Stat(output)
	if err == nil && fileInfo.Size() > 0 {
		if !args.Overwrite {
			return nil // File already exists
		}
	}

	bar := progressbar.NewOptions(
		-1, // Unknown size initially
		progressbar.OptionSetDescription("downloading HLS "+output),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	downloader := NewHLSDownloader(siteClient)
	downloader.SetProgressCallback(func(info HLSProgressInfo) {
		bar.Set(info.CurrentSegment)
		if info.TotalSegments > 0 {
			bar.ChangeMax(info.TotalSegments)
		}
	})

	err = downloader.Download(playlistURL, output)
	if err != nil {
		return fmt.Errorf("error downloading HLS: %w", err)
	}

	return nil
}

func downloadFile(url, output string, maxAttempt int) error {
	for attempt := 1; attempt <= maxAttempt; attempt++ {
		err := attemptDownload(url, output)
		if err == nil {
			return nil
		}
		if attempt == maxAttempt {
			return fmt.Errorf("after %d attempts, last error: %v", maxAttempt, err)
		}
		waitTime := time.Duration(attempt*attempt)*time.Second + time.Duration(rand.Intn(1000))*time.Millisecond
		fmt.Printf("Error downloading file: %v\nRetrying in %v, attempt %d\n", err, waitTime, attempt+1)
		time.Sleep(waitTime)
	}
	return nil // This line will never be reached, but it's needed for compilation
}

func attemptDownload(url, output string) error {
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}
	currentSize := fileInfo.Size()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	headResp, err := siteClient.Head(url)
	if err != nil {
		return fmt.Errorf("error making HEAD request: %w", err)
	}
	headResp.Body.Close()

	totalSize := headResp.ContentLength
	if currentSize > 0 {
		if totalSize > 0 && totalSize == currentSize {
			return nil // File already completely downloaded
		}
		if headResp.Header.Get("Accept-Ranges") == "bytes" && totalSize > currentSize {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", currentSize))
		} else {
			if err := file.Truncate(0); err != nil {
				return fmt.Errorf("error truncating file: %w", err)
			}
			if _, err := file.Seek(0, 0); err != nil {
				return fmt.Errorf("error seeking file: %w", err)
			}
			currentSize = 0
		}
	}

	resp, err := siteClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP request returned status: %s", resp.Status)
	}

	// Fallback to GET Content-Length if HEAD returned 0 (CDN servers often ignore HEAD)
	if totalSize <= 0 {
		totalSize = resp.ContentLength
	}
	if totalSize <= 0 {
		totalSize = -1 // Unknown size — progressbar spinner mode
	}

	bar := progressbar.DefaultBytes(totalSize, "downloading "+output)
	if currentSize > 0 {
		bar.Add64(currentSize)
	}

	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("error copying data: %w", err)
	}

	return file.Sync()
}
