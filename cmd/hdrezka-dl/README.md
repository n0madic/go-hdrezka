# hdrezka-dl

Utility that downloads videos from the HDrezka site

## Install

```
go install github.com/n0madic/go-hdrezka/cmd/hdrezka-dl@latest
```

## Help

```
Usage: hdrezka-dl [--base-url URL] [--info] [--max-attempt INT] [--overwrite] [--quality QUALITY] [--season RANGE] [--episodes RANGE] [--translation NAME] [--subtitle LANG] [--resolver IP] [--proxy URL] [--hls] [--login NAME] [--password PASS] [--cookies STRING] URL [OUTPUT]

Positional arguments:
  URL                    url for download video
  OUTPUT                 output file or path for downloaded video

Options:
  --base-url URL, -b URL
                         base URL of hdrezka site (e.g., https://hdrezka.ag)
  --info, -i             show info about video only
  --max-attempt INT, -m INT
                         max attempts for download file [default: 3]
  --overwrite, -o        overwrite output file if exists
  --quality QUALITY, -q QUALITY
                         quality for download video [default: 1080p]
  --season RANGE, -s RANGE
                         season or range of seasons to download (e.g. 1, 2-3, 1,3,5)
  --episodes RANGE, -e RANGE
                         range of episodes to download, requires single --season (e.g. 1, 3-5, 1,3,7-9)
  --translation NAME, -t NAME
                         translation for download video
  --subtitle LANG, -c LANG
                         get subtitle for downloaded video
  --resolver IP, -r IP   DNS resolver for download video
  --proxy URL, -p URL    proxy for download video (supports HTTP, HTTPS, SOCKS5)
  --hls, -l              use HLS instead of MP4 for download video
  --login NAME           hdrezka account login (email or username), requires --password
  --password PASS        hdrezka account password, requires --login
  --cookies STRING       raw cookies string, e.g. "dle_user_id=123;dle_password=abc"
  --help, -h             display this help and exit
```

## Authentication

1080p / 1080p Ultra quality, premium audio tracks and 18+ titles are gated behind a registered account. Pass either `--login`/`--password` (the tool will POST to `/ajax/login/`) or `--cookies` with a raw `dle_user_id=...;dle_password=...` string copied from the browser. The session cookies are reused for all metadata, AJAX and download requests.

```sh
hdrezka-dl --login user@example.com --password 'secret' -q 1080p https://hdrezka.ag/films/.../12345-foo.html
hdrezka-dl --cookies "dle_user_id=123;dle_password=<md5>" -i https://hdrezka.ag/films/.../12345-foo.html
```
