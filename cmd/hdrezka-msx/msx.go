package main

import (
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/n0madic/go-hdrezka"
)

// Media Station X object model. MSX builds its whole UI from JSON documents
// served over plain HTTP; the structures below cover the subset of the format
// this portal emits. See https://msx.benzac.de/wiki/ for the full spec.

const (
	msxPluginHost = "msx.benzac.de"
	// msxInputPlugin is the stock input dialog used for the search page.
	msxInputPlugin = "/interaction/input.html"
	// msxHTML5XPlugin is the stock HTML5 player plugin. It is the only player
	// that accepts external subtitle tracks.
	msxHTML5XPlugin = "/plugins/html5x.html"
)

// msxPlugin builds the URL of a stock MSX plugin using the same scheme as the
// portal itself.
//
// This matters for plugins that call back into this server: a plugin page
// loaded over https cannot request an http portal — the browser blocks it as
// mixed content and the request never arrives, which MSX reports as
// "Server responded with status: 0".
func msxPlugin(base, path string) string {
	scheme := "https"
	if strings.HasPrefix(base, "http://") {
		scheme = "http"
	}
	return scheme + "://" + msxPluginHost + path
}

// Start is the MSX Start Object — the document a user points MSX at.
type Start struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Parameter string `json:"parameter"`
}

// MenuItem is a single entry of the Menu Root Object.
type MenuItem struct {
	ID    string `json:"id,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Label string `json:"label,omitempty"`
	Data  string `json:"data,omitempty"`
}

// Menu is the MSX Menu Root Object rendered as the left-hand menu.
type Menu struct {
	Headline string     `json:"headline,omitempty"`
	Menu     []MenuItem `json:"menu"`
}

// Item is an entry of a Content Root Object. Fields left empty are inherited
// from the content template, which is why almost everything is omitempty.
type Item struct {
	Type        string `json:"type,omitempty"`
	Layout      string `json:"layout,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Title       string `json:"title,omitempty"`
	TitleFooter string `json:"titleFooter,omitempty"`
	Label       string `json:"label,omitempty"`
	Tag         string `json:"tag,omitempty"`
	// Text is a single string. MSX also accepts an array, but it concatenates
	// the entries without any separator instead of breaking lines, so
	// multi-line blocks are built from separate items with their own layout.
	Text        string            `json:"text,omitempty"`
	Image       string            `json:"image,omitempty"`
	ImageFiller string            `json:"imageFiller,omitempty"`
	PlayerLabel string            `json:"playerLabel,omitempty"`
	Action      string            `json:"action,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// Page is an MSX Content Page Object: a manually laid out group of items.
// Unlike templated items — whose own layout is ignored — items of a page are
// positioned by their layout on the content grid.
type Page struct {
	Offset string `json:"offset,omitempty"`
	Items  []Item `json:"items"`
}

// Content is the MSX Content Root Object — one page of the portal.
//
// The "background" property is deliberately absent: MSX offers no way to
// control how a background image is scaled, and the only artwork HDrezka
// provides is a portrait poster, which gets stretched across the screen.
type Content struct {
	Type     string `json:"type,omitempty"`
	Headline string `json:"headline,omitempty"`
	Header   *Page  `json:"header,omitempty"`
	Template *Item  `json:"template,omitempty"`
	Items    []Item `json:"items"`
}

// posterTemplate is the item template of a cover grid: a portrait poster with
// the title underneath.
//
// The content grid is 12 columns by 6 rows, so a 2×3 poster puts six covers in
// a row and two full rows on screen. Together with a "list" content type (which
// scrolls vertically) that shows twelve covers at once instead of the single
// horizontally scrolling row a "pages" content produces.
func posterTemplate() *Item {
	return &Item{
		Type:        "separate",
		Layout:      "0,0,2,3",
		Color:       "msx-glass",
		ImageFiller: "height-center",
	}
}

// listTemplate is the item template of a plain full-width option list. An
// empty icon leaves the property unset, so pages that mix plain text blocks
// with selectable rows can put the icon on the rows only — a templated icon is
// inherited by every item, including text blocks that must not show one.
func listTemplate(icon string) *Item {
	return &Item{
		Type:   "default",
		Layout: "0,0,8,1",
		Icon:   icon,
		Color:  "msx-glass",
	}
}

// joinParts concatenates the non-empty parts with " · ". Rows of a "default"
// item render the title at the bottom left and the label at the top left,
// where it collides with the centered icon — so secondary details go into the
// title instead of a separate label.
func joinParts(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, strings.TrimSpace(part))
		}
	}
	return strings.Join(kept, " · ")
}

// contentAction builds a "content:" action pointing at a page of this server.
func contentAction(base, path string, query url.Values) string {
	return "content:" + pageURL(base, path, query)
}

// pageURL joins the server base URL, a page path and its query parameters.
func pageURL(base, path string, query url.Values) string {
	u := strings.TrimSuffix(base, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

// searchAction builds the action that opens the stock MSX input dialog and
// feeds the entered text to the search page as the "q" parameter. The options
// after the URL are {TYPE}|{DEFAULT_LANG}|{HEADLINE}: "search:3" submits the
// query automatically once at least three characters have been typed.
//
// The input plugin is documented for item actions only, so it must not be put
// into the "data" of a menu item: MSX fails the request before it reaches this
// server. The menu links to the search page instead, which offers this action
// on a button.
func searchAction(base string) string {
	target := pageURL(base, pathSearch, nil) + "?q={INPUT}"
	return "content:request:interaction:" + target +
		"|search:3|ru|Поиск@" + msxPlugin(base, msxInputPlugin)
}

// videoAction builds the playback action for a media URL. Without subtitles
// the stream is handed to the built-in player, which is what MSX documents for
// both MP4 and HLS and which lets every platform use its native player. Only
// subtitle tracks require the HTML5X plugin.
func videoAction(base, mediaURL string, subtitles map[string]string) (string, map[string]string) {
	if len(subtitles) == 0 {
		return "video:" + mediaURL, nil
	}
	action := "video:plugin:" + msxPlugin(base, msxHTML5XPlugin) + "?url=" + url.QueryEscape(mediaURL)
	props := make(map[string]string, len(subtitles))
	for i, lang := range sortedKeys(subtitles) {
		props["html5x:subtitle:"+subtitleCode(lang, i)+":"+lang] = subtitles[lang]
	}
	return action, props
}

// subtitleCode returns the ISO 639-1 code expected by the HTML5X plugin for a
// subtitle track name. Unknown names fall back to a positional code so the
// property keys stay unique.
func subtitleCode(lang string, index int) string {
	if code, ok := hdrezka.SubtitleLanguageCode(lang); ok {
		return code
	}
	return "s" + strconv.Itoa(index)
}

// sortedKeys returns the map keys in a stable order so generated JSON does not
// depend on map iteration order.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
