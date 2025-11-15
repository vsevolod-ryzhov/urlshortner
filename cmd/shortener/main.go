package main

import (
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
)

func main() {
	config.ParseFlags()

	if err := logger.Initialize(config.Options.FlagLogLevel); err != nil {

		panic(err)
	}

	repo, repoErr := repository.NewRepository()
	if repoErr != nil {
		panic(repoErr)
	}
	if repo != nil {
		defer repo.Close()
		service.InitRepo(repo)
	}

	srv := &http.Server{
		Addr:         config.Options.AppPort,
		Handler:      logger.WithLogging(handler.GzipMiddleware(handler.MakeHandler())),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
