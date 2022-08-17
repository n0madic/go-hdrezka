package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/n0madic/go-hdrezka"
	"github.com/schollz/progressbar/v3"
)

var args struct {
	URL        string `arg:"positional,required" help:"url for download video"`
	Output     string `arg:"positional" help:"output file for downloaded video"`
	Info       bool   `arg:"-i" help:"show info about video only"`
	Overwrite  bool   `arg:"-o" help:"overwrite output file if exists"`
	Quality    string `arg:"-q,--quality" default:"1080p" help:"quality for download video"`
	Season     int    `arg:"-s,--season" help:"season for download series"`
	Translator string `arg:"-t,--translator" placeholder:"NAME" help:"translator for download video"`
}

func main() {
	arg.MustParse(&args)

	r, err := hdrezka.New(args.URL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	video, err := r.GetVideo(args.URL)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	fmt.Println(video)
	if args.Info {
		for _, translation := range video.Translation {
			episodes, err := r.GetEpisodes(video.ID, translation.ID)
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

	if args.Output == "" {
		title := video.Title
		if video.TitleOriginal != "" {
			title = video.TitleOriginal
		}
		filename := fmt.Sprintf("%s (%s) %s.mp4", title, video.Year, strings.ToLower(args.Quality))
		args.Output = filename
	}

	var translation *hdrezka.Translator
	for _, tr := range video.Translation {
		if args.Translator != "" && tr.Name == args.Translator {
			translation = tr
			break
		} else if tr.IsDefault {
			translation = tr
		}
	}
	if args.Translator != "" && translation.Name != args.Translator {
		fmt.Printf("Translation %s not found\n", args.Translator)
		os.Exit(3)
	}

	downloadStream := func(videoID string, translationID string, season int, episode int) {
		output := args.Output
		if season > 0 && episode > 0 {
			output = fmt.Sprintf("S%02dE%02d %s", season, episode, output)
		}
		_, err := os.Stat(output)
		if !args.Overwrite && err == nil {
			fmt.Printf("File %s already exists, skipping\n", output)
			return
		}
		stream, err := r.GetStream(videoID, translationID, season, episode)
		if err != nil {
			fmt.Printf("ERROR %s: %s", output, err)
			return
		}
		format, ok := stream.Formats[args.Quality]
		if !ok {
			fmt.Printf("ERROR %s: quality %s not found\n", output, args.Quality)
			return
		}
		err = downloadFile(format.MP4, output)
		if err != nil {
			fmt.Printf("ERROR %s: %s", output, err)
			return
		}
	}

	episodes, err := r.GetEpisodes(video.ID, translation.ID)
	if err == nil {
		for _, season := range episodes.ListSeasons() {
			if args.Season > 0 && season != args.Season {
				continue
			}
			for _, episode := range episodes.ListEpisodes(season) {
				downloadStream(video.ID, translation.ID, season, episode)
			}
		}
	} else {
		downloadStream(video.ID, translation.ID, 0, 0)
	}
}

func downloadFile(url, output string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading "+output,
	)
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)

	return err
}
