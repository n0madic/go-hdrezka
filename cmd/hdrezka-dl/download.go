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

	client := &http.Client{}
	resp, err := client.Head(url)
	if err != nil {
		return fmt.Errorf("error making HEAD request: %w", err)
	}
	resp.Body.Close()

	totalSize := resp.ContentLength
	if currentSize > 0 {
		if totalSize == currentSize {
			return nil // File already completely downloaded
		}
		if resp.Header.Get("Accept-Ranges") == "bytes" && totalSize > currentSize {
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

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error making GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP request returned status: %s", resp.Status)
	}

	bar := progressbar.DefaultBytes(
		totalSize,
		"downloading "+output,
	)
	if currentSize > 0 {
		bar.Add64(currentSize)
	}

	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("error copying data: %w", err)
	}

	return file.Sync()
}
