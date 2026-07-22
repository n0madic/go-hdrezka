# hdrezka-msx

HTTP server that serves an [Media Station X](https://msx.benzac.de) portal on top of the
`hdrezka` library, so HDrezka can be browsed and watched on a TV.

The server only serves JSON documents describing the UI — media is played directly from the
HDrezka CDN by the TV, nothing is proxied through this process. No state is stored except an
in-memory metadata cache.

## Install

```bash
go install github.com/n0madic/go-hdrezka/cmd/hdrezka-msx@latest
```

## Run

```bash
hdrezka-msx --listen :8080
```

Then open <https://msx.benzac.de> on the TV (or in a browser) and enter the start parameter:

```
http://<host>:8080/msx/start.json
```

`<host>` is the address of the machine running the server as seen from the TV. When the server
runs behind a reverse proxy, pass its external address with `--public-url` so the nested links
point back to the right place.

## Flags

| Flag | Description |
|---|---|
| `--listen ADDR` | listen address (default `:8080`) |
| `--public-url URL` | external base URL when running behind a reverse proxy |
| `-m, --mirrors URL` | HDrezka mirrors to probe (default: built-in list) |
| `-p, --proxy URL` | proxy for site requests |
| `-r, --resolver IP` | DNS resolver for site requests |
| `--login NAME` / `--password PASS` | HDrezka account credentials (used together) |
| `--cookies STRING` | raw cookies for auth or anti-bot clearance (Cloudflare/Anubis) |
| `--no-antibot-solver` | disable automatic solving of Anubis proof-of-work challenges |
| `--mp4` | serve progressive MP4 instead of HLS (see Playback) |
| `-n, --items INT` | items per catalog page (default 60) |
| `--sizes` | show the file size of each quality (extra HEAD request per quality) |
| `--cache-ttl DURATION` | TTL of the metadata cache (default `10m`) |
| `--debug` | enable debug logging |

## Navigation

```
menu → poster grid → video card → translation → (season → episode) → quality → playback
```

CDN links are only obtainable through an AJAX call, so they are resolved on the quality page —
the last step before playback. That keeps it to one site request per click.

## Playback

Streams are served as **HLS** by default. The progressive MP4 files of the HDrezka CDN stall in
HTML5 players — the CDN answers HEAD with 502 and never delivers the initial media segment for
the open `Range: bytes=0-` request a `<video>` element sends, so playback ends in
"source is not supported". The HLS manifests of the same stream (what the site's own player uses)
are served correctly. `--mp4` switches back for devices that prefer a direct file.

The CDN also requires a `Referer` header on media segments — any value works, and every player
sends one, so this only matters when testing with `curl`.

Playback goes to the built-in MSX player, which lets each platform use its native decoder. Only
streams with subtitles go through the stock
[HTML5X plugin](https://msx.benzac.de/plugins/html5x.html), which attaches the tracks via
`html5x:subtitle:{LANGUAGE}:{LABEL}` properties.

## Pages

All documents live under `/msx/`; `/` redirects to `/msx/start.json`.

| Route | Purpose |
|---|---|
| `/msx/start.json` | start object |
| `/msx/menu.json` | root menu |
| `/msx/content/genre` | entry points of one site section |
| `/msx/content/covers` | poster grid (`type`, `genre`, `filter`, `category`, `country`, `year`, `n`) |
| `/msx/content/newest` | newest releases slider of a section |
| `/msx/content/categories` | genres of a section |
| `/msx/content/years` | release years |
| `/msx/content/countries` | countries |
| `/msx/content/video` | video card with translations |
| `/msx/content/seasons` | seasons of a translation |
| `/msx/content/episodes` | episodes of a season |
| `/msx/content/stream` | qualities with the playback action |
| `/msx/content/search` | search results (fed by the MSX input plugin) |

Local check:

```bash
curl -s localhost:8080/msx/start.json | jq
curl -s 'localhost:8080/msx/content/covers?type=new&genre=films' | jq '.items[0]'
```
