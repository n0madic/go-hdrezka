package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/grafov/m3u8"
)

// HLSDownloader represents an HLS playlist downloader
type HLSDownloader struct {
	Headers          http.Header   // Custom HTTP headers
	RetryAttempts    int           // Number of retry attempts for failed downloads
	RetryDelay       time.Duration // Delay between retry attempts
	client           *http.Client
	progressCallback HLSProgressCallback // Progress reporting function
}

// HLSProgressInfo contains information about download progress
type HLSProgressInfo struct {
	TotalSegments   int   // Total number of segments
	DownloadedBytes int64 // Downloaded bytes
	CurrentSegment  int   // Current downloading segment
}

// HLSProgressCallback defines the interface for progress callback functions
type HLSProgressCallback func(HLSProgressInfo)

// NewHLSDownloader creates a new downloader instance with default settings
func NewHLSDownloader(client *http.Client) *HLSDownloader {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second, // Default timeout for HTTP requests
		}
	}
	return &HLSDownloader{
		Headers:       make(http.Header),
		RetryAttempts: 3,           // Default to 3 retry attempts
		RetryDelay:    time.Second, // Default to 1 second delay between retries
		client:        client,
	}
}

// SetRetryAttempts sets the number of retry attempts for failed downloads
func (d *HLSDownloader) SetRetryAttempts(attempts int) {
	if attempts < 0 {
		attempts = 0
	}
	d.RetryAttempts = attempts
}

// SetRetryDelay sets the delay between retry attempts
func (d *HLSDownloader) SetRetryDelay(delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	d.RetryDelay = delay
}

// SetHLSProgressCallback sets the function for reporting download progress
func (d *HLSDownloader) SetProgressCallback(callback HLSProgressCallback) {
	d.progressCallback = callback
}

// Download downloads an HLS playlist from the specified URL and saves it to a single TS file
func (d *HLSDownloader) Download(playlistURL, outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Download segments to the file
	err = d.downloadPlaylist(playlistURL, outFile)
	if err != nil {
		return fmt.Errorf("failed to download playlist: %w", err)
	}

	return nil
}

// downloadPlaylist handles the playlist download process
func (d *HLSDownloader) downloadPlaylist(playlistURL string, out io.Writer) error {
	// Fetch the playlist
	resp, err := d.makeRequest(playlistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse the playlist
	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return fmt.Errorf("failed to parse playlist: %w", err)
	}

	// Handle playlist based on its type
	switch listType {
	case m3u8.MASTER:
		// Handle master playlist - select the best quality variant
		masterpl := playlist.(*m3u8.MasterPlaylist)
		if len(masterpl.Variants) == 0 {
			return errors.New("no variants found in master playlist")
		}

		// Select the best quality (highest bandwidth)
		bestVariant := masterpl.Variants[0]
		bestBandwidth := bestVariant.Bandwidth

		for _, variant := range masterpl.Variants {
			if variant.Bandwidth > bestBandwidth {
				bestBandwidth = variant.Bandwidth
				bestVariant = variant
			}
		}

		// Get absolute URL for the selected variant
		variantURL, err := resolveURL(playlistURL, bestVariant.URI)
		if err != nil {
			return err
		}

		// Download the media playlist
		return d.downloadPlaylist(variantURL, out)

	case m3u8.MEDIA:
		// Handle media playlist - download all segments
		mediapl := playlist.(*m3u8.MediaPlaylist)

		baseURL, err := url.Parse(playlistURL)
		if err != nil {
			return fmt.Errorf("failed to parse playlist URL: %w", err)
		}

		return d.downloadSegments(mediapl, baseURL, out)

	default:
		return errors.New("unknown playlist type")
	}
}

// downloadSegments downloads all segments from a media playlist
func (d *HLSDownloader) downloadSegments(playlist *m3u8.MediaPlaylist, baseURL *url.URL, out io.Writer) error {
	// Count non-nil segments
	totalSegments := 0
	for _, segment := range playlist.Segments {
		if segment != nil {
			totalSegments++
		}
	}

	// Initialize progress info
	HLSprogressInfo := HLSProgressInfo{
		TotalSegments: totalSegments,
	}

	// Get file for syncing if output is a file
	outFile, isFile := out.(*os.File)

	// Download segments sequentially
	segmentIndex := 0
	for i, segment := range playlist.Segments {
		if segment == nil {
			continue
		}

		// Resolve segment URL
		segmentURL, err := resolveURL(baseURL.String(), segment.URI)
		if err != nil {
			return fmt.Errorf("failed to resolve segment URL %s: %w", segment.URI, err)
		}

		// Download segment with retries
		var data []byte
		for attempt := 0; attempt <= d.RetryAttempts; attempt++ {
			if attempt > 0 {
				// Wait before retry
				time.Sleep(d.RetryDelay)
			}

			data, err = d.downloadSegment(segmentURL)
			if err == nil {
				break // Successfully downloaded
			}

			// If this was the last attempt, return the error
			if attempt == d.RetryAttempts {
				return fmt.Errorf("failed to download segment %d after %d attempts: %w",
					i, d.RetryAttempts+1, err)
			}
		}

		// Write segment to file
		_, err = out.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write segment %d: %w", segmentIndex, err)
		}

		// Sync file to disk if it's a file
		if isFile {
			if err := outFile.Sync(); err != nil {
				return fmt.Errorf("failed to sync file after segment %d: %w", segmentIndex, err)
			}
		}

		// Update progress info
		HLSprogressInfo.DownloadedBytes += int64(len(data))
		HLSprogressInfo.CurrentSegment = segmentIndex + 1

		// Call progress callback
		if d.progressCallback != nil {
			d.progressCallback(HLSprogressInfo)
		}

		segmentIndex++
	}

	// Final progress update
	if d.progressCallback != nil {
		d.progressCallback(HLSprogressInfo)
	}

	return nil
}

// downloadSegment downloads a single segment
func (d *HLSDownloader) downloadSegment(segmentURL string) ([]byte, error) {
	resp, err := d.makeRequest(segmentURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read segment data
	return io.ReadAll(resp.Body)
}

// makeRequest makes an HTTP request with configured headers and timeout
func (d *HLSDownloader) makeRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add custom headers
	for key, values := range d.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseURL, relativeURL string) (string, error) {
	// If relativeURL is already an absolute URL, just use it
	if bytes.HasPrefix([]byte(relativeURL), []byte("http://")) || bytes.HasPrefix([]byte(relativeURL), []byte("https://")) {
		return relativeURL, nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	relative, err := url.Parse(relativeURL)
	if err != nil {
		return "", err
	}

	resolved := base.ResolveReference(relative)
	return resolved.String(), nil
}
