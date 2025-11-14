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
	//if err := service.InitStorage(); err != nil {
	//	fmt.Printf("Error loading data storage file: %v\n", err)
	//}

	if err := logger.Initialize(config.Options.FlagLogLevel); err != nil {

		panic(err)
	}

	//if len(config.Options.DatabaseDSN) > 0 {
	//	var dbErr error
	//	repository.DB, dbErr = sql.Open("pgx", config.Options.DatabaseDSN)
	//	if dbErr != nil {
	//		panic(dbErr)
	//	}
	//	defer repository.DB.Close()
	//}

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
