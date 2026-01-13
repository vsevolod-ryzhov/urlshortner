package main

import (
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
)

func main() {
	config.ParseFlags()

	if err := logger.Initialize(config.Options.FlagLogLevel); err != nil {

		panic(err)
	}

	auditObserver := &audit.AuditMessenger{}

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

	srv := &http.Server{
		Addr:         config.Options.AppPort,
		Handler:      handlerChain,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
