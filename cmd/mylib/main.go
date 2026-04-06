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
	"github.com/bueti/mylib/internal/auth"
	"github.com/bueti/mylib/internal/config"
	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/enrich"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/opds"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/bueti/mylib/internal/watcher"
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

	if err := bootstrapAdmin(context.Background(), store); err != nil {
		return err
	}

	coverCache, err := covers.New(cfg.DataDir)
	if err != nil {
		return err
	}
	sc := scanner.New(store, cfg.LibraryRoots, coverCache)
	enricher := enrich.New(store, coverCache)

	// Async enrichment queue: scanner pushes book IDs in after insert,
	// a worker goroutine drains them in the background.
	enrichQueue := make(chan int64, 100)
	sc.EnrichQueue = enrichQueue

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go enricher.RunWorker(ctx, enrichQueue)

	// All API routes live under /api/... already, OPDS under /opds, and
	// the embedded SPA serves / with an SPA fallback for unknown paths.
	apiRouter := api.NewRouter(api.Deps{
		Store:       store,
		Scanner:     sc,
		Covers:      coverCache,
		Enricher:    enricher,
		LibraryRoot: cfg.LibraryRoots[0],
		EnrichQueue: enrichQueue,
	})
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

	// Start the fsnotify watcher so new files trigger scans in
	// near-real-time, on top of the periodic ticker fallback.
	if w, err := watcher.New(sc, cfg.LibraryRoots, 2*time.Second); err == nil {
		go func() {
			if err := w.Run(ctx); err != nil {
				slog.Warn("watcher stopped", "err", err)
			}
		}()
	} else {
		slog.Warn("could not start fs watcher", "err", err)
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

// bootstrapAdmin creates the initial admin user from MYLIB_ADMIN_USER
// and MYLIB_ADMIN_PASSWORD when the users table is empty. If the table
// is empty and the env vars are not set, it logs a loud warning.
func bootstrapAdmin(ctx context.Context, store *library.Store) error {
	count, err := store.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	user := os.Getenv("MYLIB_ADMIN_USER")
	pass := os.Getenv("MYLIB_ADMIN_PASSWORD")
	if user == "" || pass == "" {
		slog.Warn("no users exist and MYLIB_ADMIN_USER/MYLIB_ADMIN_PASSWORD not set; write endpoints will reject all requests until a user is created")
		return nil
	}
	hash, err := auth.HashPassword(pass)
	if err != nil {
		return err
	}
	if _, err := store.CreateUser(ctx, user, hash, library.RoleAdmin); err != nil {
		return err
	}
	slog.Info("bootstrapped admin user", "username", user)
	return nil
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
