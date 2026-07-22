package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/n0madic/go-hdrezka"
)

// Page paths. Everything lives under /msx/ so the portal can share a host with
// other services behind a reverse proxy.
const (
	pathStart      = "/msx/start.json"
	pathMenu       = "/msx/menu.json"
	pathCovers     = "/msx/content/covers"
	pathNewest     = "/msx/content/newest"
	pathGenre      = "/msx/content/genre"
	pathCategories = "/msx/content/categories"
	pathYears      = "/msx/content/years"
	pathCountries  = "/msx/content/countries"
	pathVideo      = "/msx/content/video"
	pathSeasons    = "/msx/content/seasons"
	pathEpisodes   = "/msx/content/episodes"
	pathStream     = "/msx/content/stream"
	pathSearch     = "/msx/content/search"
)

// config holds the runtime options shared by all handlers.
type config struct {
	publicURL string
	maxItems  int
	useMP4    bool
	showSizes bool
}

// server serves the MSX portal on top of a single shared hdrezka client.
// After Init the client's fields are read-only and http.Client is safe for
// concurrent use, so no additional locking is required.
type server struct {
	rezka *hdrezka.HDRezka
	cache *cache
	cfg   config
}

func newServer(r *hdrezka.HDRezka, c *cache, cfg config) *server {
	return &server{rezka: r, cache: c, cfg: cfg}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, pathStart, http.StatusFound)
	})
	mux.HandleFunc(pathStart, s.handleStart)
	mux.HandleFunc(pathMenu, s.handleMenu)
	mux.HandleFunc(pathCovers, s.handleCovers)
	mux.HandleFunc(pathNewest, s.handleNewest)
	mux.HandleFunc(pathGenre, s.handleGenre)
	mux.HandleFunc(pathCategories, s.handleCategories)
	mux.HandleFunc(pathYears, s.handleYears)
	mux.HandleFunc(pathCountries, s.handleCountries)
	mux.HandleFunc(pathVideo, s.handleVideo)
	mux.HandleFunc(pathSeasons, s.handleSeasons)
	mux.HandleFunc(pathEpisodes, s.handleEpisodes)
	mux.HandleFunc(pathStream, s.handleStream)
	mux.HandleFunc(pathSearch, s.handleSearch)
	return logging(cors(mux))
}

// cors makes the documents readable by MSX, which runs as a web application on
// a foreign origin and would otherwise be blocked by the browser.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// logging records every request at info level. A request that never shows up
// here did not reach the server at all — which is the difference between a
// broken page and an action MSX refused to execute, and the first thing worth
// knowing when the TV reports "status: 0".
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// baseURL returns the absolute URL this server is reachable at, used to build
// links between pages. The Host header is the only reliable source when the TV
// talks to the server over the local network.
func (s *server) baseURL(r *http.Request) string {
	if s.cfg.publicURL != "" {
		return strings.TrimSuffix(s.cfg.publicURL, "/")
	}
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		slog.Error("encode response", "error", err)
	}
}

// writeContent renders a content page, replacing it with an error placeholder
// when the page could not be built. The HTTP status stays 200 because MSX does
// not render bodies of error responses, leaving the user with a blank screen.
func (s *server) writeContent(w http.ResponseWriter, headline string, content *Content, err error) {
	if err != nil {
		slog.Error("build content", "headline", headline, "error", err)
		writeJSON(w, errorContent(headline, err))
		return
	}
	writeJSON(w, content)
}

// navParams carries the navigation state threaded through the page links:
// which video, translation, season and episode the user has selected.
type navParams struct {
	VideoURL    string
	Translation string
	Season      int
	Episode     int
}

// values renders the navigation state back into query parameters.
func (n navParams) values() url.Values {
	q := url.Values{}
	if n.VideoURL != "" {
		q.Set("u", n.VideoURL)
	}
	if n.Translation != "" {
		q.Set("t", n.Translation)
	}
	if n.Season > 0 {
		q.Set("s", strconv.Itoa(n.Season))
	}
	if n.Episode > 0 {
		q.Set("e", strconv.Itoa(n.Episode))
	}
	return q
}

// parseNav extracts the navigation state from a request query.
func parseNav(q url.Values) (navParams, error) {
	nav := navParams{
		VideoURL:    q.Get("u"),
		Translation: q.Get("t"),
	}
	if nav.VideoURL == "" {
		return nav, fmt.Errorf("missing video url parameter %q", "u")
	}
	var err error
	if nav.Season, err = parseIntParam(q.Get("s")); err != nil {
		return nav, fmt.Errorf("invalid season: %w", err)
	}
	if nav.Episode, err = parseIntParam(q.Get("e")); err != nil {
		return nav, fmt.Errorf("invalid episode: %w", err)
	}
	return nav, nil
}

func parseIntParam(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
