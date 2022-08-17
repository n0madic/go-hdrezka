# hdrezka-rlz

Utility for receiving and searching for releases (covers) from the HDrezka site

## Install

```
go install github.com/n0madic/go-hdrezka/cmd/hdrezka-rlz@latest
```

## Help

```
Usage: hdrezka-rlz [--extended] [--filter FILTER] [--genre GENRE] [--list-categories] [--mirrors MIRRORS] [--number NUMBER] <command> [<args>]

Options:
  --extended, -e         Show extended info for release
  --filter FILTER, -f FILTER
                         Set filter for release (last|popular|watching)
  --genre GENRE, -g GENRE
                         Set genre for release (animation|cartoons|films|series|show)
  --list-categories, -l
                         List categories of videos
  --mirrors MIRRORS, -m MIRRORS
                         mirrors for hdrezka site
  --number NUMBER, -n NUMBER
                         number of releases to show [default: 36]
  --help, -h             display this help and exit

Commands:
  all                    Show all releases
  best                   Show best releases
  category               Show releases by category
  country                Show releases by country
  new                    Show new releases
  newest                 Show newest releases
  year                   Show releases by year
  search                 Search releases
```
