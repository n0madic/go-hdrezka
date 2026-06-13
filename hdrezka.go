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
	"https://hdrzk.org",
}

// browserCookies mimic the cookies a real browser/session sends to HDrezka on
// every page and AJAX request. The official app seeds the same set; without
// them some endpoints behave differently for anonymous clients.
var browserCookies = []*http.Cookie{
	{Name: "allowed_comments", Value: "1", Path: "/"},
	{Name: "_ym_isad", Value: "1", Path: "/"},
	{Name: "_ym_visorc", Value: "b", Path: "/"},
	{Name: "dle_newpm", Value: "0", Path: "/"},
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

	mirrors        []string
	proxyAddr      string
	resolverAddr   string
	persistSession bool
	initialized    bool
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

// WithPersistentSession requests a persistent login: Login will ask the site
// for long-lived dle_user_id / dle_password cookies (login_not_save=0) instead
// of a session-only cookie. It only affects Login, so it does not invalidate
// the cached transport / mirror state. Pair with ExportCookies to persist the
// session across runs.
func (r *HDRezka) WithPersistentSession() *HDRezka {
	r.persistSession = true
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

	// Seed browser-like cookies for the active host. Auth cookies (PHPSESSID,
	// dle_user_id, ...) use different names, so the jar merges both sets; CDN
	// hosts differ, so the jar never leaks these to them.
	r.Client.Jar.SetCookies(r.URL, browserCookies)

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

// ExportCookies serializes the cookies the client currently holds for the
// active site URL into a "name=value;name=value;..." string — the same format
// SetCookies accepts. Use it to persist a logged-in session across runs.
func (r *HDRezka) ExportCookies() string {
	cookies := r.Client.Jar.Cookies(r.URL)
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, ";")
}

// IsPremiumUser reports whether the currently authenticated session belongs to
// a premium account. It fetches the site home page and inspects the body class
// (mirrors UserModel.checkPremiumUser in the official app). Requires a prior
// Login or SetCookies; an anonymous session returns false.
func (r *HDRezka) IsPremiumUser() (bool, error) {
	doc, err := r.getDoc(r.URL.String())
	if err != nil {
		return false, err
	}
	return doc.Find("body").HasClass("b-premium_user__body"), nil
}
