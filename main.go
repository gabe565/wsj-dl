package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

var ErrUpstream = errors.New("upstream error")

func run() error {
	conf, err := Load()
	if err != nil {
		return err
	}

	s3, err := NewS3(conf)
	if err != nil {
		return err
	}

	r := chi.NewRouter()

	r.Use(middleware.Heartbeat("/ping"))
	if conf.TrustedProxies != nil {
		r.Use(middleware.ClientIPFromXFF(conf.TrustedProxies...))
	}
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(httprate.LimitBy(conf.LimitRequests, conf.LimitWindow, func(r *http.Request) (string, error) {
		return middleware.GetClientIP(r.Context()), nil
	}))

	upload := uploadHandler(conf, s3)
	r.Get("/api/upload", upload)
	r.Post("/api/upload", upload)

	if conf.RedirectToLatest {
		r.Get("/", redirectLatest())
	}

	r.Get("/*", get(conf, s3))

	server := &http.Server{
		Addr:        conf.ListenAddress,
		Handler:     r,
		ReadTimeout: 5 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	if issue, err := findLatest(ctx, conf, s3); err == nil {
		slog.Info("Found latest file", "issue", issue)
		latest.Store(issue)
	} else {
		return fmt.Errorf("failed to find latest file: %w", err)
	}

	errCh := make(chan error, 1)

	go func() {
		slog.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		const timeout = 30 * time.Second

		ctx, cancelTimeout := context.WithTimeout(context.Background(), timeout)
		defer cancelTimeout()

		ctx, cancelSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		defer cancelSignal()

		slog.Info("Shutting down", "timeout", timeout)
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func handleHTTPError(w http.ResponseWriter, msg string, status int) {
	slog.Error("Download failed", "error", msg, "status", status)
	http.Error(w, msg, status)
}
