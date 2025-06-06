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
	"github.com/robfig/cron/v3"
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
		httprate.Limit(conf.LimitRequests, conf.LimitWindow, httprate.WithKeyByIP()),
	)

	if conf.UpdateAuthKey != "" {
		r.Get("/api/update", updateHandler(conf, s3))
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

	group, ctx := errgroup.WithContext(ctx)

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

	if conf.UpdateCron != "" {
		group.Go(func() error {
			if conf.UpdateOnStartup {
				if _, err := update(ctx, conf, s3, false); err != nil {
					slog.Error("Update failed", "error", err)
				}
			}

			schedule, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).
				Parse(conf.UpdateCron)
			if err != nil {
				return err
			}

			for {
				next := schedule.Next(time.Now())
				until := time.Until(next)
				slog.Info("Waiting for next update", "timestamp", &next, "in", until.Round(time.Second))

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(until):
					if _, err := update(ctx, conf, s3, false); err != nil {
						slog.Error("Update failed", "error", err)
						continue
					}
				}
			}
		})
	}

	err = group.Wait()
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func handleHTTPError(w http.ResponseWriter, msg string, status int) {
	slog.Error("Download failed", "error", msg, "status", status)
	http.Error(w, msg, status)
}
