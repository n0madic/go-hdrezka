package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/n0madic/go-hdrezka"
)

const testBase = "http://tv.local:8080"

func TestBuildStart(t *testing.T) {
	t.Parallel()

	start := buildStart(testBase)
	if start.Name != "HDrezka" {
		t.Errorf("name = %q, want HDrezka", start.Name)
	}
	want := "menu:" + testBase + pathMenu
	if start.Parameter != want {
		t.Errorf("parameter = %q, want %q", start.Parameter, want)
	}
}

func TestBuildMenu(t *testing.T) {
	t.Parallel()

	menu := buildMenu(testBase)
	if len(menu.Menu) != 4+len(genreTitles) {
		t.Fatalf("menu has %d items, want %d", len(menu.Menu), 4+len(genreTitles))
	}
	for _, item := range menu.Menu {
		// A menu item's "data" is the URL of the page to load. Any action
		// prefix makes MSX request an invalid URL and fail with "status: 0"
		// before the request ever reaches the server.
		if !strings.HasPrefix(item.Data, testBase+"/msx/") {
			t.Errorf("item %q data = %q, want a plain portal URL", item.Label, item.Data)
		}
		if _, err := url.ParseRequestURI(item.Data); err != nil {
			t.Errorf("item %q data = %q is not a valid URL: %v", item.Label, item.Data, err)
		}
		if item.Label == "" || item.Icon == "" {
			t.Errorf("item %+v is missing label or icon", item)
		}
	}
}

func TestSearchAction(t *testing.T) {
	t.Parallel()

	// The input plugin is documented for item actions only, so the action
	// carries the "content:" prefix and lives on a page button — never in the
	// "data" of a menu item, where MSX fails it before reaching the server.
	action := searchAction(testBase)
	if !strings.HasPrefix(action, "content:request:interaction:"+testBase+pathSearch) {
		t.Errorf("action = %q", action)
	}
	if !strings.Contains(action, "?q={INPUT}|search:3|ru|") {
		t.Errorf("action %q is missing the input placeholder or options", action)
	}
	if !strings.HasSuffix(action, "@http://"+msxPluginHost+msxInputPlugin) {
		t.Errorf("action %q must end with the plugin URL", action)
	}
}

// TestMsxPluginScheme guards the rule that broke search: a plugin page served
// over https cannot call back into an http portal, so the browser blocks the
// request as mixed content and MSX reports "status: 0".
func TestMsxPluginScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		base string
		want string
	}{
		{base: "http://192.168.1.5:8080", want: "http://" + msxPluginHost + msxInputPlugin},
		{base: "https://rezka.example", want: "https://" + msxPluginHost + msxInputPlugin},
	}

	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			t.Parallel()

			if got := msxPlugin(tt.base, msxInputPlugin); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if !strings.Contains(searchAction(tt.base), "@"+tt.want) {
				t.Errorf("search action does not use %q", tt.want)
			}
		})
	}
}

func TestParseCoverOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		want    hdrezka.CoverOption
		wantErr bool
	}{
		{
			name:  "defaults to all",
			query: "",
			want:  hdrezka.CoverOption{Type: hdrezka.CoverAll},
		},
		{
			name:  "new films",
			query: "type=new&genre=films",
			want:  hdrezka.CoverOption{Type: hdrezka.CoverNew, Genre: hdrezka.Films},
		},
		{
			name:  "category with filter",
			query: "type=category&genre=series&category=Драмы&filter=popular",
			want: hdrezka.CoverOption{
				Type:     hdrezka.CoverByCategory,
				Genre:    hdrezka.Series,
				Category: "Драмы",
				Filter:   hdrezka.FilterPopular,
			},
		},
		{
			name:  "by year",
			query: "type=year&year=2024&genre=cartoons",
			want:  hdrezka.CoverOption{Type: hdrezka.CoverByYear, Genre: hdrezka.Cartoons, Year: "2024"},
		},
		{
			name:  "by country",
			query: "type=country&country=США",
			want:  hdrezka.CoverOption{Type: hdrezka.CoverByCountry, Country: "США"},
		},
		{name: "unknown type", query: "type=bogus", wantErr: true},
		{name: "unknown filter", query: "filter=bogus", wantErr: true},
		{name: "unknown genre", query: "genre=bogus", wantErr: true},
		{name: "year without value", query: "type=year", wantErr: true},
		{name: "country without value", query: "type=country", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("bad test query: %v", err)
			}
			got, err := parseCoverOption(values)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVideoAction(t *testing.T) {
	t.Parallel()

	const mp4 = "https://cdn.example/video.mp4"
	subs := map[string]string{
		"Русский": "https://cdn.example/ru.vtt",
		"Klingon": "https://cdn.example/tlh.vtt",
	}

	tests := []struct {
		name       string
		mediaURL   string
		subtitles  map[string]string
		wantAction string
		wantProps  map[string]string
	}{
		{
			name:       "plain mp4 uses the built-in player",
			mediaURL:   mp4,
			wantAction: "video:" + mp4,
		},
		{
			name:       "hls uses the built-in player too",
			mediaURL:   "https://cdn.example/index.m3u8",
			wantAction: "video:https://cdn.example/index.m3u8",
		},
		{
			name:       "subtitles go through the plugin",
			mediaURL:   mp4,
			subtitles:  subs,
			wantAction: "video:plugin:" + msxPlugin(testBase, msxHTML5XPlugin) + "?url=" + url.QueryEscape(mp4),
			wantProps: map[string]string{
				"html5x:subtitle:s0:Klingon": "https://cdn.example/tlh.vtt",
				"html5x:subtitle:ru:Русский": "https://cdn.example/ru.vtt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			action, props := videoAction(testBase, tt.mediaURL, tt.subtitles)
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
			if !reflect.DeepEqual(props, tt.wantProps) {
				t.Errorf("properties = %v, want %v", props, tt.wantProps)
			}
		})
	}
}

func TestNavParamsRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		nav     navParams
		wantErr bool
	}{
		{name: "video only", nav: navParams{VideoURL: "https://hdrezka.co/films/drama/1-x.html"}},
		{name: "with translation", nav: navParams{VideoURL: "https://hdrezka.co/x.html", Translation: "56"}},
		{name: "with episode", nav: navParams{VideoURL: "https://hdrezka.co/x.html", Translation: "56", Season: 2, Episode: 13}},
		{name: "url with query", nav: navParams{VideoURL: "https://hdrezka.co/x.html?a=b&c=d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseNav(tt.nav.values())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.nav {
				t.Errorf("got %+v, want %+v", got, tt.nav)
			}
		})
	}
}

func TestParseNavErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query url.Values
	}{
		{name: "missing url", query: url.Values{}},
		{name: "bad season", query: url.Values{"u": {"x"}, "s": {"one"}}},
		{name: "bad episode", query: url.Values{"u": {"x"}, "e": {"last"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := parseNav(tt.query); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestSortQualities(t *testing.T) {
	t.Parallel()

	const premium = `<span class="pjs-prem-quality">1080p Ultra<img src="/prem.svg" alt=""></span>`
	formats := map[string]hdrezka.VideoFormat{
		"480p":   {},
		premium:  {},
		"1080p":  {},
		"2160p":  {},
		"720p":   {},
		"origin": {},
	}
	want := []string{"2160p", premium, "1080p", "720p", "480p", "origin"}
	if got := sortQualities(formats); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestVideoFacts(t *testing.T) {
	t.Parallel()

	video := &hdrezka.Video{
		Rating:     hdrezka.Rating{Score: 7.2},
		RatingIMDB: hdrezka.Rating{Score: 6.7},
		Duration:   "145 мин.",
		Categories: []string{"Фантастика", "Драмы"},
		Country:    []string{"США"},
	}

	if want := "HDrezka 7.2 · IMDb 6.7 · 145 мин. · Фантастика, Драмы · США"; videoFacts(video) != want {
		t.Errorf("facts = %q, want %q", videoFacts(video), want)
	}
	if got := videoFacts(&hdrezka.Video{}); got != "" {
		t.Errorf("empty video produced %q", got)
	}
}

func TestTruncateWords(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("слово ", 200)
	got := truncateWords(long, descriptionLimit)
	if n := len([]rune(got)); n > descriptionLimit+1 {
		t.Errorf("got %d runes, want at most %d", n, descriptionLimit+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated text %q should end with an ellipsis", got)
	}
	if short := "Коротко."; truncateWords(short, descriptionLimit) != short {
		t.Errorf("short text was modified")
	}
}

func TestJoinParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		parts []string
		want  string
	}{
		{parts: []string{"1080p", "премиум", "1.2 GB"}, want: "1080p · премиум · 1.2 GB"},
		{parts: []string{"Оригинал", ""}, want: "Оригинал"},
		{parts: []string{"", "  ", ""}, want: ""},
		{parts: []string{" Сезон 1 ", "10 серий"}, want: "Сезон 1 · 10 серий"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := joinParts(tt.parts...); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamURL(t *testing.T) {
	t.Parallel()

	const (
		mp4 = "https://cdn.example/video.mp4"
		hls = "https://cdn.example/video.mp4:hls:manifest.m3u8"
	)

	tests := []struct {
		name      string
		format    hdrezka.VideoFormat
		preferMP4 bool
		want      string
	}{
		{name: "hls by default", format: hdrezka.VideoFormat{MP4: mp4, HLS: hls}, want: hls},
		{name: "mp4 on request", format: hdrezka.VideoFormat{MP4: mp4, HLS: hls}, preferMP4: true, want: mp4},
		{name: "falls back to mp4", format: hdrezka.VideoFormat{MP4: mp4}, want: mp4},
		{name: "falls back to hls", format: hdrezka.VideoFormat{HLS: hls}, preferMP4: true, want: hls},
		{name: "nothing to play", format: hdrezka.VideoFormat{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := streamURL(tt.format, tt.preferMP4); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	// HDrezka appends the manifest to the file name, so an ".mp4" substring
	// must not be mistaken for a progressive file.
	if !isHLS(hls) {
		t.Errorf("isHLS(%q) = false", hls)
	}
	if isHLS(mp4) {
		t.Errorf("isHLS(%q) = true", mp4)
	}
}

func TestQualityTitleAndLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		quality   string
		wantTitle string
		wantLabel string
	}{
		{quality: "1080p", wantTitle: "1080p"},
		{
			quality:   `<span class="pjs-prem-quality">1080p Ultra<img src="/prem.svg" alt=""></span>`,
			wantTitle: "1080p Ultra",
			wantLabel: "премиум",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantTitle, func(t *testing.T) {
			t.Parallel()

			if got := qualityTitle(tt.quality); got != tt.wantTitle {
				t.Errorf("title = %q, want %q", got, tt.wantTitle)
			}
			if got := qualityLabel(tt.quality); got != tt.wantLabel {
				t.Errorf("label = %q, want %q", got, tt.wantLabel)
			}
		})
	}
}

func TestCoversContentJSON(t *testing.T) {
	t.Parallel()

	content := coversContent(testBase, "Фильмы", []*hdrezka.CoverItem{
		{Title: "Титул", Cover: "https://cdn/1.jpg", Info: " 2024 ", Description: "драма", URL: "https://hdrezka.co/films/1-x.html"},
		{Title: "Без ссылки"},
	})

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		// "list" scrolls vertically and fills whole rows of posters; "pages"
		// would lay them out as a single horizontally scrolling row.
		"type":     "list",
		"headline": "Фильмы",
		"template": map[string]any{
			"type":        "separate",
			"layout":      "0,0,2,3",
			"color":       "msx-glass",
			"imageFiller": "height-center",
		},
		"items": []any{
			map[string]any{
				"title":       "Титул",
				"titleFooter": "драма",
				"tag":         "2024",
				"image":       "https://cdn/1.jpg",
				"action":      "content:" + testBase + pathVideo + "?u=" + url.QueryEscape("https://hdrezka.co/films/1-x.html"),
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestVideoCard(t *testing.T) {
	t.Parallel()

	video := &hdrezka.Video{
		Title:       "Титул",
		Year:        "2026",
		Cover:       "https://cdn/poster.jpg",
		Description: "Синопсис.",
		Rating:      hdrezka.Rating{Score: 7.2},
		Director:    []hdrezka.Person{{Name: "Режиссёр Р."}},
		Cast:        []hdrezka.Person{{Name: "Актёр А."}, {Name: "Актриса Б."}},
		Translation: []*hdrezka.Translation{{ID: "56", Name: "Дубляж", IsDefault: true}},
	}

	nav := navParams{VideoURL: "https://hdrezka.co/films/1-x.html"}
	card := videoCard(testBase, nav, video)
	if len(card.Items) != 4 {
		t.Fatalf("card has %d items, want poster + facts + synopsis + credits: %+v", len(card.Items), card.Items)
	}

	poster, facts, synopsis, credits := card.Items[0], card.Items[1], card.Items[2], card.Items[3]
	if poster.Image != video.Cover {
		t.Errorf("poster = %+v", poster)
	}
	// "cover"/"default" and the "height-*" fillers scale a portrait poster to
	// the full height of a taller area and crop its sides.
	switch poster.ImageFiller {
	case "fit", "width", "width-top", "width-center", "width-bottom":
	default:
		t.Errorf("imageFiller = %q crops the poster", poster.ImageFiller)
	}
	if !strings.Contains(facts.Text, "HDrezka 7.2") {
		t.Errorf("facts = %q", facts.Text)
	}
	if synopsis.Text != video.Description {
		t.Errorf("synopsis = %q, want %q", synopsis.Text, video.Description)
	}
	// MSX refuses to render a page whose items are all unselectable
	// ("Page[0] has no selectable items"), so the card must keep exactly one
	// focusable element — the poster acting as a play button.
	selectable := 0
	for _, item := range card.Items {
		if item.Action != "" {
			selectable++
			if item.Type == "space" {
				t.Errorf("item %+v has an action but is unselectable", item)
			}
		}
	}
	if selectable != 1 {
		t.Errorf("card has %d selectable items, want exactly 1", selectable)
	}
	if !strings.HasPrefix(poster.Action, "content:"+testBase+pathSeasons) {
		t.Errorf("poster action = %q, want the default translation", poster.Action)
	}
	if !strings.Contains(credits.Text, "Режиссёр Р.") || !strings.Contains(credits.Text, "Актёр А.") {
		t.Errorf("credits = %q", credits.Text)
	}
	// The page headline already carries the title; repeating it in a block
	// printed it on screen twice.
	for _, item := range card.Items {
		if item.Title != "" {
			t.Errorf("item %+v repeats the title", item)
		}
	}

	// No two blocks may claim the same cell of the 12x6 grid, and none may
	// leave it — overlapping blocks print on top of each other.
	occupied := map[[2]int]bool{}
	for _, item := range card.Items {
		var x, y, w, h int
		if _, err := fmt.Sscanf(item.Layout, "%d,%d,%d,%d", &x, &y, &w, &h); err != nil {
			t.Fatalf("layout %q: %v", item.Layout, err)
		}
		if x < 0 || y < 0 || x+w > 12 || y+h > 6 {
			t.Errorf("layout %q leaves the 12x6 grid", item.Layout)
		}
		for cx := x; cx < x+w; cx++ {
			for cy := y; cy < y+h; cy++ {
				if occupied[[2]int{cx, cy}] {
					t.Errorf("layout %q overlaps another block at %d,%d", item.Layout, cx, cy)
				}
				occupied[[2]int{cx, cy}] = true
			}
		}
	}

	// Without a poster the card still needs a selectable element, so a plain
	// play button takes its place.
	bare := videoCard(testBase, nav, &hdrezka.Video{Title: "Без обложки"})
	if bare.Items[0].Type != "button" || bare.Items[0].Icon == "" {
		t.Errorf("fallback item = %+v, want a play button", bare.Items[0])
	}
	// Even a video without any translation must keep a working action.
	if bare.Items[0].Action == "" {
		t.Error("fallback item has no action, the card would not render at all")
	}
}

func TestErrorContent(t *testing.T) {
	t.Parallel()

	content := errorContent("", http.ErrNoLocation)
	if content.Headline != "Ошибка" {
		t.Errorf("headline = %q", content.Headline)
	}
	if len(content.Items) != 2 || content.Items[0].Text != http.ErrNoLocation.Error() {
		t.Fatalf("items = %+v, want the message plus a back button", content.Items)
	}
	// The message block must stay unselectable and iconless, so it cannot
	// inherit the play arrow of a list template.
	if content.Items[0].Type != "space" || content.Items[0].Icon != "" {
		t.Errorf("message item = %+v", content.Items[0])
	}
	if content.Items[1].Action != "back" {
		t.Errorf("back item = %+v", content.Items[1])
	}
}

func TestBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		publicURL string
		headers   map[string]string
		want      string
	}{
		{name: "from host header", want: "http://tv.local:8080"},
		{name: "public url wins", publicURL: "https://rezka.example/", want: "https://rezka.example"},
		{
			name:    "forwarded headers",
			headers: map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "portal.example"},
			want:    "https://portal.example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &server{cfg: config{publicURL: tt.publicURL}}
			req := httptest.NewRequest(http.MethodGet, "http://tv.local:8080"+pathMenu, nil)
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}
			if got := s.baseURL(req); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHandleStartAndMenu exercises the routing and CORS layers end to end; both
// pages are built without touching the site.
func TestHandleStartAndMenu(t *testing.T) {
	t.Parallel()

	handler := (&server{cfg: config{}}).routes()

	for _, path := range []string{pathStart, pathMenu} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://tv.local:8080"+path, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", path, rec.Code)
		}
		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("%s: CORS origin = %q", path, origin)
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Errorf("%s: invalid JSON: %v", path, err)
		}
	}
}

func TestHandleRootRedirect(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	(&server{}).routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://tv.local/", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != pathStart {
		t.Errorf("location = %q, want %q", got, pathStart)
	}
}

// TestHandleSearchCached serves a search page from a pre-populated cache, so
// the handler is covered without hitting the site.
func TestHandleSearchCached(t *testing.T) {
	t.Parallel()

	c := newCache(time.Minute)
	c.set("search:матрица", []*hdrezka.CoverItem{
		{Title: "Матрица", URL: "https://hdrezka.co/films/1-matrix.html"},
	})
	s := &server{cache: c, cfg: config{maxItems: 10}}

	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://tv.local"+pathSearch+"?q=матрица", nil))

	var content Content
	if err := json.Unmarshal(rec.Body.Bytes(), &content); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if content.Headline != "Поиск: матрица" {
		t.Errorf("headline = %q", content.Headline)
	}
	if len(content.Items) != 1 || content.Items[0].Title != "Матрица" {
		t.Fatalf("items = %+v", content.Items)
	}
	if !strings.HasPrefix(content.Items[0].Action, "content:http://tv.local"+pathVideo) {
		t.Errorf("action = %q", content.Items[0].Action)
	}
}

func TestHandleSearchEmptyQuery(t *testing.T) {
	t.Parallel()

	s := &server{cache: newCache(time.Minute)}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://tv.local"+pathSearch, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 so MSX renders the message", rec.Code)
	}
	var content Content
	if err := json.Unmarshal(rec.Body.Bytes(), &content); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(content.Items) != 1 || content.Items[0].Action != searchAction("http://tv.local") {
		t.Errorf("items = %+v, want the input dialog action", content.Items)
	}
}

func TestCacheExpiry(t *testing.T) {
	t.Parallel()

	c := newCache(time.Minute)
	calls := 0
	load := func() (string, error) {
		calls++
		return "value", nil
	}

	for range 3 {
		got, err := cached(c, "key", load)
		if err != nil || got != "value" {
			t.Fatalf("got %q, err %v", got, err)
		}
	}
	if calls != 1 {
		t.Errorf("loader called %d times, want 1", calls)
	}

	c.entries.Store("key", cacheEntry{value: "stale", expires: time.Now().Add(-time.Second)})
	if _, err := cached(c, "key", load); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("loader called %d times after expiry, want 2", calls)
	}
}
