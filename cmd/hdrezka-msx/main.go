package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/n0madic/go-hdrezka"
)

var args struct {
	Listen    string        `arg:"--listen" placeholder:"ADDR" default:":8080" help:"listen address of the MSX portal"`
	PublicURL string        `arg:"--public-url" placeholder:"URL" help:"external base URL when running behind a reverse proxy"`
	Mirrors   []string      `arg:"-m,--mirrors" placeholder:"URL" help:"hdrezka mirrors to probe (default: built-in list)"`
	Proxy     string        `arg:"-p,--proxy" placeholder:"URL" help:"proxy for site requests"`
	Resolver  string        `arg:"-r,--resolver" placeholder:"IP" help:"DNS resolver for site requests"`
	Login     string        `arg:"--login" placeholder:"NAME" help:"hdrezka account login (email or username), requires --password"`
	Password  string        `arg:"--password" placeholder:"PASS" help:"hdrezka account password, requires --login"`
	Cookies   string        `arg:"--cookies" placeholder:"STRING" help:"raw cookies string for auth or anti-bot clearance (Cloudflare/Anubis)"`
	NoSolver  bool          `arg:"--no-antibot-solver" help:"disable automatic solving of Anubis proof-of-work anti-bot challenges"`
	UseMP4    bool          `arg:"--mp4" help:"serve progressive MP4 instead of HLS (many players stall on the HDrezka MP4 CDN)"`
	Items     int           `arg:"-n,--items" placeholder:"INT" default:"60" help:"number of items per catalog page"`
	Sizes     bool          `arg:"--sizes" help:"show file size for each quality (extra HEAD request per quality)"`
	CacheTTL  time.Duration `arg:"--cache-ttl" placeholder:"DURATION" default:"10m" help:"TTL of the metadata cache"`
	Debug     bool          `arg:"--debug" help:"enable debug logging"`
}

func main() {
	arg.MustParse(&args)

	level := slog.LevelInfo
	if args.Debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if (args.Login != "") != (args.Password != "") {
		slog.Error("--login and --password must be used together")
		os.Exit(1)
	}
	if args.Cookies != "" && args.Login != "" {
		slog.Error("--cookies cannot be combined with --login/--password")
		os.Exit(1)
	}
	if args.Items <= 0 {
		slog.Error("--items must be positive")
		os.Exit(1)
	}

	rezka := hdrezka.New().
		WithMirrors(args.Mirrors...).
		WithProxy(args.Proxy).
		WithResolver(args.Resolver).
		WithCookies(args.Cookies).
		WithAntiBotSolver(!args.NoSolver)
	if err := rezka.Init(); err != nil {
		slog.Error("site initialization failed", "error", err)
		os.Exit(2)
	}
	slog.Info("connected to hdrezka", "mirror", rezka.URL.String())

	if args.Login != "" {
		if err := rezka.Login(args.Login, args.Password); err != nil {
			slog.Error("login failed", "error", err)
			os.Exit(2)
		}
		slog.Info("logged in", "login", args.Login)
	}

	srv := newServer(rezka, newCache(args.CacheTTL), config{
		publicURL: args.PublicURL,
		maxItems:  args.Items,
		useMP4:    args.UseMP4,
		showSizes: args.Sizes,
	})

	httpServer := &http.Server{
		Addr:              args.Listen,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("MSX portal started", "listen", args.Listen, "start_parameter", pathStart)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
