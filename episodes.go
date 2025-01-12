package hdrezka

import (
	"errors"
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Episodes is a struct for seasons and episodes
type Episodes map[int]map[int]*Stream

// ListSeasons get list seasons for video.
func (e *Episodes) ListSeasons() []int {
	seasons := []int{}
	for k := range *e {
		seasons = append(seasons, k)
	}
	sort.Ints(seasons)
	return seasons
}

// ListEpisodes get list episodes for season.
func (e *Episodes) ListEpisodes(season int) []int {
	seasons := []int{}
	for k := range (*e)[season] {
		seasons = append(seasons, k)
	}
	sort.Ints(seasons)
	return seasons
}

// GetEpisodes get episodes for video.
func (t *Translation) GetEpisodes() (Episodes, error) {
	form := url.Values{
		"id":            {t.videoID},
		"translator_id": {t.ID},
		"action":        {"get_episodes"},
	}
	var data struct {
		Episodes string `json:"episodes"`
		Message  string `json:"message"`
		Success  bool   `json:"success"`
	}
	if err := t.r.getCDN(form, &data); err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, errors.New("failed to get episodes: " + data.Message)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data.Episodes))
	if err != nil {
		return nil, err
	}

	episodes := make(map[int]map[int]*Stream)
	doc.Find(".b-simple_episodes__list").Each(func(i int, s *goquery.Selection) {
		s.Find(".b-simple_episode__item").Each(func(i int, s *goquery.Selection) {
			season := parseInt(s.AttrOr("data-season_id", ""))
			episode := parseInt(s.AttrOr("data-episode_id", ""))
			url := s.AttrOr("data-cdn_url", "")
			if url == "null" {
				url = ""
			}
			if season > 0 {
				if episodes[season] == nil {
					episodes[season] = make(map[int]*Stream)
				}
				episodes[season][episode] = &Stream{URL: url}
			}
		})
	})
	return episodes, nil
}
