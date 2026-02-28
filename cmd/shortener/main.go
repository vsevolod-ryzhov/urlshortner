// Main entrypoint of Shortener app
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
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

func main() {
	sigInt := make(chan os.Signal, 1)
	signal.Notify(sigInt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	idleConnectionsClosed := make(chan struct{})

	config.ParseFlags()

	printBuildInfo()

	if err := logger.Initialize(config.Options.FlagLogLevel); err != nil {

		panic(err)
	}

	auditObserver := audit.NewAuditMessenger()

	if config.Options.AuditFilePath != "" {
		auditObserver.RegisterObserver(&audit.FileObserver{
			FilePath: config.Options.AuditFilePath,
		})
	}

	if config.Options.AuditURL != "" {
		auditObserver.RegisterObserver(&audit.HTTPObserver{
			URL: config.Options.AuditURL,
		})
	}

	repo, repoErr := repository.NewRepository()
	if repoErr != nil {
		logger.Log.Fatal("Failed to create repository", zap.Error(repoErr))
	}
	if repo != nil {
		defer repo.Close()
		service.InitRepo(repo)
		service.InitDeleteManager(repo, 3)
	}

	handlerChain := logger.WithLogging(
		handler.GzipMiddleware(
			service.AuthMiddleware(
				handler.MakeHandler(auditObserver),
			),
		),
	)

	go startPprofServer()

	srv := createServer(handlerChain)
	go func() {
		if err := startServer(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	go func() {
		<-sigInt
		logger.Log.Info("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Error("Server shutdown failed", zap.Error(err))
		}
		close(idleConnectionsClosed)
	}()

	<-idleConnectionsClosed
}

func startPprofServer() {
	logger.Log.Info("Starting pprof server on :6060")
	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
		logger.Log.Error("Pprof server failed", zap.Error(err))
	}
}

func createServer(handler http.Handler) *http.Server {
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
