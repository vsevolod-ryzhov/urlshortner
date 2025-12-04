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
	DatabaseDSN      string
	CookieSecret     string
	Environment      string `env:"ENVIRONMENT" envDefault:"development"`
}

func ParseFlags() {
	flag.StringVar(&Options.AppPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&Options.ShortenedBaseURL, "b", "localhost:8080", "The base url of shortened")
	flag.StringVar(&Options.FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&Options.StorageFilePath, "f", "/tmp/shortenerStorage", "Path to the file where the shortened URLs will be stored")
	flag.StringVar(&Options.DatabaseDSN, "d", "", "Database connection string")
	flag.StringVar(&Options.CookieSecret, "c", "", "Cookie secret")

	flag.Parse()

	if envRunAddr, exists := os.LookupEnv("SERVER_ADDRESS"); exists {
		Options.AppPort = envRunAddr
	}
	if envBaseURL, exists := os.LookupEnv("BASE_URL"); exists {
		Options.ShortenedBaseURL = envBaseURL
	}
	if envLogLevel, exists := os.LookupEnv("LOG_LEVEL"); exists {
		Options.FlagLogLevel = envLogLevel
	}
	if storageFilePath, exists := os.LookupEnv("FILE_STORAGE_PATH"); exists {
		Options.StorageFilePath = storageFilePath
	}
	if databaseDSN, exists := os.LookupEnv("DATABASE_DSN"); exists {
		Options.DatabaseDSN = databaseDSN
	}
	if cookieSecret, exists := os.LookupEnv("COOKIE_SECRET"); exists {
		Options.CookieSecret = cookieSecret
	}

	if Options.CookieSecret == "" {
		Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	}

	if len(Options.CookieSecret) != 16 && len(Options.CookieSecret) != 24 && len(Options.CookieSecret) != 32 {
		panic("COOKIE_SECRET must be 16, 24 or 32 bytes long")
	}
}
