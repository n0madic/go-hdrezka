package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/alexflint/go-arg"
	"github.com/n0madic/go-hdrezka"
)

type DefaultCmd struct{}

type BestCmd struct {
	Category string `arg:"-c,--category" help:"category for video"`
	Year     string `arg:"-y,--year" help:"year for video"`
}

type CategoryCmd struct {
	Category string `arg:"positional,required" help:"category for video"`
}

type CountryCmd struct {
	Country string `arg:"positional,required" help:"country for video"`
}

type YearCmd struct {
	Year string `arg:"positional,required" help:"year for video"`
}

type SearchCmd struct {
	Query string `arg:"positional,required"`
	Quick bool   `arg:"-q"`
}

var mirrors = []string{"https://hdrezka.ag", "https://rezka.ag"}

var args struct {
	All            *DefaultCmd    `arg:"subcommand:all" help:"Show all releases"`
	Best           *BestCmd       `arg:"subcommand:best" help:"Show best releases"`
	Category       *CategoryCmd   `arg:"subcommand:category" help:"Show releases by category"`
	Country        *CountryCmd    `arg:"subcommand:country" help:"Show releases by country"`
	New            *DefaultCmd    `arg:"subcommand:new" help:"Show new releases"`
	Newest         *DefaultCmd    `arg:"subcommand:newest" help:"Show newest releases"`
	Year           *YearCmd       `arg:"subcommand:year" help:"Show releases by year"`
	Search         *SearchCmd     `arg:"subcommand:search" help:"Search releases"`
	Extended       bool           `arg:"-e,--extended" help:"Show extended info for release"`
	Filter         hdrezka.Filter `arg:"-f,--filter" help:"Set filter for release (last|popular|watching)"`
	Genre          hdrezka.Genre  `arg:"-g,--genre" help:"Set genre for release (animation|cartoons|films|series|show)"`
	ListCategories bool           `arg:"-l,--list-categories" help:"List categories of videos"`
	Mirrors        []string       `arg:"-m,--mirrors" help:"mirrors for hdrezka site"`
	Number         int            `arg:"-n,--number" default:"36" help:"number of releases to show"`
}

func main() {
	arg.MustParse(&args)
	if len(args.Mirrors) == 0 {
		args.Mirrors = mirrors
	}

	r, err := hdrezka.New(args.Mirrors...)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	if args.ListCategories {
		for genre, category := range r.Categories {
			fmt.Println(genre + ":")
			var categories []string
			for c := range category {
				categories = append(categories, c)
			}
			sort.Strings(categories)
			for _, c := range categories {
				fmt.Println("  ", c)
			}
		}
		return
	}

	var opts = hdrezka.CoverOption{
		Genre:  args.Genre,
		Filter: args.Filter,
	}
	if args.All != nil {
		opts.Type = hdrezka.CoverAll
	} else if args.Best != nil {
		opts.Type = hdrezka.CoverBest
		opts.Category = args.Best.Category
		opts.Year = args.Best.Year
	} else if args.Category != nil {
		opts.Type = hdrezka.CoverByCategory
		opts.Category = args.Category.Category
	} else if args.Country != nil {
		opts.Type = hdrezka.CoverByCountry
		opts.Country = args.Country.Country
	} else if args.New != nil {
		opts.Type = hdrezka.CoverNew
	} else if args.Year != nil {
		opts.Type = hdrezka.CoverByYear
		opts.Year = args.Year.Year
	}

	var items []*hdrezka.CoverItem
	if args.Newest != nil {
		items, err = r.GetCoversNewest(args.Genre)
	} else if args.Search != nil {
		if args.Search.Quick {
			items, err = r.QuickSearch(args.Search.Query)
		} else {
			items, err = r.Search(args.Search.Query, args.Number)
		}
	} else {
		items, err = r.GetCovers(opts, args.Number)
	}
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("List of releases (found %d):\n", len(items))
	for _, item := range items {
		fmt.Println("--------------------------------------------------")
		if args.Extended {
			video, err := r.GetVideo(item.URL)
			if err == nil {
				fmt.Print(video)
			} else {
				fmt.Print(item)
			}
		} else {
			fmt.Print(item)
		}
	}
	fmt.Println()
}
