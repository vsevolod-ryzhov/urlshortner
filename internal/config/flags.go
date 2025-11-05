package config

import (
	"flag"
	"os"
)

var Options struct {
	AppPort          string
	ShortenedBaseURL string
	FlagLogLevel     string
	StorageFilePath  string
}

func ParseFlags() {
	flag.StringVar(&Options.AppPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&Options.ShortenedBaseURL, "b", "localhost:8080", "The base url of shortened")
	flag.StringVar(&Options.FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&Options.StorageFilePath, "f", "/tmp/shortenerStorage", "Path to the file where the shortened URLs will be stored")
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		Options.AppPort = envRunAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		Options.ShortenedBaseURL = envBaseURL
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		Options.FlagLogLevel = envLogLevel
	}
	if storageFilePath := os.Getenv("FILE_STORAGE_PATH"); storageFilePath != "" {
		Options.StorageFilePath = storageFilePath
	}
}
