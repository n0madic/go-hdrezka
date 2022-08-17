# go-hdrezka
[![Go Reference](https://pkg.go.dev/badge/github.com/n0madic/go-hdrezka.svg)](https://pkg.go.dev/github.com/n0madic/go-hdrezka)

Scraper package for HDrezka site.

## Install

```
go get -u github.com/n0madic/go-hdrezka
```

## Usage example

```go
package main

import (
	"fmt"

	"github.com/n0madic/go-hdrezka"
)

func main() {
	r, err := hdrezka.New("https://hdrezka.ag", "https://rezka.ag")
	if err != nil {
		panic(err)
	}
	// Get list of the new popular series
	opts := hdrezka.CoverOption{
		Filter: hdrezka.FilterPopular,
		Genre:  hdrezka.Series,
		Type:   hdrezka.CoverNew,
	}
	items, err := r.GetCovers(opts, 5)
	if err != nil {
		panic(err)
	}
	// Get first video
	video, err := r.GetVideo(items[0].URL)
	if err != nil {
		panic(err)
	}
	// Print information about video
	fmt.Println(video)
	// Get episodes for first translation
	episodes, err := r.GetEpisodes(video.ID, video.Translation[0].ID)
	if err != nil {
		panic(err)
	}
	// Get stream for first season episodes
	for episode := range episodes[1] {
		stream, err := r.GetStream(video.ID, video.Translation[0].ID, 1, episode)
		if err != nil {
			panic(err)
		}
		// Print stream URL for episode
		fmt.Println(stream.Formats["1080p"].MP4)
	}
}
```

Fully functional examples can be found in the `cmd` folder:
* [hdrezka-dl](https://github.com/n0madic/go-hdrezka/tree/master/cmd/hdrezka-dl) - utility that downloads videos from the HDrezka site
* [hdrezka-rlz](https://github.com/n0madic/go-hdrezka/tree/master/cmd/hdrezka-rlz) - utility for receiving and searching for releases (covers) from the site
