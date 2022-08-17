package hdrezka

type (
	Cover  int8
	Filter string
	Genre  string
)

const (
	All      Genre = ""
	Anime    Genre = "animation"
	Cartoons Genre = "cartoons"
	Films    Genre = "films"
	Series   Genre = "series"
	Show     Genre = "show"
)

const (
	FilterLast     Filter = "last"
	FilterPopular  Filter = "popular"
	FilterWatching Filter = "watching"
)

const (
	CoverAll Cover = iota
	CoverByCategory
	CoverByCountry
	CoverByYear
	CoverBest
	CoverNew
)

var categoriesShow = map[string]string{
	"Боевые искусства": "/show/fighting/",
	"Детские":          "/show/kids/",
	"Конкурсы":         "/show/contests/",
	"Кулинария":        "/show/cooking/",
	"Мода":             "/show/fashion/",
	"Музыкальные":      "/show/musical/",
	"О здоровье":       "/show/health/",
	"Охота и рыбалка":  "/show/hunting/",
	"Познавательные":   "/show/cognitive/",
	"Путешествия":      "/show/travel/",
	"Реалити-шоу":      "/show/reality-shows/",
	"Cпортивные":       "/show/sport/",
	"Семейные":         "/show/family/",
	"Юмористические":   "/show/humor/",
}
