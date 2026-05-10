package hdrezka

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// WithProxy schedules all subsequent requests to be routed through the given
// proxy URL. Supported schemes: http(s), socks5, socks5h. The transport is
// rebuilt lazily on the next network call. Pass an empty string to clear a
// previously configured proxy.
func (r *HDRezka) WithProxy(addr string) *HDRezka {
	r.proxyAddr = addr
	r.initialized = false
	return r
}

// WithResolver makes the client use the given DNS server (host without port,
// UDP/53) for name resolution. Empty addr falls back to the system resolver.
// Like WithProxy, the change takes effect on the next network call.
func (r *HDRezka) WithResolver(addr string) *HDRezka {
	r.resolverAddr = addr
	r.initialized = false
	return r
}

func buildTransport(proxyAddr, resolverAddr string) (http.RoundTripper, error) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if resolverAddr != "" {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 10 * time.Second}
				return d.DialContext(ctx, "udp", resolverAddr+":53")
			},
		}
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}

	if proxyAddr == "" {
		return transport, nil
	}

	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}

	switch strings.ToLower(proxyURL.Scheme) {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if proxyURL.User != nil {
			auth = &proxy.Auth{User: proxyURL.User.Username()}
			if password, ok := proxyURL.User.Password(); ok {
				auth.Password = password
			}
		}
		socksDialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, dialer)
		if err != nil {
			return nil, fmt.Errorf("socks5 dialer: %w", err)
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return socksDialer.Dial(network, addr)
		}
	default:
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return transport, nil
}
