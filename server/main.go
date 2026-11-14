// Command memebank runs the memebank API and serves the built web app.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/radryc/memebank/server/internal/api"
	"github.com/radryc/memebank/server/internal/seed"
	"github.com/radryc/memebank/server/internal/store"
)

func main() {
	addr := flag.String("addr", env("MEMEBANK_ADDR", ":8080"), "listen address")
	dataDir := flag.String("data", env("MEMEBANK_DATA", "data"), "data directory for the database and uploads")
	staticDir := flag.String("static", env("MEMEBANK_STATIC", defaultStaticDir()), "directory with the built frontend")
	noSeed := flag.Bool("no-seed", false, "skip template seeding")
	debug := flag.Bool("debug", env("MEMEBANK_DEBUG", "") == "1", "enable debug logging")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	uploadsDir := filepath.Join(*dataDir, "uploads")
	for _, dir := range []string{*dataDir, filepath.Join(uploadsDir, "templates"), filepath.Join(uploadsDir, "memes")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Error("could not create directory", "dir", dir, "error", err)
			os.Exit(1)
		}
	}

	st, err := store.Open(filepath.Join(*dataDir, "memebank.db"))
	if err != nil {
		log.Error("could not open database", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := seed.Ensure(ctx, st, uploadsDir, log, !*noSeed); err != nil {
		log.Error("template seeding failed", "error", err)
	}

	handler := api.New(st, uploadsDir, *staticDir, log)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Info("memebank is up", "addr", *addr, "data", *dataDir, "static", *staticDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultStaticDir picks the first plausible Vite build output.
func defaultStaticDir() string {
	for _, candidate := range []string{
		filepath.Join("..", "web", "dist"),
		filepath.Join("web", "dist"),
	} {
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("..", "web", "dist")
}
