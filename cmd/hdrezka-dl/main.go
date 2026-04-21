package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
	expandrange "github.com/n0madic/expand-range"
	"github.com/n0madic/go-hdrezka"
)

var args struct {
	URL         string `arg:"positional,required" help:"url for download video"`
	Output      string `arg:"positional" help:"output file or path for downloaded video"`
	BaseURL     string `arg:"-b,--base-url" placeholder:"URL" help:"base URL of hdrezka site (e.g., https://hdrezka.ag)"`
	Info        bool   `arg:"-i" help:"show info about video only"`
	MaxAttempt  int    `arg:"-m,--max-attempt" placeholder:"INT" default:"3" help:"max attempts for download file"`
	Overwrite   bool   `arg:"-o,--overwrite" help:"overwrite output file if exists"`
	Quality     string `arg:"-q,--quality" default:"1080p" help:"quality for download video"`
	Season      string `arg:"-s,--season" placeholder:"RANGE" help:"season or range of seasons to download (e.g. 1, 2-3, 1,3,5)"`
	Episodes    string `arg:"-e,--episodes" placeholder:"RANGE" help:"range of episodes to download, requires single --season (e.g. 1, 3-5, 1,3,7-9)"`
	Translation string `arg:"-t,--translation" placeholder:"NAME" help:"translation for download video"`
	Subtitle    string `arg:"-c,--subtitle" placeholder:"LANG" help:"get subtitle for downloaded video"`
	Resolver    string `arg:"-r,--resolver" placeholder:"IP" help:"DNS resolver for download video"`
	Proxy       string `arg:"-p,--proxy" placeholder:"URL" help:"proxy for download video"`
	UseHLS      bool   `arg:"-l,--hls" help:"use HLS instead of MP4 for download video"`
}

func sanitizeFilename(filename string) string {
	if runtime.GOOS == "windows" {
		// Replace invalid characters for Windows filesystem with spaces
		invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
		result := filename
		for _, char := range invalidChars {
			result = strings.ReplaceAll(result, char, " ")
		}

		// Clean up consecutive spaces
		for strings.Contains(result, "  ") {
			result = strings.ReplaceAll(result, "  ", " ")
		}

		// Trim spaces from beginning and end
		result = strings.TrimSpace(result)
		return result
	}
	return filename
}

func main() {
	arg.MustParse(&args)

	if args.Season == "" && args.Episodes != "" {
		fmt.Println("error: --season arg is required")
		os.Exit(1)
	}
	if args.Season != "" && args.Episodes != "" {
		if _, err := strconv.Atoi(args.Season); err != nil {
			fmt.Println("error: --episodes cannot be combined with a season range")
			os.Exit(1)
		}
	}

	var seasonRange expandrange.Range
	if args.Season != "" {
		var parseErr error
		seasonRange, parseErr = expandrange.Parse(args.Season)
		if parseErr != nil {
			fmt.Printf("error: invalid season range: %v\n", parseErr)
			os.Exit(1)
		}
	}

	epRange, err := expandrange.Parse(args.Episodes)
	if args.Episodes != "" && err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var r *hdrezka.HDRezka
	if args.BaseURL != "" {
		// Use provided base URL instead of extracting from video URL
		r, err = hdrezka.New(args.BaseURL)
	} else {
		// Extract base URL from video URL (original behavior)
		r, err = hdrezka.New(args.URL)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	// GetVideo will automatically normalize the URL to use r.URL
	video, err := r.GetVideo(args.URL)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	fmt.Println(video)
	if args.Info {
		for _, translation := range video.Translation {
			episodes, err := translation.GetEpisodes()
			if err == nil {
				fmt.Println(translation.Name)
				for _, season := range episodes.ListSeasons() {
					fmt.Printf("Season %d: %v\n", season, episodes.ListEpisodes(season))
				}
			}
		}
		return
	}

	fmt.Println()

	if pathInfo, err := os.Stat(args.Output); err == nil && pathInfo.IsDir() {
		os.Chdir(args.Output)
		args.Output = ""
	}

	if args.Output == "" {
		title := video.Title
		if video.TitleOriginal != "" {
			title = video.TitleOriginal
		}
		title = strings.ReplaceAll(title, "/", "-")
		ext := ".mp4"
		if args.UseHLS {
			ext = ".ts"
		}
		args.Output = sanitizeFilename(fmt.Sprintf("%s (%s)%s", title, video.Year, ext))
	}

	var translation *hdrezka.Translation
	for _, tr := range video.Translation {
		if args.Translation != "" && tr.Name == args.Translation {
			translation = tr
			break
		} else if tr.IsDefault {
			translation = tr
		}
	}
	if args.Translation != "" && translation.Name != args.Translation {
		fmt.Printf("Translation %s not found\n", args.Translation)
		os.Exit(4)
	}

	downloadStream := func(season int, episode int) {
		output := args.Output
		if season > 0 {
			output = sanitizeFilename(fmt.Sprintf("s%02de%02d %s", season, episode, output))
		}
		fileInfo, err := os.Stat(output)
		if !args.Overwrite && err == nil && fileInfo.Size() > 0 {
			fmt.Printf("File %s already exists, skipping\n", output)
			return
		}
		stream, err := translation.GetStream(season, episode)
		if err != nil {
			fmt.Printf("ERROR %s: %s\n", output, err)
			return
		}
		format, ok := stream.Formats[args.Quality]
		if !ok {
			fmt.Printf("ERROR %s: quality %s not found\n", output, args.Quality)
			return
		}

		// Download using HLS or MP4 based on user choice
		if args.UseHLS {
			// Use HLS stream
			if format.HLS == "" {
				fmt.Printf("ERROR %s: HLS stream not available for quality %s\n", output, args.Quality)
				return
			}
			err = downloadHLSPlaylist(format.HLS, output)
		} else {
			// Use MP4 stream
			err = downloadFile(format.MP4, output, args.MaxAttempt)
		}

		if err != nil {
			fmt.Printf("ERROR %s: %s\n", output, err)
			return
		}

		// Download subtitles if requested
		if args.Subtitle != "" {
			subtitle, ok := stream.Subtitles[args.Subtitle]
			if !ok {
				fmt.Printf("ERROR %s: subtitle %s not found\n", output, args.Subtitle)
				return
			}
			outputSub := output[:strings.LastIndex(output, ".")] + ".vtt"
			err = downloadFile(subtitle, outputSub, args.MaxAttempt)
			if err != nil {
				fmt.Printf("ERROR %s: %s\n", outputSub, err)
				return
			}
		}
	}

	episodes, err := translation.GetEpisodes()
	if err == nil {
		for _, season := range episodes.ListSeasons() {
			if len(seasonRange) > 0 && !seasonRange.InRange(uint64(season)) {
				continue
			}
			for _, episode := range episodes.ListEpisodes(season) {
				if args.Episodes != "" && !epRange.InRange(uint64(episode)) {
					continue
				}
				downloadStream(season, episode)
			}
		}
	} else {
		downloadStream(0, 0)
	}
}
