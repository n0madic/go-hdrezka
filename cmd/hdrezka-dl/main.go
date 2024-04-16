package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	expandrange "github.com/n0madic/expand-range"
	"github.com/n0madic/go-hdrezka"
)

var args struct {
	URL         string `arg:"positional,required" help:"url for download video"`
	Output      string `arg:"positional" help:"output file or path for downloaded video"`
	Info        bool   `arg:"-i" help:"show info about video only"`
	Overwrite   bool   `arg:"-o" help:"overwrite output file if exists"`
	Quality     string `arg:"-q,--quality" default:"1080p" help:"quality for download video"`
	Season      int    `arg:"-s,--season" help:"season for download series"`
	Episodes    string `arg:"-e,--episodes" help:"range of episodes for download (required --season arg)"`
	Translation string `arg:"-t,--translation" placeholder:"NAME" help:"translation for download video"`
}

func main() {
	arg.MustParse(&args)

	if args.Season == 0 && args.Episodes != "" {
		fmt.Println("error: --season arg is required")
		os.Exit(1)
	}

	epRange, err := expandrange.Parse(args.Episodes)
	if args.Episodes != "" && err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	r, err := hdrezka.New(args.URL)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

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
		args.Output = fmt.Sprintf("%s (%s).mp4", title, video.Year)
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
		if season > 0 && episode > 0 {
			output = fmt.Sprintf("s%02de%02d %s", season, episode, output)
		}
		_, err := os.Stat(output)
		if !args.Overwrite && err == nil {
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
		err = downloadFile(format.MP4, output)
		if err != nil {
			fmt.Printf("ERROR %s: %s\n", output, err)
			return
		}
	}

	episodes, err := translation.GetEpisodes()
	if err == nil {
		for _, season := range episodes.ListSeasons() {
			if args.Season > 0 && season != args.Season {
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
