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

func downloadFile(url, output string) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = attemptDownload(url, output)
		if err == nil {
			return nil
		}
		// Exponential backoff with jitter
		waitTime := time.Duration(attempt*attempt) * time.Second
		time.Sleep(waitTime + time.Duration(rand.Intn(1000))*time.Millisecond)
		fmt.Printf("Retrying to download file, attempt %d\n", attempt)
	}
	return fmt.Errorf("after 3 attempts, last error: %v", err)
}

func attemptDownload(url, output string) error {
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	currentSize := fileInfo.Size()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	var totalSize int64
	if currentSize > 0 {
		respHead, err := http.Head(url)
		if err != nil {
			return err
		}
		defer respHead.Body.Close()

		if respHead.ContentLength == currentSize {
			return nil
		}

		totalSize = respHead.ContentLength
		if respHead.Header.Get("Accept-Ranges") == "bytes" && respHead.ContentLength > currentSize {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", currentSize))
		} else {
			file.Truncate(0)
			file.Seek(0, 0)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if totalSize == 0 {
		totalSize = resp.ContentLength
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
		bar := progressbar.DefaultBytes(
			totalSize,
			"downloading "+output,
		)
		if currentSize > 0 {
			bar.Add64(currentSize)
		}
		_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
		if err == nil {
			file.Sync()
		}
		return err
	} else {
		return fmt.Errorf("HTTP request returned status: %s", resp.Status)
	}
}
