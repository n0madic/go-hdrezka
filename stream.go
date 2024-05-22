package hdrezka

import (
	"fmt"
	"net/url"
	"strconv"
)

// VideoFormat of stream
type VideoFormat struct {
	HLS string `json:"hls"`
	MP4 string `json:"mp4"`
}

// Stream is a struct for stream info
type Stream struct {
	Formats     map[string]VideoFormat
	Subtitles   map[string]string
	Subtitle    any    `json:"subtitle"`
	SubtitleDef any    `json:"subtitle_def"`
	Thumbnails  string `json:"thumbnails"`
	URL         string `json:"url"`
}

// GetStream get stream for video.
// No parameters GetStream() for films, choose GetStream(season, episodes) for series.
func (t *Translation) GetStream(season_episode ...int) (*Stream, error) {
	var season, episode int
	if len(season_episode) > 0 {
		if len(season_episode) == 2 {
			season, episode = season_episode[0], season_episode[1]
		} else {
			return nil, fmt.Errorf("only two parameters are required for season and episode")
		}
	}
	var form url.Values
	if season > 0 && episode > 0 {
		form = url.Values{
			"id":            {t.videoID},
			"translator_id": {t.ID},
			"season":        {strconv.Itoa(season)},
			"episode":       {strconv.Itoa(episode)},
			"action":        {"get_stream"},
		}
	} else {
		form = url.Values{
			"id":            {t.videoID},
			"translator_id": {t.ID},
			"action":        {"get_movie"},
		}
	}

	var stream Stream
	err := t.r.getCDN(form, &stream)
	if err != nil {
		return nil, err
	}

	if subtitleStr, ok := stream.Subtitle.(string); ok && subtitleStr != "" {
		stream.Subtitles = parseSubtitles(subtitleStr)
	}

	stream.URL, err = decodeURL(stream.URL)
	if err != nil {
		return nil, err
	}
	stream.Formats = parseStreamFormats(stream.URL)

	if stream.Thumbnails != "" {
		thumbURL, err := url.QueryUnescape(t.r.URL.JoinPath(stream.Thumbnails).String())
		if err == nil {
			stream.Thumbnails = thumbURL
		}
	}
	return &stream, nil
}
