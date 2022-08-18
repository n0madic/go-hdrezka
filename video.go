package hdrezka

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Person is a struct for person info
type Person struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Rating is a struct for rating
type Rating struct {
	Score float64 `json:"score,omitempty"`
	Votes int     `json:"votes,omitempty"`
}

// Translation is a struct for translator info
type Translation struct {
	r         *HDRezka
	videoID   string
	Name      string `json:"name"`
	ID        string `json:"id"`
	IsAds     bool   `json:"is_ads"`
	IsCamRip  bool   `json:"is_camrip"`
	IsDefault bool   `json:"is_default"`
}

// Video is a struct for video info
type Video struct {
	Age             string         `json:"age,omitempty"`
	Cast            []Person       `json:"cast,omitempty"`
	Categories      []string       `json:"categories,omitempty"`
	Country         []string       `json:"country,omitempty"`
	Cover           string         `json:"cover,omitempty"`
	DefaultStream   *Stream        `json:"default_stream,omitempty"`
	Description     string         `json:"description,omitempty"`
	Director        []Person       `json:"director,omitempty"`
	Duration        string         `json:"duration,omitempty"`
	ID              string         `json:"id"`
	Rating          Rating         `json:"rating,omitempty"`
	RatingIMDB      Rating         `json:"rating_imdb,omitempty"`
	RatingKinopoisk Rating         `json:"rating_kinopoisk,omitempty"`
	ReleaseDate     string         `json:"release_date,omitempty"`
	Quality         string         `json:"quality,omitempty"`
	Tagline         string         `json:"tagline,omitempty"`
	Title           string         `json:"title"`
	TitleOriginal   string         `json:"title_original,omitempty"`
	Translation     []*Translation `json:"translation,omitempty"`
	Type            Genre          `json:"type"`
	Year            string         `json:"year,omitempty"`
}

// GetVideo returns video info from URL.
func (r *HDRezka) GetVideo(videoURL string) (*Video, error) {
	doc, err := getDoc(videoURL)
	if err != nil {
		return nil, err
	}

	video := &Video{}

	video.Age = doc.Find("tr:contains('Возраст:')").Find("td").First().Next().Text()
	doc.Find("span.person-name-item[itemprop=actor]").Each(func(i int, s *goquery.Selection) {
		video.Cast = append(video.Cast, Person{
			Name: s.Find("span[itemprop=name]").Text(),
			URL:  s.Find("a[itemprop=url]").AttrOr("href", ""),
		})
	})
	doc.Find("span[itemprop=genre]").Each(func(i int, s *goquery.Selection) {
		video.Categories = append(video.Categories, s.Text())
	})
	doc.Find("tr:contains('Страна:') > td > a").Each(func(i int, s *goquery.Selection) {
		video.Country = append(video.Country, s.Text())
	})
	video.Cover = doc.Find("a[data-imagelightbox=cover]").AttrOr("href", "")
	video.Description = strings.TrimSpace(doc.Find("div.b-post__description_text").Text())
	video.Duration = doc.Find("td[itemprop=duration]").Text()
	doc.Find("span.person-name-item[itemprop=director]").Each(func(i int, s *goquery.Selection) {
		video.Director = append(video.Director, Person{
			Name: s.Find("span[itemprop=name]").Text(),
			URL:  s.Find("a[itemprop=url]").AttrOr("href", ""),
		})
	})
	video.ID = doc.Find(".b-userset__fav_holder").AttrOr("data-post_id", "")
	video.Quality = doc.Find("tr:contains('В качестве:')").Find("td").First().Next().Text()
	video.Rating = Rating{
		Score: parseFloat(doc.Find("span[itemprop=rating] > span.num").Text()),
		Votes: parseInt(doc.Find(".votes > span").Text()),
	}
	video.RatingIMDB = Rating{
		Score: parseFloat(doc.Find("span.imdb > span").Text()),
		Votes: parseInt(doc.Find("span.imdb > i").Text()),
	}
	video.RatingKinopoisk = Rating{
		Score: parseFloat(doc.Find("span.kp > span").Text()),
		Votes: parseInt(doc.Find("span.kp > i").Text()),
	}
	video.ReleaseDate = doc.Find("tr:contains('Дата выхода:')").Find("td").First().Next().Text()
	video.Tagline = strings.Trim(doc.Find("tr:contains('Слоган:')").Find("td").First().Next().Text(), "«»")
	video.Title = doc.Find("h1[itemprop=name]").Text()
	video.TitleOriginal = doc.Find(".b-post__origtitle").Text()

	// Get default stream
	var defaultTranslator string
	html, _ := doc.Html()
	initCDNMatch := reTranslate.FindStringSubmatch(html)
	if len(initCDNMatch) > 0 {
		defaultTranslator = string(initCDNMatch[2])
	}
	var jsn map[string]interface{}
	err = json.NewDecoder(strings.NewReader(initCDNMatch[3])).Decode(&jsn)
	if err != nil {
		return nil, err
	}
	thumbnails := r.URL.JoinPath(jsn["thumbnails"].(string))
	video.DefaultStream = &Stream{
		URL:         jsn["streams"].(string),
		Subtitle:    jsn["subtitle"],
		SubtitleDef: jsn["subtitle_def"],
		Thumbnails:  thumbnails.String(),
	}
	video.DefaultStream.URL, err = decodeURL(video.DefaultStream.URL)
	if err != nil {
		return nil, err
	} else {
		video.DefaultStream.Formats = parseStreamFormats(video.DefaultStream.URL)
	}

	// Get translators
	doc.Find(".b-translator__item").Each(func(i int, s *goquery.Selection) {
		translation := &Translation{
			r:        r,
			videoID:  video.ID,
			Name:     strings.TrimSpace(s.Text()),
			ID:       s.AttrOr("data-translator_id", ""),
			IsAds:    s.AttrOr("data-ads", "") == "1",
			IsCamRip: s.AttrOr("data-camrip", "") == "1",
		}
		if translation.ID == defaultTranslator {
			translation.IsDefault = true
		}
		video.Translation = append(video.Translation, translation)
	})
	if len(video.Translation) == 0 {
		name := doc.Find("tr:contains('В переводе:')").Find("td").First().Next().Text()
		video.Translation = append(video.Translation, &Translation{
			r:         r,
			videoID:   video.ID,
			Name:      strings.TrimSpace(name),
			ID:        defaultTranslator,
			IsDefault: true,
		})
	}

	video.Type = Genre(strings.Split(videoURL, "/")[3])
	video.Year = regexp.MustCompile(`\d{4}`).FindString(video.ReleaseDate)

	return video, nil
}

func (video *Video) JSON() string {
	js, _ := json.MarshalIndent(video, "", "    ")
	return string(js)
}

func (video *Video) String() string {
	output := fmt.Sprintf("Type:\t\t%s\n", video.Type)

	title := video.Title
	if video.TitleOriginal != "" {
		title += " / " + video.TitleOriginal
	}
	if video.Year != "" {
		title += " (" + video.Year + ")"
	}
	output += fmt.Sprintf("Title:\t\t%s\n", title)

	if video.Tagline != "" {
		output += fmt.Sprintf("Tagline:\t%s\n", video.Tagline)
	}

	rating := fmt.Sprintf("%0.1f (%d)", video.Rating.Score, video.Rating.Votes)
	if video.RatingIMDB.Score > 0 {
		rating += fmt.Sprintf(" / IMDB: %0.1f (%d)", video.RatingIMDB.Score, video.RatingIMDB.Votes)
	}
	if video.RatingKinopoisk.Score > 0 {
		rating += fmt.Sprintf(" / Kinopoisk: %0.1f (%d)", video.RatingKinopoisk.Score, video.RatingKinopoisk.Votes)
	}
	output += fmt.Sprintf("Rating:\t\t%s\n", rating)

	if len(video.Country) > 0 {
		output += fmt.Sprintf("Country:\t%s\n", strings.Join(video.Country, ", "))
	}

	if video.ReleaseDate != "" {
		output += fmt.Sprintf("Release date:\t%s\n", video.ReleaseDate)
	}

	if video.Duration != "" {
		output += fmt.Sprintf("Duration:\t%s\n", video.Duration)
	}

	if video.Age != "" {
		output += fmt.Sprintf("Age:\t\t%s\n", video.Age)
	}

	if len(video.Categories) > 0 {
		output += fmt.Sprintf("Categories:\t%s\n", strings.Join(video.Categories, ", "))
	}

	var directors []string
	for _, director := range video.Director {
		directors = append(directors, director.Name)
	}
	if len(directors) > 0 {
		output += fmt.Sprintf("Director:\t%s\n", strings.Join(directors, ", "))
	}

	var actors []string
	for _, actor := range video.Cast {
		actors = append(actors, actor.Name)
	}
	if len(actors) > 0 {
		output += fmt.Sprintf("Cast:\t\t%s\n", strings.Join(actors, ", "))
	}

	var translations []string
	for _, translation := range video.Translation {
		if translation.Name != "" {
			if translation.IsDefault {
				translation.Name += " [default]"
			}
			translations = append(translations, translation.Name)
		}
	}
	if len(translations) > 0 {
		output += fmt.Sprintf("Translation:\t%s\n", strings.Join(translations, ", "))
	}

	if video.Description != "" {
		output += fmt.Sprintf("Description:\t%s\n", video.Description)
	}

	return output
}
