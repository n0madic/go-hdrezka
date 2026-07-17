package hdrezka

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// anubisPassPath is the endpoint (relative to the challenge's base prefix) that
// validates a solved proof-of-work and issues the clearance cookie.
const anubisPassPath = "/.within.website/x/cmd/anubis/api/pass-challenge"

// anubisMaxIterations bounds the proof-of-work search so an absurd difficulty
// cannot hang the client. Difficulty 6 already averages ~16.7M hashes; the cap
// leaves generous headroom for the realistic 2-5 range while staying finite.
const anubisMaxIterations = 1 << 30

// anubisChallenge mirrors the JSON embedded in the #anubis_challenge script of
// an Anubis interstitial. Only the fields required to solve are decoded.
type anubisChallenge struct {
	Rules struct {
		Algorithm  string `json:"algorithm"`
		Difficulty int    `json:"difficulty"`
	} `json:"rules"`
	Challenge struct {
		ID         string `json:"id"`
		RandomData string `json:"randomData"`
	} `json:"challenge"`
}

// solveAnubisPoW finds a nonce such that SHA-256(randomData + decimal(nonce))
// begins with `difficulty` zero hex nibbles, mirroring the Anubis client worker
// (fast/slow both use this SHA-256 scheme). It returns the hex-encoded digest
// and the winning nonce.
func solveAnubisPoW(randomData string, difficulty int) (string, int, error) {
	if difficulty <= 0 {
		return "", 0, fmt.Errorf("invalid Anubis difficulty %d", difficulty)
	}
	fullZeroBytes := difficulty / 2
	oddNibble := difficulty%2 != 0
	for nonce := range anubisMaxIterations {
		sum := sha256.Sum256([]byte(randomData + strconv.Itoa(nonce)))
		ok := true
		for i := range fullZeroBytes {
			if sum[i] != 0 {
				ok = false
				break
			}
		}
		if ok && oddNibble && sum[fullZeroBytes]>>4 != 0 {
			ok = false
		}
		if ok {
			return hex.EncodeToString(sum[:]), nonce, nil
		}
	}
	return "", 0, fmt.Errorf("Anubis proof-of-work not solved within %d iterations (difficulty %d)", anubisMaxIterations, difficulty)
}

// solveAnubis parses the Anubis challenge embedded in doc, computes the
// proof-of-work and submits it to the pass-challenge endpoint on pageURL's
// host. On success the site stores the clearance cookie in the client jar, so
// the caller can simply retry the original request.
func (r *HDRezka) solveAnubis(doc *goquery.Document, pageURL *url.URL) error {
	raw := strings.TrimSpace(doc.Find("#anubis_challenge").First().Text())
	if raw == "" {
		return errors.New("Anubis challenge data not found")
	}
	var ch anubisChallenge
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		return fmt.Errorf("decode Anubis challenge: %w", err)
	}
	if ch.Challenge.RandomData == "" {
		return errors.New("Anubis challenge is missing randomData")
	}

	// The base prefix is a JSON string ("" by default) prepended to Anubis paths.
	basePrefix := ""
	if bp := strings.TrimSpace(doc.Find("#anubis_base_prefix").First().Text()); bp != "" {
		_ = json.Unmarshal([]byte(bp), &basePrefix)
	}

	start := time.Now()
	hash, nonce, err := solveAnubisPoW(ch.Challenge.RandomData, ch.Rules.Difficulty)
	if err != nil {
		return err
	}

	passURL := &url.URL{
		Scheme: pageURL.Scheme,
		Host:   pageURL.Host,
		Path:   basePrefix + anubisPassPath,
	}
	q := url.Values{
		"id":          {ch.Challenge.ID},
		"response":    {hash},
		"nonce":       {strconv.Itoa(nonce)},
		"redir":       {pageURL.String()},
		"elapsedTime": {strconv.FormatInt(time.Since(start).Milliseconds()+500, 10)},
	}
	passURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, passURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", pageURL.String())

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// The endpoint answers 302 (redirect back to the page, followed by the
	// client) or 200; the clearance cookie is captured by the jar either way.
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Anubis pass-challenge rejected the solution: %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}
