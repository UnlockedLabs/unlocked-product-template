// Package server is the product's HTTP API.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Version is what GET <prefix>/api/version reports, the same JSON as core's
// and the tutor's /api/version.
type Version struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Branch  string `json:"branch"`
}

// VersionFromEnv reads the version, commit and branch the Dockerfile bakes
// into the image from its build args. Each is "unknown" when it isn't set.
func VersionFromEnv() Version {
	return Version{
		Version: envOr("APP_VERSION", "unknown"),
		Commit:  envOr("GIT_COMMIT", "unknown"),
		Branch:  envOr("GIT_BRANCH", "unknown"),
	}
}

// NewHandler returns the API's routes, all under prefix: the path ingress
// sends this product's requests to, such as "/example". An empty prefix
// serves from the root.
func NewHandler(prefix string, v Version) http.Handler {
	prefix = strings.TrimRight(prefix, "/")
	mux := http.NewServeMux()
	// The platform chart's readiness probe. Same path and body as core's.
	mux.HandleFunc("GET "+prefix+"/api/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("write healthcheck response", "err", err)
		}
	})
	mux.HandleFunc("GET "+prefix+"/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("write version response", "err", err)
		}
	})
	return mux
}

// Serve runs the API on addr until ctx is canceled, then shuts it down,
// giving requests in flight up to 10 seconds to finish.
func Serve(ctx context.Context, addr, prefix string, v Version) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           NewHandler(prefix, v),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	slog.Info("serving", "addr", addr, "prefix", prefix, "version", v.Version)

	select {
	case err := <-errs:
		return fmt.Errorf("serve on %s: %w", addr, err)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve on %s: %w", addr, err)
	}
	slog.Info("stopped serving")
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
