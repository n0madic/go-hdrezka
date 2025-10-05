package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/proxy"
)

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

	downloader := NewHLSDownloader(httpClient())
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

	client := httpClient()
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

func httpClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if args.Resolver != "" {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Second * 10,
				}
				return d.DialContext(ctx, "udp", args.Resolver+":53")
			},
		}
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}

	if args.Proxy != "" {
		proxyURL, err := url.Parse(args.Proxy)
		if err == nil {
			scheme := strings.ToLower(proxyURL.Scheme)

			// Handle SOCKS5 proxies
			if scheme == "socks5" || scheme == "socks5h" {
				var auth *proxy.Auth
				if proxyURL.User != nil {
					auth = &proxy.Auth{
						User: proxyURL.User.Username(),
					}
					if password, ok := proxyURL.User.Password(); ok {
						auth.Password = password
					}
				}

				// Create SOCKS5 dialer
				socksDialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, dialer)
				if err == nil {
					// Use SOCKS5 dialer for all connections
					transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
						return socksDialer.Dial(network, addr)
					}
				}
			} else {
				// Handle HTTP/HTTPS proxies
				transport.Proxy = http.ProxyURL(proxyURL)
			}
		}
	}

	return &http.Client{
		Transport: transport,
	}
}
