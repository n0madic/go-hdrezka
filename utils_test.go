package hdrezka

import (
	"encoding/base64"
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
