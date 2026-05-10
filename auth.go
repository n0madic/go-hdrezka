package hdrezka

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// defaultUserAgent is sent on every request. HDrezka mirrors reject empty
// or non-browser User-Agent strings on some endpoints.
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Login authenticates against /ajax/login/ using the standard DataLife Engine
// login form. On success the session cookies (dle_user_id, dle_password and
// related) are stored in r.Client.Jar and applied to all subsequent requests.
func (r *HDRezka) Login(login, password string) error {
	loginURL := r.URL.JoinPath("/ajax/login/").String()
	form := url.Values{
		"login_name":     {login},
		"login_password": {password},
		"login_not_save": {"1"},
		"login":          {"submit"},
	}

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", r.URL.String()+"/")

	// Some HDrezka mirrors respond to a successful login with a 30x redirect
	// to "/" (auth cookies arriving in Set-Cookie headers), and to a failed
	// login with HTTP 200 + JSON {"success": false, "message": "..."}.
	// Disable auto-redirect for this single request so we can distinguish.
	prevCheck := r.Client.CheckRedirect
	r.Client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := r.Client.Do(req)
	r.Client.CheckRedirect = prevCheck
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		// HDrezka responds to a successful login with a 30x to "/". Failed
		// logins return 200 + JSON. The exact set of session cookies depends
		// on login_not_save (persistent dle_user_id/dle_password vs.
		// session-only PHPSESSID), so the redirect is the primary success
		// signal. As a sanity check, ensure PHPSESSID actually landed in the
		// jar — if it didn't, the cookie pipeline is broken and any later
		// authenticated request would be silently treated as anonymous.
		for _, c := range r.Client.Jar.Cookies(r.URL) {
			if c.Name == "PHPSESSID" && c.Value != "" {
				return nil
			}
		}
		return errors.New("login redirected but PHPSESSID was not stored in the cookie jar")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("login response read: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return fmt.Errorf("login response is not JSON (HTTP %d, content-type %q): %s",
			resp.StatusCode, resp.Header.Get("Content-Type"), snippet)
	}
	if !result.Success {
		msg := result.Message
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("login failed: %s", msg)
	}
	return nil
}

// SetCookies stores raw cookies on the client jar so they are sent on every
// subsequent request to the site domain. Useful when the user has copied
// dle_user_id / dle_password from the browser and does not want to log in
// programmatically.
//
// Format: "name=value;name=value;..." (whitespace is trimmed).
func (r *HDRezka) SetCookies(cookieStr string) error {
	cookies, err := parseCookieString(cookieStr, r.URL.Hostname())
	if err != nil {
		return err
	}
	if len(cookies) == 0 {
		return errors.New("no valid cookies provided")
	}
	r.Client.Jar.SetCookies(r.URL, cookies)
	return nil
}

// parseCookieString parses a "name=value;name=value" string into HTTP cookies
// scoped to the given domain. Empty entries are ignored. Entries without
// '=' or with empty names are reported as errors.
func parseCookieString(cookieStr, domain string) ([]*http.Cookie, error) {
	var cookies []*http.Cookie
	for _, entry := range strings.Split(cookieStr, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		eq := strings.IndexByte(entry, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("invalid cookie entry %q: expected name=value", entry)
		}
		name := strings.TrimSpace(entry[:eq])
		value := strings.TrimSpace(entry[eq+1:])
		if name == "" {
			return nil, fmt.Errorf("invalid cookie entry %q: empty name", entry)
		}
		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   "/",
		})
	}
	return cookies, nil
}
