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
	reStreams   = regexp.MustCompile(`\[(.*?)\](.*?) [\w]+ (.*?)(,|$)`)
	reTranslate = regexp.MustCompile(`initCDN(Series|Movies)Events\(\d+,\s(\d+),.+?(\{.*?\})\);`)
)

func decodeURL(url string) (string, error) {
	url = strings.TrimLeft(url, "#h")
	for i := 1; i <= 2; i++ {
		url = strings.ReplaceAll(url, "//_//", "")
		url = strings.ReplaceAll(url, "IyMjI14hISMjIUBA", "")
		url = strings.ReplaceAll(url, "QEBAQEAhIyMhXl5e", "")
		url = strings.ReplaceAll(url, "JCQhIUAkJEBeIUAjJCRA", "")
		url = strings.ReplaceAll(url, "JCQjISFAIyFAIyM=", "")
		url = strings.ReplaceAll(url, "Xl5eIUAjIyEhIyM=", "")
	}
	decoded, err := base64.StdEncoding.DecodeString(url)
	return string(decoded), err
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

func getDoc(uri string) (*goquery.Document, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func getItems(url string, maxItems int) ([]*CoverItem, error) {
	items := make([]*CoverItem, 0)
	for {
		doc, err := getDoc(url)
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
	streams := reStreams.FindAllStringSubmatch(str, -1)
	formats := make(map[string]VideoFormat)
	for _, s := range streams {
		formats[string(s[1])] = VideoFormat{
			HLS: string(s[2]),
			MP4: string(s[3]),
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
