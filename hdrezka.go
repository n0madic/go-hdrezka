// Package hdrezka site parser.
package hdrezka

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HDRezka is a struct for working with hdrezka site
type HDRezka struct {
	// URL is a base url for hdrezka site
	URL *url.URL
	// Categories is a map of categories by genre and their urls
	Categories map[Genre]map[string]string
	// Years is list of years for filtering
	Years []string
}

func (r *HDRezka) getCDN(form url.Values, data interface{}) error {
	cdnURL := r.URL.JoinPath("/ajax/get_cdn_series/").String() + "?t=" + strconv.FormatInt(time.Now().UnixNano(), 10)
	resp, err := http.PostForm(cdnURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(&data)
}

// New create new instance of HDRezka
// mirrors - list of site mirrors, first working will be used
func New(mirrors ...string) (*HDRezka, error) {
	hdrezka := &HDRezka{
		Categories: make(map[Genre]map[string]string),
	}

	var doc *goquery.Document
	for _, mirror := range mirrors {
		u, err := url.Parse(mirror)
		if err != nil {
			return nil, err
		}
		uri := u.ResolveReference(&url.URL{Path: "/"})
		uri.Scheme = "https"
		doc, err = getDoc(uri.String())
		if err == nil {
			hdrezka.URL = uri
			break
		}
	}

	if hdrezka.URL == nil {
		return nil, errors.New("no working mirrors found")
	}

	hdrezka.Categories[Films] = getCategory("li.b-topnav__item.i1 > div > div > ul.left > li", doc)
	hdrezka.Categories[Series] = getCategory("li.b-topnav__item.i2 > div > div > ul.left > li", doc)
	hdrezka.Categories[Cartoons] = getCategory("li.b-topnav__item.i3 > div > div > ul.left > li", doc)
	hdrezka.Categories[Anime] = getCategory("li.b-topnav__item.i5 > div > div > ul.left > li", doc)
	hdrezka.Categories[Show] = categoriesShow

	doc.Find("#find-best-block-1 > div > select.select-year > option").Each(func(i int, s *goquery.Selection) {
		if s.Text() == "за все время" {
			return
		}
		hdrezka.Years = append(hdrezka.Years, s.Text())
	})

	return hdrezka, nil
}
