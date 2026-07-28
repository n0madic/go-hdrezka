package hdrezka

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
)

// encodedSample is a real obfuscated stream URL captured from HDrezka
// (2025-04-04). It exercises every known salt, including the 20-char
// "//_//JCQhIUAkJEBeIUAjJCRA" that a fixed 16-char strip would corrupt.
const encodedSample = "#hWzM2MHBdaHR0cHM6Ly9mZW1lcmV0ZXMub3JnLzI2ODA2NGQ0NDlmNTdmODUwNDU3YzEzY2Q3OGI4N2VkOjIwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJFVnpTREo0T//_//IyMjI14hISMjIUBATAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvb2xxMnEubXA0OmhsczptYW5pZmVzdC5tM3U4IG9yIGh0dHBzOi8vc3RyZWFtLnZvaWRib29zdC5jYy8yNjgwNjRkNDQ5ZjU3Zjg1MDQ1N2MxM2NkNzhiODdlZDoyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81L29scTJxLm1wNCBvciBodHRwczovL2ZlbWVyZXRlcy5vcmcvMjY4MDY0ZDQ0OWY1N2Y4NTA0NTdjMTNjZDc4Yjg3ZWQ6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS9vbHEycS5tcDQgb3IgaHR0cHM6Ly9zdHJlYW0udm9pZGJvb3N0LmNjLzI2ODA2NGQ0NDlmNTdmODUwNDU3YzEzY2Q3OGI4N2VkOjIwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJ//_//QEBAQEAhIyMhXl5eFVnpTREo0TTAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvb2xxMnEubXA0LFs0ODBwXWh0dHBzOi8vZmVtZXJldGVzLm9yZy8wMjczMTJmNjAzMzA2ZWZhMGI2MzdkNzRkN2EyZDFlYzoyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81LzlpbThpLm1wNDpobHM6bWFuaWZlc3QubTN1OCBvciBodHRwczovL3N0cmVhbS52b2lkYm9vc3QuY2MvMDI3MzEyZjYwMzMwNmVmYTBiNjM3ZDc0ZDdhMmQxZWM6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS85aW04aS5tcDQgb3IgaHR0cHM6Ly9mZW1lcmV0ZXMub3JnLzAyNzMxMmY2MDMzMDZlZmEwYjYzN2Q3NGQ3YTJkMWVjOj//_//Xl5eIUAjIyEhIyM=N0Lm0zdTggb3IgaHR0cHM6Ly9mZW1lcmV0ZXMub3JnLzAyNzMxMmY2MDMzMDZlZmEwYjYzN2Q3NGQ3YTJkMWVjOj//_//JCQjISFAIyFAIyM=IwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJFVnpTREo0TTAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvOWltOGkubXA0IG9yIGh0dHBzOi8vc3RyZWFtLnZvaWRib29zdC5jYy8wMjczMTJmNjAzMzA2ZWZhMGI2MzdkNzRkN2EyZDFlYzoyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81LzlpbThpLm1wNCxbNzIwcF1odHRwczovL2ZlbWVyZXRlcy5vcmcvZmUyMmE0NmI5ZDM4ZTI1MmNjNWY1ZWIyNGQ4MTQ3YzE6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS9jZmlzYy5tcDQ6aGxzOm1hbmlmZXN0Lm0zdTggb3IgaHR0cHM6Ly9zdHJlYW0udm9pZGJvb3N0LmNjL2ZlMjJhNDZiOWQzOGUyNTJjYzVmNWViMjRkODE0N2MxOjIwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJFVnpTREo0TTAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvY2Zpc2MubXA0OmhsczptYW5pZmVzdC5tM3U4IG9yIGh0dHBzOi8vZmVtZXJldGVzLm9yZy9mZTIyYTQ2YjlkMzhlMjUyY2M1ZjVlYjI0ZDgxNDdjMToyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81L2NmaXNjLm1wNCBvciBodHRwczovL3N0cmVhbS52b2lkYm9vc3QuY2MvZmUyMmE0NmI5ZDM4ZTI1MmNjNWY1ZWIyNGQ4MTQ3YzE6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS9jZmlzYy5tcDQsWzEwODBwXWh0dHBzOi8vZmVtZXJldGVzLm9yZy9jOTQ1ZjNhMmVmOWVlNTZhOGNlMjNmMzQzOWU2NGI4NzoyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81L3pwN2VjLm1wNDpobHM6bWFuaWZlc3QubTN1OCBvciBodHRwczovL3N0cmVhbS52b2lkYm9vc3QuY2MvYzk0NWYzYTJlZjllZTU2YThjZTIzZjM0MzllNjRiODc6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS96cDdlYy5tcDQgb3IgaHR0cHM6Ly9mZW1lcmV0ZXMub3JnL2M5NDVmM2EyZWY5ZWU1NmE4Y2UyM2YzNDM5ZTY0Yjg3OjIwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJFVnpTREo0TTAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvenA3ZWMubXA0LFsxMDgwcCBVbHRyYV1odHRwczovL2ZlbWVyZXRlcy5vcmcvYzk0NWYzYTJlZjllZTU2YThjZTIzZjM0MzllNjRiODc6MjAyNTA0MDQxNjpMMFJXVTJ0NlZIbHFiMlZZU25KdWJWVXhjMXB0UkVWelNESjRNMDFLU2xwbFV6QlFXV1F3Y210a2RtZGxVMmRZTDA5eFducE5LM1J5VHpKa0wzWlZZbTh3ZEdFcmNYTmFSbEJRVmxKcE9VczFha05aZW1sTldEaFZhR05yU3pseE4xWXdhV05yTDJ0Q1FsRTkvMS8xLzIvNC8zLzIvNS96cDdlYy5tcDQ6aGxzOm1hbmlmZXN0Lm0zdTggb3IgaHR0cHM6Ly9zdHJlYW0udm9pZGJvb3N0LmNjL2M5NDVmM2EyZWY5ZWU1NmE4Y2UyM2YzNDM5ZTY0Yjg3OjIwMjUwNDA0MTY6TDBSV1UydDZWSGxxYjJWWVNuSnViVlV4YzFwdFJFVnpTREo0TTAxS1NscGxVekJRV1dRd2NtdGtkbWRsVTJkWUwwOXhXbnBOSzNSeVR6SmtMM1pWWW04d2RHRXJjWE5hUmxCUVZsSnBPVXMxYWtOWmVtbE5XRGhWYUdOclN6bHhOMVl3YVdOckwydENRbEU5LzEvMS8yLzQvMy8yLzUvenA3ZWMubXA0IG9yIGh0dHBzOi8vc3RyZWFtLnZvaWRib29zdC5jYy9jOTQ1ZjNhMmVmOWVlNTZhOGNlMjNmMzQzOWU2NGI4NzoyMDI1MDQwNDE2OkwwUldVMnQ2VkhscWIyVllTbkp1YlZVeGMxcHRSRVZ6U0RKNE0wMUtTbHBsVXpCUVdXUXdjbXRrZG1kbFUyZFlMMDl4V25wTkszUnlUekprTDNaVlltOHdkR0VyY1hOYVJsQlFWbEpwT1VzMWFrTlplbWxOV0RoVmFHTnJTemx4TjFZd2FXTnJMMnRDUWxFOS8xLzEvMi80LzMvMi81L3pwN2VjLm1wNA=="

func TestDecodeURL(t *testing.T) {
	t.Parallel()

	// Synthetic case: an unknown 16-char salt must be stripped via the
	// app-style fallback ("//_//" + 16) so the remaining base64 decodes cleanly.
	payload := "the-real-decoded-payload-value"
	b64 := base64.StdEncoding.EncodeToString([]byte(payload))
	unknownSalt := "ABCDEFGHIJKLMNOP" // 16 chars, not in knownSalts
	syntheticEncoded := b64[:8] + "//_//" + unknownSalt + b64[8:]

	tests := []struct {
		name        string
		input       string
		want        string   // exact match (skipped when empty)
		contains    []string // substrings that must be present
		notContains []string // substrings that must be absent
	}{
		{
			name:  "plain quality-tagged URL is returned as-is",
			input: "[360p]http://example.com/video.mp4",
			want:  "[360p]http://example.com/video.mp4",
		},
		{
			name:  "plain http URL is returned as-is",
			input: "http://example.com/video.mp4",
			want:  "http://example.com/video.mp4",
		},
		{
			name:  "unknown 16-char salt degrades to app fallback",
			input: syntheticEncoded,
			want:  payload,
		},
		{
			name:  "real obfuscated sample decodes to plain stream URL",
			input: encodedSample,
			contains: []string{
				"[360p]", "[1080p Ultra]",
				"femeretes.org", "stream.voidboost.cc",
				":hls:manifest.m3u8",
			},
			notContains: []string{
				"//_//",
				"IyMjI14hISMjIUBA", "QEBAQEAhIyMhXl5e",
				"JCQhIUAkJEBeIUAjJCRA", "JCQjISFAIyFAIyM=", "Xl5eIUAjIyEhIyM=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeURL(tt.input)
			if err != nil {
				t.Fatalf("decodeURL returned error: %v", err)
			}
			if tt.want != "" && got != tt.want {
				t.Fatalf("decodeURL = %q, want %q", got, tt.want)
			}
			for _, sub := range tt.contains {
				if !strings.Contains(got, sub) {
					t.Errorf("decodeURL result missing %q", sub)
				}
			}
			for _, sub := range tt.notContains {
				if strings.Contains(got, sub) {
					t.Errorf("decodeURL result still contains %q", sub)
				}
			}
		})
	}
}

func TestParseSubtitles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "two languages",
			input: "[English]http://a.vtt,[Russian]http://b.vtt",
			want: map[string]string{
				"English": "http://a.vtt",
				"Russian": "http://b.vtt",
			},
		},
		{
			name:  "alternatives separated by ' or ' use the last URL",
			input: "[English]http://a.vtt or http://b.vtt",
			want: map[string]string{
				"English": "http://b.vtt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseSubtitles(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseSubtitles = %+v, want %+v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseSubtitles[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseStreamFormats(t *testing.T) {
	t.Parallel()

	decoded, err := decodeURL(encodedSample)
	if err != nil {
		t.Fatalf("decodeURL returned error: %v", err)
	}
	formats := parseStreamFormats(decoded)

	wantQualities := []string{"360p", "480p", "720p", "1080p", "1080p Ultra"}
	for _, q := range wantQualities {
		f, ok := formats[q]
		if !ok {
			t.Errorf("missing quality %q in %v", q, formats)
			continue
		}
		if !strings.HasSuffix(f.HLS, ":hls:manifest.m3u8") {
			t.Errorf("quality %q HLS = %q, want :hls:manifest.m3u8 suffix", q, f.HLS)
		}
		if !strings.HasSuffix(f.MP4, ".mp4") {
			t.Errorf("quality %q MP4 = %q, want .mp4 suffix", q, f.MP4)
		}
	}
}

func TestBoolTo10(t *testing.T) {
	t.Parallel()

	if got := boolTo10(true); got != "1" {
		t.Errorf("boolTo10(true) = %q, want %q", got, "1")
	}
	if got := boolTo10(false); got != "0" {
		t.Errorf("boolTo10(false) = %q, want %q", got, "0")
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, ""},
		{"negative", -1, ""},
		{"bytes", 512, "512 B"},
		{"exact KB", 1024, "1.0 KB"},
		{"KB", 1536, "1.5 KB"},
		{"MB", 5 * 1024 * 1024, "5.0 MB"},
		{"GB", 3*1024*1024*1024 + 512*1024*1024, "3.5 GB"},
		{"TB", 2 * 1024 * 1024 * 1024 * 1024, "2.0 TB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBytes(tt.bytes); got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestParseQualityLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		want        string
		wantPremium bool
	}{
		{name: "plain", raw: "1080p", want: "1080p"},
		{name: "plain ultra", raw: "1080p Ultra", want: "1080p Ultra"},
		{
			name:        "premium markup",
			raw:         `<span class="pjs-prem-quality">1080p Ultra<img src="https://static.hdrezka.ac/templates/hdrezka/images/prem-icon.svg" alt=""></span>`,
			want:        "1080p Ultra",
			wantPremium: true,
		},
		{
			name:        "entities and extra spacing",
			raw:         "<span class=\"pjs-prem-quality\">1080p&nbsp; Ultra\n</span>",
			want:        "1080p Ultra",
			wantPremium: true,
		},
		{name: "markup without text keeps the raw tag", raw: "<span></span>", want: "<span></span>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, premium := parseQualityLabel(tt.raw)
			if got != tt.want {
				t.Errorf("label = %q, want %q", got, tt.want)
			}
			if premium != tt.wantPremium {
				t.Errorf("premium = %v, want %v", premium, tt.wantPremium)
			}
		})
	}
}

// TestParseStreamFormatsPremium checks the markup HDrezka serves to accounts
// without a subscription: the quality tag carries a span with the premium badge
// instead of a bare label.
func TestParseStreamFormatsPremium(t *testing.T) {
	t.Parallel()

	const premiumTag = `<span class="pjs-prem-quality">1080p Ultra<img src="https://static.hdrezka.ac/templates/hdrezka/images/prem-icon.svg" alt=""></span>`
	formats := parseStreamFormats("[1080p]https://cdn.test/a.mp4:hls:manifest.m3u8 or https://cdn.test/a.mp4,[" +
		premiumTag + "]https://cdn.test/b.mp4:hls:manifest.m3u8 or https://cdn.test/b.mp4")

	if len(formats) != 2 {
		t.Fatalf("parseStreamFormats = %+v, want 2 formats", formats)
	}
	regular, ok := formats["1080p"]
	if !ok {
		t.Fatalf("missing quality %q in %+v", "1080p", formats)
	}
	if regular.IsPremium {
		t.Error("1080p reported as premium")
	}
	ultra, ok := formats["1080p Ultra"]
	if !ok {
		t.Fatalf("missing quality %q in %+v", "1080p Ultra", formats)
	}
	if !ultra.IsPremium {
		t.Error("1080p Ultra not reported as premium")
	}
	if ultra.HLS != "https://cdn.test/b.mp4:hls:manifest.m3u8" || ultra.MP4 != "https://cdn.test/b.mp4" {
		t.Errorf("1080p Ultra = %+v, want the b.mp4 URLs", ultra)
	}
}

func TestStreamQualities(t *testing.T) {
	t.Parallel()

	stream := &Stream{Formats: map[string]VideoFormat{
		"480p":        {},
		"1080p Ultra": {IsPremium: true},
		"1080p":       {},
		"2160p":       {},
		"720p":        {},
		"origin":      {},
	}}
	want := []string{"2160p", "1080p Ultra", "1080p", "720p", "480p", "origin"}
	if got := stream.Qualities(); !slices.Equal(got, want) {
		t.Errorf("Qualities() = %v, want %v", got, want)
	}
	if got := (&Stream{}).Qualities(); len(got) != 0 {
		t.Errorf("Qualities() on empty stream = %v, want empty", got)
	}
}

func TestStreamBestQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		formats map[string]VideoFormat
		want    string
	}{
		{
			name:    "prefers highest known",
			formats: map[string]VideoFormat{"480p": {}, "1080p": {}, "720p": {}},
			want:    "1080p",
		},
		{
			name:    "ultra over 1080p",
			formats: map[string]VideoFormat{"1080p": {}, "1080p Ultra": {}},
			want:    "1080p Ultra",
		},
		{
			name:    "premium ultra skipped while a regular format exists",
			formats: map[string]VideoFormat{"1080p": {}, "1080p Ultra": {IsPremium: true}},
			want:    "1080p",
		},
		{
			name:    "premium used when nothing else is available",
			formats: map[string]VideoFormat{"1080p Ultra": {IsPremium: true}},
			want:    "1080p Ultra",
		},
		{
			name:    "single",
			formats: map[string]VideoFormat{"360p": {}},
			want:    "360p",
		},
		{
			name:    "unknown labels pick highest resolution",
			formats: map[string]VideoFormat{"144p": {}, "240p": {}},
			want:    "240p",
		},
		{
			name:    "empty",
			formats: map[string]VideoFormat{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Stream{Formats: tt.formats}
			if got := s.bestQuality(); got != tt.want {
				t.Errorf("bestQuality() = %q, want %q", got, tt.want)
			}
		})
	}
}
