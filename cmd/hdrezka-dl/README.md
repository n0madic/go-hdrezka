# hdrezka-dl

Utility that downloads videos from the HDrezka site

## Install

```
go install github.com/n0madic/go-hdrezka/cmd/hdrezka-dl@latest
```

## Help

```
Usage: hdrezka-dl [--info] [--overwrite] [--quality QUALITY] [--season SEASON] [--episodes EPISODES] [--translation NAME] [--subtitle LANG] URL [OUTPUT]

Positional arguments:
  URL                    url for download video
  OUTPUT                 output file or path for downloaded video

Options:
  --info, -i             show info about video only
  --overwrite, -o        overwrite output file if exists
  --quality QUALITY, -q QUALITY
                         quality for download video [default: 1080p]
  --season SEASON, -s SEASON
                         season for download series
  --episodes EPISODES, -e EPISODES
                         range of episodes for download (required --season arg)
  --translation NAME, -t NAME
                         translation for download video
  --subtitle LANG, -c LANG
                         get subtitle for downloaded video
  --help, -h             display this help and exit
```
