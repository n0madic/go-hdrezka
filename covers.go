package hdrezka

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CoverOption is a struct for cover options
type CoverOption struct {
	Category string
	Country  string
	Filter   Filter
	Genre    Genre
	Type     Cover
	Year     string
}

// CoverItem is a struct for cover item
type CoverItem struct {
	Cover       string
	Description string
	Info        string
	Title       string
	URL         string
}

func (c *CoverItem) String() string {
	output := ""
	output += fmt.Sprintf("Title: %s\n", c.Title)
	if c.Description != "" {
		output += fmt.Sprintf("Description: %s\n", c.Description)
	}
	if c.Info != "" {
		output += fmt.Sprintf("Info: %s\n", c.Info)
	}
	if c.Cover != "" {
		output += fmt.Sprintf("Cover: %s\n", c.Cover)
	}
	output += fmt.Sprintf("URL: %s\n", c.URL)
	return output
}

// GetCoversURL generate video URL by options.
func (r *HDRezka) GetCoversURL(opts CoverOption) (string, error) {
	uri := []string{"/"}
	switch opts.Type {
	case CoverByCategory:
		uri = []string{"/" + string(opts.Genre) + "/"}
	case CoverByCountry:
		uri = []string{"/country/", opts.Country}
	case CoverByYear:
		uri = []string{"/year/", opts.Year}
	case CoverNew:
		uri = []string{"/new/"}
	}

	if (opts.Type == CoverByCategory || opts.Type == CoverBest) && opts.Category != "" {
		cat, found := r.Categories[opts.Genre][opts.Category]
		if !found {
			return "", fmt.Errorf("category %s not found", opts.Category)
		}
		if opts.Type == CoverBest {
			uri = strings.Split(cat, "/")
			uri = append(uri, "")
			copy(uri[3:], uri[2:])
			uri[2] = "best"
			if opts.Year != "" {
				uri = append(uri, opts.Year+"/")
			}
		} else {
			uri = []string{cat}
		}
	}

	coverURL := r.URL.JoinPath(uri...)

	q := coverURL.Query()
	if opts.Filter != "" {
		q.Set("filter", string(opts.Filter))
	}
	if opts.Type == CoverByCountry || opts.Type == CoverByYear || opts.Type == CoverNew || opts.Type == CoverAll {
		switch opts.Genre {
		case Films:
			q.Set("genre", "1")
		case Series:
			q.Set("genre", "2")
		case Cartoons:
			q.Set("genre", "3")
		case Show:
			q.Set("genre", "4")
		case Anime:
			q.Set("genre", "82")
		}
	}
	coverURL.RawQuery = q.Encode()

	if !strings.HasSuffix(coverURL.Path, "/") {
		coverURL.Path += "/"
	}

	return coverURL.String(), nil
}

// GetCovers returns video covers with options.
func (r *HDRezka) GetCovers(opts CoverOption, maxItems int) ([]*CoverItem, error) {
	uri, err := r.GetCoversURL(opts)
	if err != nil {
		return nil, err
	}
	return getItems(uri, maxItems)
}

// GetCoversNewest returns newest video covers by genres.
func (r *HDRezka) GetCoversNewest(genre Genre) ([]*CoverItem, error) {
	id := "0"
	switch genre {
	case Films:
		id = "1"
	case Series:
		id = "2"
	case Cartoons:
		id = "3"
	case Anime:
		id = "82"
	}

	uri := r.URL.JoinPath("/engine/ajax/get_newest_slider_content.php").String()
	resp, err := http.PostForm(uri, url.Values{"id": {id}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	items := []*CoverItem{}
	doc.Find("div.b-content__inline_item").Each(func(i int, s *goquery.Selection) {
		link := s.Find("div.b-content__inline_item-link > a")
		info, err := s.Find("span.info").Html()
		if err == nil && info != "" {
			info = strings.ReplaceAll(info, "<br/>", " ")
		}
		items = append(items, &CoverItem{
			Cover:       s.Find("div.b-content__inline_item-cover > a > img").AttrOr("src", ""),
			Description: strings.ReplaceAll(s.Find("div.b-content__inline_item-link > div").Text(), " - ...", ""),
			Info:        info,
			Title:       link.Text(),
			URL:         link.AttrOr("href", ""),
		})
	})
	return items, nil
}
