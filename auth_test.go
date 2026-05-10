package hdrezka

import (
	"net/http"
	"testing"
)

func TestParseCookieString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []*http.Cookie
		wantErr bool
	}{
		{
			name:  "two cookies",
			input: "dle_user_id=123;dle_password=abc",
			want: []*http.Cookie{
				{Name: "dle_user_id", Value: "123", Domain: "hdrezka.ag", Path: "/"},
				{Name: "dle_password", Value: "abc", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:  "whitespace around entries and values",
			input: "  dle_user_id = 123 ; dle_password = abc  ",
			want: []*http.Cookie{
				{Name: "dle_user_id", Value: "123", Domain: "hdrezka.ag", Path: "/"},
				{Name: "dle_password", Value: "abc", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:  "trailing and consecutive separators",
			input: ";;dle_user_id=123;;;dle_password=abc;",
			want: []*http.Cookie{
				{Name: "dle_user_id", Value: "123", Domain: "hdrezka.ag", Path: "/"},
				{Name: "dle_password", Value: "abc", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:  "single cookie without semicolon",
			input: "dle_user_id=123",
			want: []*http.Cookie{
				{Name: "dle_user_id", Value: "123", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:  "empty value is preserved",
			input: "dle_user_id=",
			want: []*http.Cookie{
				{Name: "dle_user_id", Value: "", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:  "value containing equals sign",
			input: "token=a=b=c",
			want: []*http.Cookie{
				{Name: "token", Value: "a=b=c", Domain: "hdrezka.ag", Path: "/"},
			},
		},
		{
			name:    "missing equals sign",
			input:   "dle_user_id",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   "=value",
			wantErr: true,
		},
		{
			name:  "empty input yields no cookies",
			input: "",
			want:  nil,
		},
		{
			name:  "only separators yields no cookies",
			input: " ; ; ;",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCookieString(tt.input, "hdrezka.ag")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (cookies: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("cookie count mismatch: got %d, want %d (got=%+v)", len(got), len(tt.want), got)
			}
			for i, c := range got {
				w := tt.want[i]
				if c.Name != w.Name || c.Value != w.Value || c.Domain != w.Domain || c.Path != w.Path {
					t.Errorf("cookie[%d] = %+v, want %+v", i, c, w)
				}
			}
		})
	}
}
