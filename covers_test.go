package hdrezka

import (
	"testing"
)

func TestHDRezkaGetCoversURL(t *testing.T) {
	r, err := New("https://hdrezka.ag")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		args    CoverOption
		want    string
		wantErr bool
	}{
		{"CoverAll",
			CoverOption{
				Genre:  Series,
				Filter: FilterWatching,
				Type:   CoverAll,
			}, "https://hdrezka.ag/?filter=watching&genre=2", false},
		{"CoverBest",
			CoverOption{
				Genre:    Series,
				Category: "Ужасы",
				Type:     CoverBest,
				Year:     "2016",
			}, "https://hdrezka.ag/series/best/horror/2016/", false},
		{"CoverByCategory",
			CoverOption{
				Genre:    Films,
				Category: "Драмы",
				Filter:   FilterPopular,
				Type:     CoverByCategory,
			}, "https://hdrezka.ag/films/drama/?filter=popular", false},
		{"CoverByCountry",
			CoverOption{
				Genre:   Films,
				Country: "США",
				Filter:  FilterPopular,
				Type:    CoverByCountry,
			}, "https://hdrezka.ag/country/%D0%A1%D0%A8%D0%90/?filter=popular&genre=1", false},
		{"CoverByYear",
			CoverOption{
				Genre:  Show,
				Filter: FilterPopular,
				Type:   CoverByYear,
				Year:   "1986",
			}, "https://hdrezka.ag/year/1986/?filter=popular&genre=4", false},
		{"CoverNew",
			CoverOption{
				Genre:    Anime,
				Category: "Драмы",
				Filter:   FilterLast,
				Type:     CoverNew,
			}, "https://hdrezka.ag/new/?filter=last&genre=82", false},
		{"NoType",
			CoverOption{
				Genre:  Series,
				Filter: FilterLast,
			}, "https://hdrezka.ag/?filter=last&genre=2", false},
		{"CategoryUnknow",
			CoverOption{
				Genre:    Anime,
				Category: "Нет",
				Type:     CoverByCategory,
			}, "", true},
		{"NoOptions", CoverOption{}, "https://hdrezka.ag/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetCoversURL(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("HDRezka.GetCoversURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HDRezka.GetCoversURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
