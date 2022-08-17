package hdrezka

import (
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
	Subtitle    any    `json:"subtitle"`
	SubtitleDef any    `json:"subtitle_def"`
	Thumbnails  string `json:"thumbnails"`
	URL         string `json:"url"`
}

// GetStream get stream for video
func (r *HDRezka) GetStream(videoID string, translationID string, season, episode int) (*Stream, error) {
	var form url.Values
	if season > 0 && episode > 0 {
		form = url.Values{
			"id":            {videoID},
			"translator_id": {translationID},
			"season":        {strconv.Itoa(season)},
			"episode":       {strconv.Itoa(episode)},
			"action":        {"get_stream"},
		}
	} else {
		form = url.Values{
			"id":            {videoID},
			"translator_id": {translationID},
			"action":        {"get_movie"},
		}
	}

	var stream Stream
	err := r.getCDN(form, &stream)
	if err != nil {
		return nil, err
	}

	stream.URL, err = decodeURL(stream.URL)
	if err != nil {
		return nil, err
	}
	stream.Formats = parseStreamFormats(stream.URL)

	if stream.Thumbnails != "" {
		thumbURL, err := url.QueryUnescape(r.URL.JoinPath(stream.Thumbnails).String())
		if err == nil {
			stream.Thumbnails = thumbURL
		}
	}
	return &stream, nil
}
