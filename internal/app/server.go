package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/fx"

	"9router/proxy/internal/config"
	"9router/proxy/internal/db"
	"9router/proxy/internal/providers"
	"9router/proxy/internal/shutdown"
	"9router/proxy/internal/updater"
)

// ServerModule provides *http.Server and manages its lifecycle and background tasks.
var ServerModule = fx.Module("server",
	fx.Provide(
		ProvideServer,
	),
	fx.Invoke(
		func(*http.Server) {},
	),
)

// ServerParams defines inputs for constructing the HTTP server and lifecycle hooks.
type ServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *config.Config
	Repo      *db.Repo
	Handler   http.Handler
	CLIParams CLIParams
}

// ProvideServer creates *http.Server and registers lifecycle hooks.
func ProvideServer(p ServerParams) *http.Server {
	addr := fmt.Sprintf(":%d", p.Config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: p.Handler,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			autoUpdate := p.CLIParams.AutoUpdate
			if !autoUpdate && p.Repo != nil {
				if settings, sErr := p.Repo.GetSettings(); sErr == nil && settings != nil {
					autoUpdate = settings.AutoUpdate
				}
			}
			updater.StartBackgroundCheck(context.Background(), autoUpdate)
			log.Printf("[config] auto-update enabled=%v", autoUpdate)

			catalogPath := filepath.Join(filepath.Dir(p.Config.DatabasePath), "model-catalog.json")
			providers.StartBackgroundCatalogSync(context.Background(), nil, catalogPath)

			log.Printf("9Router Go Proxy (%s) starting on port %d", updater.CurrentVersion, p.Config.Port)

			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("Server failed: %v", err)
				}
			}()

			fmt.Fprintf(os.Stdout, "\n  🚀 9Router Go Proxy (%s) on %s\n\n", updater.CurrentVersion, addr)
			log.Printf("Server is ready to handle requests at %s", addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Fprintln(os.Stdout, "\n  Shutting down...")

			// Signal in-flight SSE streams to end promptly: the stall reader closes each
			// upstream body, handlers emit a final [DONE], and Shutdown completes well
			// within its deadline instead of waiting out the full timeout.
			shutdown.Cancel()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Server shutdown did not complete in time: %v", err)
			} else {
				log.Println("Server stopped gracefully")
			}
			return nil
		},
	})

	return server
}
