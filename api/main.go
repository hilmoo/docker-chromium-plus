package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const MCPVersion = "0.21.0"

type config struct {
	ApiPort    string
	ChromePort string
}

type app struct {
	cfg config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func loadConfig() config {
	return config{
		ApiPort:    getEnv("API_PORT", "8080"),
		ChromePort: getEnv("CHROME_PORT", "9222"),
	}
}

func (a *app) screenshotHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("scrot", "-")
	cmd.Env = append(os.Environ(), "DISPLAY=:1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to create stdout pipe for scrot", "error", err)
		http.Error(w, "Failed to capture screenshot", http.StatusInternalServerError)
		return
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start scrot command", "error", err)
		http.Error(w, "Failed to start scrot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")

	if _, err := io.Copy(w, stdout); err != nil {
		slog.Error("Stream error while copying screenshot data", "error", err)
	}

	cmd.Wait()
}

func (a *app) proxyHandler(w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse("http://localhost:3000")
	if err != nil {
		slog.Error("Failed to parse target URL for proxy", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(url)

			pr.Out.URL.Path = "/"
		},
	}

	proxy.ServeHTTP(w, r)
}

func (a *app) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	cfg := loadConfig()
	app := &app{cfg: cfg}

	http.HandleFunc("/screenshot", app.screenshotHandler)
	http.HandleFunc("/browser", app.proxyHandler)
	http.HandleFunc("/health", app.healthHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stdioSession, err := newStdio(ctx, cfg)
	if err != nil {
		slog.Error("Failed to establish stdio session", "error", err)
		os.Exit(1)
	}
	defer stdioSession.Close()

	proxyServer := newServer(stdioSession, logger)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return proxyServer
	}, &mcp.StreamableHTTPOptions{JSONResponse: true, Logger: logger})
	http.Handle("/mcp", intercept204(mcpHandler))

	port := ":" + cfg.ApiPort

	srv := &http.Server{
		Addr: port,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("API server starting", "port", port)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start or crashed", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		slog.Info("Shutdown signal received, initiating graceful shutdown", "signal", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Graceful shutdown failed, forcing close", "error", err)

			if err := srv.Close(); err != nil {
				slog.Error("Error forcing server to close", "error", err)
			}
		} else {
			slog.Info("HTTP server stopped gracefully")
		}

		cancel()
	}

	slog.Info("Application exited")
}
