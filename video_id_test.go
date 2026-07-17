package hdrezka

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractVideoID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "fav holder",
			html: `<a class="b-userset__fav_holder" data-post_id="1171"></a>`,
			want: "1171",
		},
		{
			name: "post_id input",
			html: `<input type="hidden" id="post_id" value="1171">`,
			want: "1171",
		},
		{
			name: "post container id",
			html: `<div class="b-post" id="post-1171"></div>`,
			want: "1171",
		},
		{
			name: "initCDN movies fallback",
			html: `<script>sof.tv.initCDNMoviesEvents(1171, 110, {"streams":""});</script>`,
			want: "1171",
		},
		{
			name: "initCDN series fallback",
			html: `<script>sof.tv.initCDNSeriesEvents(1171, 110, 1, 1, {"streams":""});</script>`,
			want: "1171",
		},
		{
			name: "not found",
			html: `<div>no id anywhere</div>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse html: %v", err)
			}
			if got := extractVideoID(doc, tt.html); got != tt.want {
				t.Errorf("extractVideoID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsChallengePage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want bool
	}{
		{
			name: "cloudflare just a moment",
			html: `<html><head><title>Just a moment...</title></head><body></body></html>`,
			want: true,
		},
		{
			name: "cloudflare challenge script",
			html: `<html><head><title></title></head><body><script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script></body></html>`,
			want: true,
		},
		{
			name: "ddos-guard",
			html: `<html><head><title>DDoS-Guard</title></head><body></body></html>`,
			want: true,
		},
		{
			name: "anubis by challenge element",
			html: `<html><head><title>Проверяем, что вы не бот!</title></head><body><div id="anubis_challenge"></div></body></html>`,
			want: true,
		},
		{
			name: "anubis by within.website asset",
			html: `<html><head><title></title></head><body><script type="module" src="/.within.website/x/cmd/anubis/static/js/main.mjs"></script></body></html>`,
			want: true,
		},
		{
			name: "real page",
			html: `<html><head><title>Мумия (2026)</title></head><body><a class="b-userset__fav_holder" data-post_id="86909"></a></body></html>`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse html: %v", err)
			}
			if got := isChallengePage(doc); got != tt.want {
				t.Errorf("isChallengePage() = %v, want %v", got, tt.want)
			}
		})
	}
}
