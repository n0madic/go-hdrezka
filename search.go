package hdrezka

import "github.com/PuerkitoBio/goquery"

// QuickSearch simple search for videos by query.
func (r *HDRezka) QuickSearch(query string) ([]*CoverItem, error) {
	searchURL := r.URL.JoinPath("/engine/ajax/search.php")

	q := searchURL.Query()
	q.Set("q", query)
	searchURL.RawQuery = q.Encode()

	items := make([]*CoverItem, 0)
	doc, err := getDoc(searchURL.String())
	if err != nil {
		return nil, err
	}

	doc.Find("div.b-search__live_section > ul > li").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a")
		rating := s.Find("span.rating").Text()
		s.Find("span.rating").Remove()
		items = append(items, &CoverItem{
			Description: link.Text(),
			Info:        rating,
			Title:       s.Find("span.enty").Text(),
			URL:         link.AttrOr("href", ""),
		})
	})

	return items, nil
}

// Search search for videos by query.
func (r *HDRezka) Search(query string, maxItems int) ([]*CoverItem, error) {
	searchURL := r.URL.JoinPath("/search/")

	q := searchURL.Query()
	q.Set("do", "search")
	q.Set("subaction", "search")
	q.Set("q", query)
	searchURL.RawQuery = q.Encode()

	return getItems(searchURL.String(), maxItems)
}
