package hdrezka

// subtitleLanguageCodes maps the language names HDrezka uses for subtitle
// tracks to ISO 639-1 codes. The site labels a track either in Russian or in
// the language's own name, so both spellings are listed.
var subtitleLanguageCodes = map[string]string{
	"Русский":     "ru",
	"Українська":  "uk",
	"Украинский":  "uk",
	"English":     "en",
	"Английский":  "en",
	"Deutsch":     "de",
	"Немецкий":    "de",
	"Français":    "fr",
	"Французский": "fr",
	"Español":     "es",
	"Испанский":   "es",
	"Italiano":    "it",
	"Итальянский": "it",
	"Polski":      "pl",
	"Польский":    "pl",
	"Türkçe":      "tr",
	"Турецкий":    "tr",

	// Post-Soviet languages.
	"Беларуская":      "be",
	"Белорусский":     "be",
	"Қазақша":         "kk",
	"Казахский":       "kk",
	"Oʻzbekcha":       "uz",
	"Ўзбекча":         "uz",
	"Узбекский":       "uz",
	"Azərbaycanca":    "az",
	"Азербайджанский": "az",
	"Հայերեն":         "hy",
	"Армянский":       "hy",
	"ქართული":         "ka",
	"Грузинский":      "ka",
	"Кыргызча":        "ky",
	"Киргизский":      "ky",
	"Тоҷикӣ":          "tg",
	"Таджикский":      "tg",
	"Türkmençe":       "tk",
	"Туркменский":     "tk",
	// Moldovan is Romanian; ISO 639-1 "mo" was withdrawn in favour of "ro".
	"Română":     "ro",
	"Молдавский": "ro",
	"Румынский":  "ro",
	"Latviešu":   "lv",
	"Латышский":  "lv",
	"Lietuvių":   "lt",
	"Литовский":  "lt",
	"Eesti":      "et",
	"Эстонский":  "et",
}

// SubtitleLanguageCode returns the ISO 639-1 code for a subtitle track name as
// it appears in Stream.Subtitles, e.g. "Русский" -> "ru". The second result is
// false for names not seen on the site, in which case the code is empty.
func SubtitleLanguageCode(lang string) (string, bool) {
	code, ok := subtitleLanguageCodes[lang]
	return code, ok
}
