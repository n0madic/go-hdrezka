package main

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/n0madic/go-hdrezka"
)

// genreTitles lists the site sections in menu order together with their labels.
var genreTitles = []struct {
	Genre hdrezka.Genre
	Title string
	Icon  string
}{
	{hdrezka.Films, "Фильмы", "movie"},
	{hdrezka.Series, "Сериалы", "tv"},
	{hdrezka.Cartoons, "Мультфильмы", "child-care"},
	{hdrezka.Anime, "Аниме", "stars"},
	{hdrezka.Show, "Шоу", "live-tv"},
}

// filterTitles lists the catalog sort orders offered on a section page.
var filterTitles = []struct {
	Filter hdrezka.Filter
	Title  string
	Icon   string
}{
	{hdrezka.FilterLast, "Последние", "new-releases"},
	{hdrezka.FilterPopular, "Популярные", "trending-up"},
	{hdrezka.FilterWatching, "Сейчас смотрят", "visibility"},
}

// countries the site indexes at /country/<name>/. HDrezka has no machine
// readable list, so the most used ones are hardcoded.
var countries = []string{
	"США", "Россия", "СССР", "Украина", "Великобритания", "Франция", "Германия",
	"Италия", "Испания", "Канада", "Япония", "Корея Южная", "Китай", "Индия",
	"Турция", "Австралия", "Швеция", "Польша",
}

// genreByName resolves the "genre" query parameter.
func genreByName(name string) (hdrezka.Genre, error) {
	if name == "" {
		return hdrezka.All, nil
	}
	for _, g := range genreTitles {
		if string(g.Genre) == name {
			return g.Genre, nil
		}
	}
	return hdrezka.All, fmt.Errorf("unknown genre %q", name)
}

// genreTitle returns the human readable name of a section.
func genreTitle(genre hdrezka.Genre) string {
	for _, g := range genreTitles {
		if g.Genre == genre {
			return g.Title
		}
	}
	return "Все"
}

// coverTypes maps the "type" query parameter onto the library cover types.
var coverTypes = map[string]hdrezka.Cover{
	"all":      hdrezka.CoverAll,
	"category": hdrezka.CoverByCategory,
	"country":  hdrezka.CoverByCountry,
	"year":     hdrezka.CoverByYear,
	"best":     hdrezka.CoverBest,
	"new":      hdrezka.CoverNew,
}

// coverFilters maps the "filter" query parameter onto the library filters.
var coverFilters = map[string]hdrezka.Filter{
	"":         hdrezka.Filter(""),
	"last":     hdrezka.FilterLast,
	"popular":  hdrezka.FilterPopular,
	"watching": hdrezka.FilterWatching,
}

// parseCoverOption builds the library cover options from a request query.
func parseCoverOption(q url.Values) (hdrezka.CoverOption, error) {
	opts := hdrezka.CoverOption{
		Category: q.Get("category"),
		Country:  q.Get("country"),
		Year:     q.Get("year"),
	}

	coverType, ok := coverTypes[cmpOr(q.Get("type"), "all")]
	if !ok {
		return opts, fmt.Errorf("unknown cover type %q", q.Get("type"))
	}
	opts.Type = coverType

	filter, ok := coverFilters[q.Get("filter")]
	if !ok {
		return opts, fmt.Errorf("unknown filter %q", q.Get("filter"))
	}
	opts.Filter = filter

	genre, err := genreByName(q.Get("genre"))
	if err != nil {
		return opts, err
	}
	opts.Genre = genre

	if opts.Type == hdrezka.CoverByCountry && opts.Country == "" {
		return opts, fmt.Errorf("country is required for type=country")
	}
	if opts.Type == hdrezka.CoverByYear && opts.Year == "" {
		return opts, fmt.Errorf("year is required for type=year")
	}
	return opts, nil
}

// cmpOr returns value when it is not empty, otherwise fallback.
func cmpOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// buildStart renders the Start Object pointing MSX at the portal menu.
func buildStart(base string) Start {
	return Start{
		Name:      "HDrezka",
		Version:   "1.0.0",
		Parameter: "menu:" + pageURL(base, pathMenu, nil),
	}
}

// buildMenu renders the root menu: search, cross-section shortcuts and one
// entry per site section.
//
// The "data" property of a menu item is the URL of the page to load, not an
// action: prefixing it with "content:" makes MSX request a URL that starts
// with "content:" and fail before any request reaches this server. Only
// request actions (the search dialog) may be used in their place.
func buildMenu(base string) Menu {
	items := []MenuItem{
		{ID: "search", Icon: "search", Label: "Поиск", Data: pageURL(base, pathSearch, nil)},
		{ID: "new", Icon: "fiber-new", Label: "Новинки", Data: pageURL(base, pathCovers, url.Values{"type": {"new"}})},
		{ID: "watching", Icon: "whatshot", Label: "Сейчас смотрят", Data: pageURL(base, pathCovers, url.Values{"type": {"all"}, "filter": {"watching"}})},
		{ID: "popular", Icon: "trending-up", Label: "Популярное", Data: pageURL(base, pathCovers, url.Values{"type": {"all"}, "filter": {"popular"}})},
	}
	for _, g := range genreTitles {
		items = append(items, MenuItem{
			ID:    string(g.Genre),
			Icon:  g.Icon,
			Label: g.Title,
			Data:  pageURL(base, pathGenre, url.Values{"genre": {string(g.Genre)}}),
		})
	}
	return Menu{Headline: "HDrezka", Menu: items}
}

// errorContent renders a failure as a readable page instead of a blank screen.
func errorContent(headline string, err error) *Content {
	return &Content{
		Type:     "list",
		Headline: cmpOr(headline, "Ошибка"),
		Template: listTemplate(""),
		Items: []Item{
			{Type: "space", Layout: "0,0,8,2", Text: err.Error()},
			{Title: "Назад", Icon: "arrow-back", Action: "back"},
		},
	}
}

// coversContent renders a grid of posters linking to the video pages.
func coversContent(base, headline string, covers []*hdrezka.CoverItem) *Content {
	items := make([]Item, 0, len(covers))
	for _, cover := range covers {
		if cover.URL == "" {
			continue
		}
		items = append(items, Item{
			Title:       cover.Title,
			TitleFooter: cover.Description,
			Tag:         strings.TrimSpace(cover.Info),
			Image:       cover.Cover,
			Action:      contentAction(base, pathVideo, navParams{VideoURL: cover.URL}.values()),
		})
	}
	return &Content{
		Type:     "list",
		Headline: headline,
		Template: posterTemplate(),
		Items:    items,
	}
}

func (s *server) handleStart(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, buildStart(s.baseURL(r)))
}

func (s *server) handleMenu(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, buildMenu(s.baseURL(r)))
}

// handleGenre renders the section page: the catalog entry points for a genre.
func (s *server) handleGenre(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	genre, err := genreByName(r.URL.Query().Get("genre"))
	if err != nil {
		s.writeContent(w, "Раздел", nil, err)
		return
	}
	genreValue := url.Values{"genre": {string(genre)}}

	items := []Item{{
		Title:  "Новинки",
		Icon:   "fiber-new",
		Action: contentAction(base, pathNewest, genreValue),
	}}
	for _, f := range filterTitles {
		items = append(items, Item{
			Title:  f.Title,
			Icon:   f.Icon,
			Action: contentAction(base, pathCovers, url.Values{"type": {"category"}, "genre": {string(genre)}, "filter": {string(f.Filter)}}),
		})
	}
	items = append(items,
		Item{Title: "Жанры", Icon: "list", Action: contentAction(base, pathCategories, genreValue)},
		Item{Title: "По годам", Icon: "date-range", Action: contentAction(base, pathYears, genreValue)},
		Item{Title: "По странам", Icon: "public", Action: contentAction(base, pathCountries, genreValue)},
	)

	writeJSON(w, &Content{
		Type:     "list",
		Headline: genreTitle(genre),
		Template: listTemplate("menu"),
		Items:    items,
	})
}

func (s *server) handleCovers(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	query := r.URL.Query()
	opts, err := parseCoverOption(query)
	if err != nil {
		s.writeContent(w, "Каталог", nil, err)
		return
	}
	maxItems := s.cfg.maxItems
	if n, err := parseIntParam(query.Get("n")); err == nil && n > 0 {
		maxItems = n
	}

	key := fmt.Sprintf("covers:%d:%+v", maxItems, opts)
	covers, err := cached(s.cache, key, func() ([]*hdrezka.CoverItem, error) {
		return s.rezka.GetCovers(opts, maxItems)
	})
	headline := coversHeadline(opts, query.Get("headline"))
	if err != nil {
		s.writeContent(w, headline, nil, err)
		return
	}
	writeJSON(w, coversContent(base, headline, covers))
}

// coversHeadline builds the page title for a catalog request.
func coversHeadline(opts hdrezka.CoverOption, override string) string {
	if override != "" {
		return override
	}
	parts := []string{genreTitle(opts.Genre)}
	switch {
	case opts.Category != "":
		parts = append(parts, opts.Category)
	case opts.Country != "":
		parts = append(parts, opts.Country)
	case opts.Year != "":
		parts = append(parts, opts.Year)
	case opts.Type == hdrezka.CoverNew:
		parts = append(parts, "новинки")
	}
	return strings.Join(parts, " · ")
}

func (s *server) handleNewest(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	genre, err := genreByName(r.URL.Query().Get("genre"))
	if err != nil {
		s.writeContent(w, "Новинки", nil, err)
		return
	}
	headline := genreTitle(genre) + " · новинки"
	covers, err := cached(s.cache, "newest:"+string(genre), func() ([]*hdrezka.CoverItem, error) {
		return s.rezka.GetCoversNewest(genre)
	})
	if err != nil {
		s.writeContent(w, headline, nil, err)
		return
	}
	writeJSON(w, coversContent(base, headline, covers))
}

func (s *server) handleCategories(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	genre, err := genreByName(r.URL.Query().Get("genre"))
	if err != nil {
		s.writeContent(w, "Жанры", nil, err)
		return
	}
	categories := s.rezka.Categories[genre]
	items := make([]Item, 0, len(categories))
	for _, name := range sortedKeys(categories) {
		items = append(items, Item{
			Title: name,
			Action: contentAction(base, pathCovers, url.Values{
				"type":     {"category"},
				"genre":    {string(genre)},
				"category": {name},
				"headline": {genreTitle(genre) + " · " + name},
			}),
		})
	}
	writeJSON(w, &Content{
		Type:     "list",
		Headline: genreTitle(genre) + " · жанры",
		Template: listTemplate("label"),
		Items:    items,
	})
}

func (s *server) handleYears(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	genre, err := genreByName(r.URL.Query().Get("genre"))
	if err != nil {
		s.writeContent(w, "По годам", nil, err)
		return
	}
	items := make([]Item, 0, len(s.rezka.Years))
	for _, year := range s.rezka.Years {
		items = append(items, Item{
			Title: year,
			Action: contentAction(base, pathCovers, url.Values{
				"type":  {"year"},
				"genre": {string(genre)},
				"year":  {year},
			}),
		})
	}
	writeJSON(w, &Content{
		Type:     "list",
		Headline: genreTitle(genre) + " · по годам",
		Template: listTemplate("date-range"),
		Items:    items,
	})
}

func (s *server) handleCountries(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	genre, err := genreByName(r.URL.Query().Get("genre"))
	if err != nil {
		s.writeContent(w, "По странам", nil, err)
		return
	}
	items := make([]Item, 0, len(countries))
	for _, country := range countries {
		items = append(items, Item{
			Title: country,
			Action: contentAction(base, pathCovers, url.Values{
				"type":    {"country"},
				"genre":   {string(genre)},
				"country": {country},
			}),
		})
	}
	writeJSON(w, &Content{
		Type:     "list",
		Headline: genreTitle(genre) + " · по странам",
		Template: listTemplate("public"),
		Items:    items,
	})
}

// getVideo fetches (and caches) a video page.
func (s *server) getVideo(videoURL string) (*hdrezka.Video, error) {
	return cached(s.cache, "video:"+videoURL, func() (*hdrezka.Video, error) {
		return s.rezka.GetVideo(videoURL)
	})
}

// translation resolves a translation of a video by its id, falling back to the
// default translation when no id was given.
func translationByID(video *hdrezka.Video, id string) (*hdrezka.Translation, error) {
	var fallback *hdrezka.Translation
	for _, tr := range video.Translation {
		if tr.ID == id && id != "" {
			return tr, nil
		}
		if fallback == nil || tr.IsDefault {
			fallback = tr
		}
	}
	if id != "" {
		return nil, fmt.Errorf("translation %q not found", id)
	}
	if fallback == nil {
		return nil, fmt.Errorf("video has no translations")
	}
	return fallback, nil
}

// handleVideo renders the video card: description, poster and the list of
// available translations.
func (s *server) handleVideo(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	nav, err := parseNav(r.URL.Query())
	if err != nil {
		s.writeContent(w, "Видео", nil, err)
		return
	}
	video, err := s.getVideo(nav.VideoURL)
	if err != nil {
		s.writeContent(w, "Видео", nil, err)
		return
	}

	items := make([]Item, 0, len(video.Translation))
	for _, tr := range video.Translation {
		nav := navParams{VideoURL: nav.VideoURL, Translation: tr.ID}
		items = append(items, Item{
			// No icon: MSX centres it in the row, on top of the title.
			Title:  joinParts(cmpOr(tr.Name, "По умолчанию"), translationLabel(tr)),
			Action: contentAction(base, pathSeasons, nav.values()),
		})
	}

	writeJSON(w, &Content{
		Type:     "list",
		Headline: videoHeadline(video),
		Header:   videoCard(base, nav, video),
		Template: listTemplate(""),
		Items:    items,
	})
}

// videoCard builds the header page of the video card: the poster on the left
// and the synopsis with the key facts next to it.
//
// It has to be a header page rather than leading items, because the layout of
// a templated item is ignored — items always take the size the template
// dictates, which is why a synopsis put among them stayed one row tall.
// Items of a content page keep their own layout on the 12×6 grid.
// MSX rejects a page whose items are all unselectable ("Page[0] has no
// selectable items") and renders none of it, so the card always carries one
// focusable element: the poster, which doubles as a play button for the
// default translation.
func videoCard(base string, nav navParams, video *hdrezka.Video) *Page {
	// Rows of the card on the 12×6 grid: the facts line, two rows of synopsis
	// and the credits line. Each block is a separate item because MSX joins
	// the entries of a text array without any separator.
	const (
		posterWidth  = 2
		posterHeight = 3
		textX        = posterWidth
		textWidth    = 12 - textX
		rowFacts     = 0
		rowSynopsis  = 1
		rowCredits   = 3
	)

	// The action must never be empty, or the card loses its only selectable
	// item. Without a usable translation the stream page takes over and
	// reports why nothing can be played.
	play := contentAction(base, pathStream, nav.values())
	if tr, err := translationByID(video, ""); err == nil {
		play = contentAction(base, pathSeasons, navParams{
			VideoURL:    nav.VideoURL,
			Translation: tr.ID,
		}.values())
	}

	items := make([]Item, 0, 4)
	if video.Cover != "" {
		items = append(items, Item{
			Type:   "default",
			Layout: fmt.Sprintf("0,0,%d,%d", posterWidth, posterHeight),
			Image:  video.Cover,
			// "fit" scales the poster to sit inside the area whatever its
			// aspect ratio is; the "height-*" fillers scale to the full height
			// and crop a portrait poster on both sides.
			ImageFiller: "fit",
			Icon:        "play-arrow",
			Action:      play,
		})
	} else {
		items = append(items, Item{
			Type:   "button",
			Layout: fmt.Sprintf("0,0,%d,1", posterWidth),
			Icon:   "play-arrow",
			Label:  "Смотреть",
			Action: play,
		})
	}

	// The title is not repeated here — it is already the page headline.
	textBlock := func(row, height int, text string) {
		if text == "" {
			return
		}
		items = append(items, Item{
			Type:   "space",
			Layout: fmt.Sprintf("%d,%d,%d,%d", textX, row, textWidth, height),
			Text:   text,
		})
	}
	textBlock(rowFacts, 1, videoFacts(video))
	textBlock(rowSynopsis, rowCredits-rowSynopsis, truncateWords(video.Description, descriptionLimit))
	textBlock(rowCredits, 1, videoCredits(video))

	return &Page{Items: items}
}

// videoCredits renders the director and the leading cast into a single line.
func videoCredits(video *hdrezka.Video) string {
	names := func(people []hdrezka.Person, limit int) string {
		list := make([]string, 0, limit)
		for _, person := range people {
			if len(list) == limit {
				break
			}
			if person.Name != "" {
				list = append(list, person.Name)
			}
		}
		return strings.Join(list, ", ")
	}

	var parts []string
	if director := names(video.Director, 2); director != "" {
		parts = append(parts, "Режиссёр: "+director)
	}
	if cast := names(video.Cast, 4); cast != "" {
		parts = append(parts, "В ролях: "+cast)
	}
	return joinParts(parts...)
}

// videoHeadline renders the title line of the video card.
func videoHeadline(video *hdrezka.Video) string {
	title := video.Title
	if video.TitleOriginal != "" {
		title += " / " + video.TitleOriginal
	}
	if video.Year != "" {
		title += " (" + video.Year + ")"
	}
	return title
}

// descriptionLimit caps the synopsis so it fits the two rows the card gives
// it; MSX would otherwise cut it off mid-word.
const descriptionLimit = 350

// videoFacts renders ratings, runtime, genres and countries into the single
// line shown above the synopsis.
func videoFacts(video *hdrezka.Video) string {
	var facts []string
	if video.Rating.Score > 0 {
		facts = append(facts, fmt.Sprintf("HDrezka %.1f", video.Rating.Score))
	}
	if video.RatingIMDB.Score > 0 {
		facts = append(facts, fmt.Sprintf("IMDb %.1f", video.RatingIMDB.Score))
	}
	if video.RatingKinopoisk.Score > 0 {
		facts = append(facts, fmt.Sprintf("КП %.1f", video.RatingKinopoisk.Score))
	}
	if video.Duration != "" {
		facts = append(facts, video.Duration)
	}
	if len(video.Categories) > 0 {
		facts = append(facts, strings.Join(video.Categories, ", "))
	}
	if len(video.Country) > 0 {
		facts = append(facts, strings.Join(video.Country, ", "))
	}

	return joinParts(facts...)
}

// truncateWords shortens text to at most limit characters, cutting at a word
// boundary and appending an ellipsis.
func truncateWords(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	cut := string(runes[:limit])
	if space := strings.LastIndexAny(cut, " \n\t"); space > limit/2 {
		cut = cut[:space]
	}
	return strings.TrimRight(cut, " ,.;:—-") + "…"
}

// translationLabel describes the quirks of a translation (ads, cam rip, …).
func translationLabel(tr *hdrezka.Translation) string {
	var flags []string
	if tr.IsDefault {
		flags = append(flags, "по умолчанию")
	}
	if tr.IsAds {
		flags = append(flags, "с рекламой")
	}
	if tr.IsCamRip {
		flags = append(flags, "экранка")
	}
	if tr.IsDirector {
		flags = append(flags, "режиссёрская")
	}
	if tr.IsPremium {
		flags = append(flags, "премиум")
	}
	return strings.Join(flags, ", ")
}

// handleSeasons lists the seasons of a series. Movies have no episode list, so
// the request falls through to the quality page.
func (s *server) handleSeasons(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	nav, err := parseNav(r.URL.Query())
	if err != nil {
		s.writeContent(w, "Сезоны", nil, err)
		return
	}
	video, translation, err := s.resolveTranslation(nav)
	if err != nil {
		s.writeContent(w, "Сезоны", nil, err)
		return
	}

	episodes, err := s.getEpisodes(nav, translation)
	if err != nil || len(episodes) == 0 {
		// A movie: skip straight to the quality selection.
		s.renderStream(w, base, nav, video, translation)
		return
	}

	seasons := episodes.ListSeasons()
	items := make([]Item, 0, len(seasons))
	for _, season := range seasons {
		nav := navParams{VideoURL: nav.VideoURL, Translation: nav.Translation, Season: season}
		items = append(items, Item{
			Title: joinParts(
				fmt.Sprintf("Сезон %d", season),
				fmt.Sprintf("%d серий", len(episodes.ListEpisodes(season))),
			),
			Action: contentAction(base, pathEpisodes, nav.values()),
		})
	}
	writeJSON(w, &Content{
		Type:     "list",
		Headline: videoHeadline(video),
		Template: listTemplate(""),
		Items:    items,
	})
}

// handleEpisodes lists the episodes of the selected season.
func (s *server) handleEpisodes(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	nav, err := parseNav(r.URL.Query())
	if err != nil {
		s.writeContent(w, "Серии", nil, err)
		return
	}
	if nav.Season <= 0 {
		s.writeContent(w, "Серии", nil, fmt.Errorf("season is required"))
		return
	}
	video, translation, err := s.resolveTranslation(nav)
	if err != nil {
		s.writeContent(w, "Серии", nil, err)
		return
	}
	episodes, err := s.getEpisodes(nav, translation)
	if err != nil {
		s.writeContent(w, "Серии", nil, err)
		return
	}

	numbers := episodes.ListEpisodes(nav.Season)
	items := make([]Item, 0, len(numbers))
	for _, episode := range numbers {
		nav := navParams{VideoURL: nav.VideoURL, Translation: nav.Translation, Season: nav.Season, Episode: episode}
		items = append(items, Item{
			Title:  fmt.Sprintf("Серия %d", episode),
			Action: contentAction(base, pathStream, nav.values()),
		})
	}
	writeJSON(w, &Content{
		Type:     "list",
		Headline: fmt.Sprintf("%s · сезон %d", videoHeadline(video), nav.Season),
		Template: listTemplate(""),
		Items:    items,
	})
}

// handleStream renders the quality selection, the last step before playback.
func (s *server) handleStream(w http.ResponseWriter, r *http.Request) {
	base := s.baseURL(r)
	nav, err := parseNav(r.URL.Query())
	if err != nil {
		s.writeContent(w, "Качество", nil, err)
		return
	}
	video, translation, err := s.resolveTranslation(nav)
	if err != nil {
		s.writeContent(w, "Качество", nil, err)
		return
	}
	s.renderStream(w, base, nav, video, translation)
}

// renderStream fetches the CDN stream and renders one playable item per
// quality. The CDN links are only available through this AJAX call, so they
// are resolved here — one network round trip per click.
func (s *server) renderStream(w http.ResponseWriter, base string, nav navParams, video *hdrezka.Video, translation *hdrezka.Translation) {
	headline := videoHeadline(video)
	playerLabel := headline
	if nav.Season > 0 {
		headline = fmt.Sprintf("%s · s%02de%02d", headline, nav.Season, nav.Episode)
		playerLabel = headline
	}

	var stream *hdrezka.Stream
	var err error
	if nav.Season > 0 {
		stream, err = translation.GetStream(nav.Season, nav.Episode)
	} else {
		stream, err = translation.GetStream()
	}
	if err != nil {
		s.writeContent(w, headline, nil, err)
		return
	}
	if len(stream.Formats) == 0 {
		s.writeContent(w, headline, nil, fmt.Errorf("нет доступных потоков"))
		return
	}

	qualities := sortQualities(stream.Formats)
	sizes := s.formatSizes(stream, qualities)

	items := make([]Item, 0, len(qualities))
	for _, quality := range qualities {
		mediaURL := streamURL(stream.Formats[quality], s.cfg.useMP4)
		if mediaURL == "" {
			continue
		}
		action, props := videoAction(base, mediaURL, stream.Subtitles)
		items = append(items, Item{
			Title:       joinParts(qualityTitle(quality), qualityLabel(quality), sizes[quality]),
			PlayerLabel: playerLabel,
			Action:      action,
			Properties:  props,
		})
	}

	writeJSON(w, &Content{
		Type:     "list",
		Headline: headline,
		Template: listTemplate(""),
		Items:    items,
	})
}

// formatSizes resolves the file size of every quality when --sizes is enabled.
// Each lookup is an extra HEAD request, so they run concurrently.
func (s *server) formatSizes(stream *hdrezka.Stream, qualities []string) map[string]string {
	sizes := make(map[string]string, len(qualities))
	if !s.cfg.showSizes {
		return sizes
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, quality := range qualities {
		wg.Add(1)
		go func(quality string) {
			defer wg.Done()
			size, err := stream.Formats[quality].Size(s.rezka.Client)
			if err != nil {
				return
			}
			mu.Lock()
			sizes[quality] = hdrezka.FormatBytes(size)
			mu.Unlock()
		}(quality)
	}
	wg.Wait()
	return sizes
}

// resolveTranslation loads the video page and picks the requested translation.
func (s *server) resolveTranslation(nav navParams) (*hdrezka.Video, *hdrezka.Translation, error) {
	video, err := s.getVideo(nav.VideoURL)
	if err != nil {
		return nil, nil, err
	}
	translation, err := translationByID(video, nav.Translation)
	if err != nil {
		return nil, nil, err
	}
	return video, translation, nil
}

// getEpisodes fetches (and caches) the episode list of a translation.
func (s *server) getEpisodes(nav navParams, translation *hdrezka.Translation) (hdrezka.Episodes, error) {
	return cached(s.cache, "episodes:"+nav.VideoURL+":"+translation.ID, translation.GetEpisodes)
}

// streamURL picks the media URL to play for a quality.
//
// HLS is the default: the progressive MP4 files the HDrezka CDN serves stall
// in HTML5 players (the CDN answers HEAD with 502 and never delivers the
// initial media segment for an open "Range: bytes=0-" request, which is
// exactly what a <video> element sends), so playback fails with "source is not
// supported". The HLS manifests of the same stream — what the site's own
// player uses — are served correctly. MP4 stays available behind --mp4 for
// devices that prefer a direct file.
func streamURL(format hdrezka.VideoFormat, preferMP4 bool) string {
	primary, fallback := format.HLS, format.MP4
	if preferMP4 {
		primary, fallback = format.MP4, format.HLS
	}
	if primary != "" {
		return primary
	}
	return fallback
}

// isHLS reports whether a media URL is an HLS manifest. HDrezka appends the
// manifest to the file name (…/video.mp4:hls:manifest.m3u8), so the check
// looks at the suffix rather than the extension.
func isHLS(mediaURL string) bool {
	return strings.HasSuffix(mediaURL, ".m3u8")
}

// reHTMLTag matches the markup HDrezka wraps premium quality labels in
// (e.g. `<span class="pjs-prem-quality">1080p Ultra<img …></span>`).
var reHTMLTag = regexp.MustCompile(`<[^>]*>`)

// qualityTitle turns a raw quality label into plain text fit for the screen.
func qualityTitle(quality string) string {
	return strings.TrimSpace(html.UnescapeString(reHTMLTag.ReplaceAllString(quality, "")))
}

// qualityLabel marks the qualities only a premium account can play.
func qualityLabel(quality string) string {
	if strings.Contains(quality, "prem-quality") {
		return "премиум"
	}
	return ""
}

// sortQualities orders quality labels from the highest resolution down so the
// best stream is the first item on the page.
func sortQualities(formats map[string]hdrezka.VideoFormat) []string {
	qualities := sortedKeys(formats)
	sort.SliceStable(qualities, func(i, j int) bool {
		ri, rj := qualityResolution(qualityTitle(qualities[i])), qualityResolution(qualityTitle(qualities[j]))
		if ri != rj {
			return ri > rj
		}
		return qualities[i] > qualities[j]
	})
	return qualities
}

// qualityResolution extracts the vertical resolution of a quality label
// ("1080p Ultra" -> 1080). Labels without a leading number sort last.
func qualityResolution(quality string) int {
	digits := 0
	for digits < len(quality) && quality[digits] >= '0' && quality[digits] <= '9' {
		digits++
	}
	if digits == 0 {
		return -1
	}
	value, err := strconv.Atoi(quality[:digits])
	if err != nil {
		return -1
	}
	return value
}
