package hdrezka

import "testing"

func TestSubtitleLanguageCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang   string
		want   string
		wantOK bool
	}{
		{lang: "Русский", want: "ru", wantOK: true},
		{lang: "Українська", want: "uk", wantOK: true},
		{lang: "Қазақша", want: "kk", wantOK: true},
		{lang: "Молдавский", want: "ro", wantOK: true},
		{lang: "Հայերեն", want: "hy", wantOK: true},
		{lang: "Klingon", want: "", wantOK: false},
		{lang: "", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()

			got, ok := SubtitleLanguageCode(tt.lang)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("SubtitleLanguageCode(%q) = %q, %v; want %q, %v", tt.lang, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestSubtitleLanguageCodesWellFormed guards the table itself: every entry must
// be a two-letter lowercase ISO 639-1 code, since players match tracks by it.
func TestSubtitleLanguageCodesWellFormed(t *testing.T) {
	t.Parallel()

	for lang, code := range subtitleLanguageCodes {
		if len(code) != 2 {
			t.Errorf("language %q has code %q, want two letters", lang, code)
			continue
		}
		for _, r := range code {
			if r < 'a' || r > 'z' {
				t.Errorf("language %q has code %q, want lowercase letters", lang, code)
				break
			}
		}
	}
}
