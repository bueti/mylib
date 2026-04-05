// Command mylib runs the ebook/PDF library server.
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

	"github.com/bueti/mylib/internal/api"
	"github.com/bueti/mylib/internal/config"
	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/opds"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/bueti/mylib/internal/webui"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogger(cfg.LogLevel)

	slog.Info("starting mylib",
		"roots", cfg.LibraryRoots,
		"data_dir", cfg.DataDir,
		"listen", cfg.Listen,
		"scan_interval", cfg.ScanInterval,
	)

	conn, err := db.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer conn.Close()

	store := library.New(conn)
	coverCache, err := covers.New(cfg.DataDir)
	if err != nil {
		return err
	}
	sc := scanner.New(store, cfg.LibraryRoots, coverCache)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// All API routes live under /api/... already, OPDS under /opds, and
	// the embedded SPA serves / with an SPA fallback for unknown paths.
	apiRouter := api.NewRouter(api.Deps{Store: store, Scanner: sc, Covers: coverCache})
	root := chi.NewRouter()
	// Delegate any /api/* or /opds* requests to their handlers; fall
	// through to the SPA for everything else.
	root.Handle("/api/*", apiRouter)
	opds.Mount(root, &opds.Handler{Store: store})
	spa := webui.Handler()
	root.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		spa.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start periodic scans in the background.
	go runPeriodicScans(ctx, sc, cfg.ScanInterval)

	// HTTP server goroutine.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http listening", "addr", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-serverErr:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// runPeriodicScans kicks off a scan immediately and then every interval.
// If interval is zero, it runs once and returns.
func runPeriodicScans(ctx context.Context, sc *scanner.Scanner, interval time.Duration) {
	if _, err := sc.ScanAll(ctx); err != nil {
		slog.Error("initial scan failed", "err", err)
	}
	if interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := sc.ScanAll(ctx); err != nil {
				slog.Error("periodic scan failed", "err", err)
			}
		}
	}
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})))
}
