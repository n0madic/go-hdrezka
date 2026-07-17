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

	// reInitCDNID captures the post id from the initCDN*Events(<id>, ...) call
	// present on every video page. Used as a DOM-independent fallback for the
	// video ID when the markup around the favorites holder changes.
	reInitCDNID = regexp.MustCompile(`initCDN(?:Series|Movies)Events\((\d+),`)

	// knownSalts are the obfuscation patterns HDrezka inserts after each
	// "//_//" marker in encoded stream URLs. They vary in length, so the
	// decoder strips them by exact match rather than a fixed offset.
	knownSalts = []string{
		"IyMjI14hISMjIUBA", "QEBAQEAhIyMhXl5e",
		"JCQhIUAkJEBeIUAjJCRA", "JCQjISFAIyFAIyM=", "Xl5eIUAjIyEhIyM=",
	}
)

// saltFallbackLen is the number of characters the official app strips after a
// "//_//" marker when it does not recognize the trailing salt (FilmModel.decodeUrl).
const saltFallbackLen = 16

// FormatBytes renders a byte count as a human-readable string using binary
// (1024-based) units, e.g. 1536 -> "1.5 KB". Sizes <= 0 return "".
func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return ""
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

func decodeURL(url string) (string, error) {
	// New format: URL is already in plain text (no base64 encoding)
	if strings.HasPrefix(url, "[") || strings.HasPrefix(url, "http") {
		return url, nil
	}

	// Old format: base64 encoded with obfuscation patterns. For each "//_//"
	// marker, strip a known salt of its exact length; if none matches, fall
	// back to the app's behaviour of dropping the marker plus a fixed offset.
	url = strings.TrimPrefix(url, "#h")
	for i := 0; i < 60 && strings.Contains(url, "//_//"); i++ {
		idx := strings.Index(url, "//_//")
		after := url[idx+5:]
		matched := false
		for _, s := range knownSalts {
			if strings.HasPrefix(after, s) {
				url = url[:idx] + after[len(s):]
				matched = true
				break
			}
		}
		if !matched {
			// Unknown salt: app-style fallback (drop "//_//" + fixed offset).
			end := idx + 5 + saltFallbackLen
			if end > len(url) {
				end = len(url)
			}
			url = url[:idx] + url[end:]
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(url)
	if err != nil {
		// Fallback: if base64 decoding fails, return as-is
		return url, nil
	}
	return string(decoded), nil
}

// boolTo10 maps a boolean to the "1"/"0" string flags HDrezka AJAX endpoints expect.
func boolTo10(b bool) string {
	if b {
		return "1"
	}
	return "0"
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
	return r.getDocSolving(uri, r.solveAntiBot)
}

// getDocSolving fetches uri as a parsed document. When allowSolve is true and
// the response is an Anubis proof-of-work interstitial, it solves the challenge
// (which seeds the auth cookie into the jar) and retries once with solving
// disabled to avoid loops.
func (r *HDRezka) getDocSolving(uri string, allowSolve bool) (*goquery.Document, error) {
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	if isChallengePage(doc) {
		if allowSolve && isAnubisChallenge(doc) {
			if err := r.solveAnubis(doc, resp.Request.URL); err != nil {
				return nil, fmt.Errorf("failed to solve Anubis anti-bot challenge for %s: %w", uri, err)
			}
			return r.getDocSolving(uri, false)
		}
		return nil, fmt.Errorf("request to %s blocked by an anti-bot challenge (Cloudflare/DDoS-Guard/Anubis); this mirror needs a solved-challenge cookie from a browser (pass it via --cookies) or a different mirror", uri)
	}
	return doc, nil
}

// extractVideoID resolves the numeric post id of a video page from several
// sources, in order of reliability, so that markup changes on a single element
// do not break parsing. Every source requires the real page to have loaded; it
// falls back to the id embedded in the page's initCDN*Events call.
func extractVideoID(doc *goquery.Document, html string) string {
	if id := doc.Find(".b-userset__fav_holder").AttrOr("data-post_id", ""); id != "" {
		return id
	}
	if id := doc.Find("#post_id").AttrOr("value", ""); id != "" {
		return id
	}
	// The main post container is rendered as <div class="b-post" id="post-1171">.
	if postID := doc.Find(".b-post").AttrOr("id", ""); strings.HasPrefix(postID, "post-") {
		if id := strings.TrimPrefix(postID, "post-"); id != "" {
			return id
		}
	}
	if m := reInitCDNID.FindStringSubmatch(html); len(m) > 1 {
		return m[1]
	}
	return ""
}

// isChallengePage reports whether doc is an anti-bot interstitial rather than
// real site content. It recognizes Cloudflare ("Just a moment..."), DDoS-Guard
// and Anubis (proof-of-work, served from /.within.website/) challenges. Such
// pages are frequently served with HTTP 200, so callers must inspect the body,
// not only the status code.
func isChallengePage(doc *goquery.Document) bool {
	switch strings.TrimSpace(doc.Find("head > title").Text()) {
	case "Just a moment...", "DDoS-Guard", "DDOS-GUARD":
		return true
	}
	// Cloudflare Turnstile / managed challenge.
	if doc.Find(`script[src*="challenges.cloudflare.com"]`).Length() > 0 {
		return true
	}
	return isAnubisChallenge(doc)
}

// isAnubisChallenge reports whether doc is an Anubis proof-of-work interstitial
// (https://github.com/TecharoHQ/anubis): it injects #anubis_challenge and loads
// its assets from /.within.website/.
func isAnubisChallenge(doc *goquery.Document) bool {
	return doc.Find(`#anubis_challenge, script[src*=".within.website"], link[href*=".within.website"]`).Length() > 0
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
		// A subtitle entry may carry alternatives separated by " or "; the
		// site/app uses the last one (consistent with parseStreamFormats).
		if parts := strings.Split(value, " or "); len(parts) > 1 {
			value = strings.TrimSpace(parts[len(parts)-1])
		}
		subtitles[key] = value
	}
	return subtitles
}
