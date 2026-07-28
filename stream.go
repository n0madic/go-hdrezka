package hdrezka

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// VideoFormat of stream
type VideoFormat struct {
	HLS string `json:"hls"`
	MP4 string `json:"mp4"`
	// IsPremium marks a quality the current account cannot play: HDrezka
	// labels it with the premium badge and the CDN serves a short teaser
	// file instead of the movie. A premium account gets the same quality
	// without the badge, so the flag stays false there.
	IsPremium bool `json:"is_premium"`
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
	if season > 0 {
		form = url.Values{
			"id":            {t.videoID},
			"translator_id": {t.ID},
			"season":        {strconv.Itoa(season)},
			"episode":       {strconv.Itoa(episode)},
			"action":        {"get_stream"},
		}
	} else {
		// For movies the app (FilmModel.getStreamsByTranslationId) also passes
		// the translation flags to get_movie so the CDN returns the matching stream.
		form = url.Values{
			"id":            {t.videoID},
			"translator_id": {t.ID},
			"is_camrip":     {boolTo10(t.IsCamRip)},
			"is_ads":        {boolTo10(t.IsAds)},
			"is_director":   {boolTo10(t.IsDirector)},
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

// contentLength returns the size in bytes of the resource at url using an HTTP
// HEAD request, falling back to a ranged GET when the CDN ignores HEAD (many
// HDrezka CDN servers return no Content-Length for HEAD requests). A nil
// client falls back to http.DefaultClient.
func contentLength(client *http.Client, url string) (int64, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if url == "" {
		return 0, fmt.Errorf("empty url")
	}

	if headReq, err := http.NewRequest(http.MethodHead, url, nil); err == nil {
		headReq.Header.Set("User-Agent", defaultUserAgent)
		if headResp, err := client.Do(headReq); err == nil {
			headResp.Body.Close()
			if headResp.StatusCode == http.StatusOK && headResp.ContentLength > 0 {
				return headResp.ContentLength, nil
			}
		}
	}

	// Fallback: ranged GET. Servers that honour Range return 206 with a
	// Content-Range header carrying the total size; otherwise use ContentLength.
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", "bytes=0-0")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("unexpected status %s", resp.Status)
	}
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		// Format: "bytes 0-0/123456"
		if i := strings.LastIndex(cr, "/"); i >= 0 && i+1 < len(cr) {
			if total, err := strconv.ParseInt(cr[i+1:], 10, 64); err == nil && total > 0 {
				return total, nil
			}
		}
	}
	if resp.ContentLength > 0 {
		return resp.ContentLength, nil
	}
	return 0, fmt.Errorf("could not determine content length for %s", url)
}

// Size returns the size in bytes of the format's media file. It prefers the
// MP4 URL (direct file) and falls back to the HLS URL when MP4 is absent. A nil
// client uses http.DefaultClient.
func (f VideoFormat) Size(client *http.Client) (int64, error) {
	url := f.MP4
	if url == "" {
		url = f.HLS
	}
	return contentLength(client, url)
}

// Size returns the size in bytes of the given quality (e.g. "1080p"). When
// quality is empty, the highest available quality is used. A nil client uses
// http.DefaultClient.
func (s *Stream) Size(client *http.Client, quality string) (int64, error) {
	if len(s.Formats) == 0 {
		return 0, fmt.Errorf("stream has no formats")
	}
	if quality == "" {
		quality = s.bestQuality()
	}
	format, ok := s.Formats[quality]
	if !ok {
		return 0, fmt.Errorf("quality %q not available", quality)
	}
	return format.Size(client)
}

// qualityOrder lists known HDrezka quality labels from highest to lowest so the
// best available format can be selected deterministically.
var qualityOrder = []string{"2160p", "1080p Ultra", "1080p", "720p", "480p", "360p"}

// bestQuality returns the highest-ranked quality label present in the stream.
// Formats flagged as premium are skipped while any regular format exists — the
// CDN serves a teaser file for them unless the account has a subscription (see
// VideoFormat.IsPremium).
func (s *Stream) bestQuality() string {
	if q := bestQualityOf(s.Formats, false); q != "" {
		return q
	}
	return bestQualityOf(s.Formats, true)
}

// bestQualityOf picks the highest-ranked label among the formats. Known labels
// (see qualityOrder) win first; for any unknown labels it falls back to the
// largest parsed vertical resolution, breaking ties by label so the result
// never depends on map iteration order.
func bestQualityOf(formats map[string]VideoFormat, withPremium bool) string {
	eligible := func(quality string) bool {
		format, ok := formats[quality]
		return ok && (withPremium || !format.IsPremium)
	}
	for _, q := range qualityOrder {
		if eligible(q) {
			return q
		}
	}
	best, bestRes := "", -1
	for q := range formats {
		if !eligible(q) {
			continue
		}
		res := parseResolution(q)
		if res > bestRes || (res == bestRes && q > best) {
			best, bestRes = q, res
		}
	}
	return best
}

// Qualities returns the stream's quality labels ordered from best to worst:
// by vertical resolution, premium variants of the same resolution first
// ("1080p Ultra" before "1080p"), labels without a resolution last.
func (s *Stream) Qualities() []string {
	qualities := make([]string, 0, len(s.Formats))
	for q := range s.Formats {
		qualities = append(qualities, q)
	}
	slices.SortFunc(qualities, func(a, b string) int {
		if c := cmp.Compare(parseResolution(b), parseResolution(a)); c != 0 {
			return c
		}
		return cmp.Compare(b, a)
	})
	return qualities
}

// parseResolution extracts the leading vertical resolution from a quality label
// (e.g. "1080p Ultra" -> 1080). Returns -1 when the label has no leading digits.
func parseResolution(quality string) int {
	i := 0
	for i < len(quality) && quality[i] >= '0' && quality[i] <= '9' {
		i++
	}
	if i == 0 {
		return -1
	}
	n, err := strconv.Atoi(quality[:i])
	if err != nil {
		return -1
	}
	return n
}
