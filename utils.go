package hdrezka

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	reQualityTag = regexp.MustCompile(`\[([^\]]+)\]`)
	reTranslate  = regexp.MustCompile(`initCDN(Series|Movies)Events\(\d+,\s(\d+),.+?(\{.*?\})\);`)
)

func decodeURL(url string) (string, error) {
	// New format: URL is already in plain text (no base64 encoding)
	if strings.HasPrefix(url, "[") || strings.HasPrefix(url, "http") {
		return url, nil
	}

	// Old format: base64 encoded with obfuscation patterns
	url = strings.TrimPrefix(url, "#h")
	for i := 1; i <= 2; i++ {
		url = strings.ReplaceAll(url, "//_//", "")
		url = strings.ReplaceAll(url, "IyMjI14hISMjIUBA", "")
		url = strings.ReplaceAll(url, "QEBAQEAhIyMhXl5e", "")
		url = strings.ReplaceAll(url, "JCQhIUAkJEBeIUAjJCRA", "")
		url = strings.ReplaceAll(url, "JCQjISFAIyFAIyM=", "")
		url = strings.ReplaceAll(url, "Xl5eIUAjIyEhIyM=", "")
	}
	decoded, err := base64.StdEncoding.DecodeString(url)
	if err != nil {
		// Fallback: if base64 decoding fails, return as-is
		return url, nil
	}
	return string(decoded), nil
}

func getCategory(selector string, doc *goquery.Document) map[string]string {
	categories := make(map[string]string)
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		category := s.Find("a")
		if category != nil {
			categories[category.Text()] = category.AttrOr("href", "")
		}
	})
	return categories
}

func (r *HDRezka) getDoc(uri string) (*goquery.Document, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (r *HDRezka) getItems(url string, maxItems int) ([]*CoverItem, error) {
	items := make([]*CoverItem, 0)
	for {
		doc, err := r.getDoc(url)
		if err != nil {
			return nil, err
		}

		doc.Find("div.b-content__inline_items > div.b-content__inline_item").Each(func(i int, s *goquery.Selection) {
			if len(items) == maxItems {
				return
			}
			link := s.Find("div.b-content__inline_item-link > a")
			items = append(items, &CoverItem{
				Cover:       s.Find("div.b-content__inline_item-cover > a > img").AttrOr("src", ""),
				Description: strings.ReplaceAll(s.Find("div.b-content__inline_item-link > div").Text(), " - ...", ""),
				Info:        s.Find("span.info").Text(),
				Title:       link.Text(),
				URL:         link.AttrOr("href", ""),
			})
		})

		url = doc.Find(".b-navigation__next").Parent().AttrOr("href", "")
		if len(items) >= maxItems || url == "" {
			break
		}
	}
	return items, nil
}

func parseFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(str string) int {
	re := regexp.MustCompile(`[^0-9]+`)
	i, err := strconv.Atoi(re.ReplaceAllString(str, ""))
	if err != nil {
		return 0
	}
	return i
}

func parseStreamFormats(str string) map[string]VideoFormat {
	formats := make(map[string]VideoFormat)

	locs := reQualityTag.FindAllStringSubmatchIndex(str, -1)
	for i, loc := range locs {
		quality := str[loc[2]:loc[3]]

		// Extract URL string between this tag and the next (or end of string)
		start := loc[1]
		end := len(str)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		urlStr := strings.TrimRight(strings.TrimSpace(str[start:end]), ",")

		// Split alternatives separated by " or "
		urls := strings.Split(urlStr, " or ")

		var hls, mp4 string

		// Check if any URL has :hls:manifest.m3u8 suffix (new format)
		hasHLSSuffix := false
		for _, u := range urls {
			if strings.HasSuffix(strings.TrimSpace(u), ":hls:manifest.m3u8") {
				hasHLSSuffix = true
				break
			}
		}

		if hasHLSSuffix {
			// New format: classify by suffix
			for _, u := range urls {
				u = strings.TrimSpace(u)
				if u == "" {
					continue
				}
				if strings.HasSuffix(u, ":hls:manifest.m3u8") {
					if hls == "" {
						hls = u
					}
				} else if mp4 == "" {
					mp4 = u
				}
			}
			if hls != "" && mp4 == "" {
				mp4 = strings.TrimSuffix(hls, ":hls:manifest.m3u8")
			}
		} else {
			// Old format: first URL is HLS, second is MP4
			if len(urls) >= 1 {
				hls = strings.TrimSpace(urls[0])
			}
			if len(urls) >= 2 {
				mp4 = strings.TrimSpace(urls[len(urls)-1])
			} else {
				mp4 = hls
			}
		}

		formats[quality] = VideoFormat{
			HLS: hls,
			MP4: mp4,
		}
	}

	return formats
}

func parseSubtitles(subs string) map[string]string {
	subtitles := make(map[string]string)
	for _, entry := range strings.Split(subs, ",") {
		endBracket := strings.Index(entry, "]")
		if endBracket == -1 {
			continue
		}
		key := entry[1:endBracket]
		value := entry[endBracket+1:]
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		subtitles[key] = value
	}
	return subtitles
}
