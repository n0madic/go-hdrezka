package main

import (
	"net/http"
	"strings"

	"github.com/n0madic/go-hdrezka"
)

// handleSearch renders search results. It is called by the stock MSX input
// plugin, which substitutes the typed text into the "q" parameter.
func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		// The page was opened without a query (e.g. straight from a browser):
		// offer the input dialog instead of an error.
		writeJSON(w, &Content{
			Type:     "list",
			Headline: "Поиск",
			Template: listTemplate("search"),
			Items:    []Item{{Title: "Введите запрос", Action: searchAction(base)}},
		})
		return
	}

	headline := "Поиск: " + query
	covers, err := cached(s.cache, "search:"+query, func() ([]*hdrezka.CoverItem, error) {
		return s.rezka.Search(query, s.cfg.maxItems)
	})
	if err != nil {
		s.writeContent(w, headline, nil, err)
		return
	}
	if len(covers) == 0 {
		writeJSON(w, &Content{
			Type:     "list",
			Headline: headline,
			Template: listTemplate("search"),
			Items:    []Item{{Title: "Ничего не найдено", Action: "back"}},
		})
		return
	}
	writeJSON(w, coversContent(base, headline, covers))
}
