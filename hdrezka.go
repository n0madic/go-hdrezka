// Package hdrezka site parser.
package hdrezka

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
)

// defaultMirrors is used when WithMirrors has not been configured.
var defaultMirrors = []string{
	"https://hdrezka.ag",
	"https://rezka.ag",
}

// HDRezka is a struct for working with hdrezka site
type HDRezka struct {
	// URL is a base url for hdrezka site
	URL *url.URL
	// Categories is a map of categories by genre and their urls
	Categories map[Genre]map[string]string
	// Years is list of years for filtering
	Years []string
	// Client is the HTTP client used for all site requests. Its cookie jar
	// stores authentication cookies populated by Login or SetCookies.
	Client *http.Client

	mirrors      []string
	proxyAddr    string
	resolverAddr string
	initialized  bool
}

func (r *HDRezka) getCDN(form url.Values, data interface{}) error {
	cdnURL := r.URL.JoinPath("/ajax/get_cdn_series/").String() + "?t=" + strconv.FormatInt(time.Now().UnixNano(), 10)
	req, err := http.NewRequest(http.MethodPost, cdnURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", r.URL.String()+"/")

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(&data)
}

// New creates a new HDRezka. It performs no network I/O — configure the
// instance with WithMirrors / WithProxy / WithResolver and then call Init
// to probe mirrors and populate URL / Categories / Years before using
// any other method.
func New() *HDRezka {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return &HDRezka{
		Categories: make(map[Genre]map[string]string),
		Client:     &http.Client{Jar: jar},
	}
}

// WithMirrors sets the list of mirror URLs to probe. The first reachable
// one becomes the active base URL. If WithMirrors is not called, an
// internal default list is used.
func (r *HDRezka) WithMirrors(mirrors ...string) *HDRezka {
	r.mirrors = mirrors
	r.initialized = false
	return r
}

// Init builds the HTTP transport from the current proxy / resolver settings
// and probes the configured mirrors to discover a working base URL. It is
// idempotent — repeated calls are a no-op until any With* setter
// invalidates the cached state. Must be called once before using GetVideo
// / GetCovers / Search / Login / SetCookies and other site methods.
func (r *HDRezka) Init() error {
	if r.initialized {
		return nil
	}

	transport, err := buildTransport(r.proxyAddr, r.resolverAddr)
	if err != nil {
		return err
	}
	r.Client.Transport = transport

	mirrors := r.mirrors
	if len(mirrors) == 0 {
		mirrors = defaultMirrors
	}

	var doc *goquery.Document
	for _, mirror := range mirrors {
		u, err := url.Parse(mirror)
		if err != nil {
			return err
		}
		uri := u.ResolveReference(&url.URL{Path: "/"})
		uri.Scheme = "https"
		r.URL = uri
		doc, err = r.getDoc(uri.String())
		if err == nil {
			break
		}
		r.URL = nil
	}

	if r.URL == nil {
		return errors.New("no working mirrors found")
	}

	r.Categories[Films] = getCategory("li.b-topnav__item.i1 > div > div > ul.left > li", doc)
	r.Categories[Series] = getCategory("li.b-topnav__item.i2 > div > div > ul.left > li", doc)
	r.Categories[Cartoons] = getCategory("li.b-topnav__item.i3 > div > div > ul.left > li", doc)
	r.Categories[Anime] = getCategory("li.b-topnav__item.i5 > div > div > ul.left > li", doc)
	r.Categories[Show] = categoriesShow

	r.Years = r.Years[:0]
	doc.Find("#find-best-block-1 > div > select.select-year > option").Each(func(i int, s *goquery.Selection) {
		if s.Text() == "за все время" {
			return
		}
		r.Years = append(r.Years, s.Text())
	})

	r.initialized = true
	return nil
}
