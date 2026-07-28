package main

import "testing"

func TestSubtitleOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		lang   string
		want   string
	}{
		{
			name:   "known language uses its ISO code",
			output: "Old (2021).mp4",
			lang:   "Русский",
			want:   "Old (2021).ru.vtt",
		},
		{
			name:   "episode file",
			output: "s01e02 Dark.mp4",
			lang:   "English",
			want:   "s01e02 Dark.en.vtt",
		},
		{
			name:   "HLS output extension",
			output: "Old (2021).ts",
			lang:   "Українська",
			want:   "Old (2021).uk.vtt",
		},
		{
			name:   "unknown language falls back to the track name",
			output: "Old (2021).mp4",
			lang:   "Klingon",
			want:   "Old (2021).Klingon.vtt",
		},
		{
			name:   "no language keeps the plain name",
			output: "Old (2021).mp4",
			want:   "Old (2021).vtt",
		},
		{
			name:   "name without extension",
			output: "Old",
			lang:   "Русский",
			want:   "Old.ru.vtt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := subtitleOutput(tt.output, tt.lang); got != tt.want {
				t.Errorf("subtitleOutput(%q, %q) = %q, want %q", tt.output, tt.lang, got, tt.want)
			}
		})
	}
}
