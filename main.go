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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"golang.org/x/sync/errgroup"
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
	r.Use(
		middleware.Heartbeat("/ping"),
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
	)

	r.With(httprate.Limit(conf.UploadLimitRequests, conf.UploadLimitWindow, httprate.WithKeyByIP())).
		Get("/api/upload", upload(conf, s3))

	r.With(httprate.Limit(conf.GetLimitRequests, conf.GetLimitWindow, httprate.WithKeyByIP())).
		Get("/*", get(conf, s3))

	server := &http.Server{
		Addr:        conf.ListenAddress,
		Handler:     r,
		ReadTimeout: 5 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)

	go func() {
		v, err := findLatest(ctx, conf, s3)
		if err != nil {
			slog.Error("Failed to find latest file", "error", err)
			return
		}

		slog.Info("Found latest file", "path", v)
		latest.Store(&v)
	}()

	group.Go(func() error {
		slog.Info("Starting server", "addr", server.Addr)
		return server.ListenAndServe()
	})

	group.Go(func() error {
		<-ctx.Done()
		slog.Info("Shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	})

	err = group.Wait()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func handleHTTPError(w http.ResponseWriter, msg string, status int) {
	slog.Error("Download failed", "error", msg, "status", status)
	http.Error(w, msg, status)
}
