// Main entrypoint of Shortener app
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/grpcserver"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func Run(ctx context.Context) error {
	config.ParseFlags()

	if err := logger.Initialize(config.Options.FlagLogLevel); err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}

	printBuildInfo()

	auditObserver := setupAudit()

	repo, err := repository.NewRepository()
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	service.InitRepo(repo)
	service.InitDeleteManager(repo, 3)

	handlerChain := createHandlerChain(auditObserver)

	go startPprofServer()

	httpServer := createHTTPServer(handlerChain)
	grpcServer := grpcserver.NewServer(grpcserver.ServerConfig{
		Port:   ":3200",
		Logger: logger.Log,
	})

	return runServers(ctx, httpServer, grpcServer)
}

func setupAudit() *audit.AuditMessenger {
	observer := audit.NewAuditMessenger()

	if config.Options.AuditFilePath != "" {
		observer.RegisterObserver(&audit.FileObserver{
			FilePath: config.Options.AuditFilePath,
		})
	}

	if config.Options.AuditURL != "" {
		observer.RegisterObserver(&audit.HTTPObserver{
			URL: config.Options.AuditURL,
		})
	}

	return observer
}

func createHandlerChain(auditObserver *audit.AuditMessenger) http.Handler {
	return logger.WithLogging(
		handler.GzipMiddleware(
			service.AuthMiddleware(
				handler.MakeHandler(auditObserver),
			),
		),
	)
}

func runServers(ctx context.Context, httpServer *http.Server, grpcServer *grpcserver.Server) error {
	errCh := make(chan error, 2)

	go func() {
		logger.Log.Info("Starting HTTP server", zap.String("addr", httpServer.Addr))
		if err := startServer(httpServer); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	go func() {
		logger.Log.Info("Starting gRPC server", zap.String("port", ":3200"))
		if err := grpcServer.Start(); err != nil {
			errCh <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Log.Info("Shutting down servers...")
		return shutdownServers(httpServer, grpcServer)
	case err := <-errCh:
		return err
	}
}

func shutdownServers(httpServer *http.Server, grpcServer *grpcserver.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErr error

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		shutdownErr = fmt.Errorf("HTTP shutdown error: %w", err)
		logger.Log.Error("HTTP server shutdown failed", zap.Error(err))
	}

	grpcServer.Stop()

	return shutdownErr
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	if err := Run(ctx); err != nil {
		logger.Log.Fatal("Application failed", zap.Error(err))
	}
}

func startPprofServer() {
	logger.Log.Info("Starting pprof server on :6060")
	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
		logger.Log.Error("Pprof server failed", zap.Error(err))
	}
}

func createHTTPServer(handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:         config.Options.AppPort,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if config.Options.HTTPSEnabled {
		manager := &autocert.Manager{
			Cache:  autocert.DirCache("cache-dir"),
			Prompt: autocert.AcceptTOS,
		}
		server.TLSConfig = manager.TLSConfig()
	}

	return server
}

func startServer(srv *http.Server) error {
	if config.Options.HTTPSEnabled {
		return srv.ListenAndServeTLS("", "")
	}
	return srv.ListenAndServe()
}

func printBuildInfo() {
	fmt.Println("Build version:", buildVersion)
	fmt.Println("Build date:", buildDate)
	fmt.Println("Build commit:", buildCommit)
}
