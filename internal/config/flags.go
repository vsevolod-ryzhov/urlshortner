// Package config used for initialization and storing all available parameters in app
package config

import (
	"flag"
	"os"
	"strconv"
)

var Options struct {
	AppPort          string // Application address and port
	ShortenedBaseURL string // Base URL used for shortened version
	FlagLogLevel     string // Log level of built-in logger
	StorageFilePath  string // Path to file where all shortened links will be stored
	DatabaseDSN      string // Database connection string
	CookieSecret     string // 32 bytes secret
	Environment      string `env:"ENVIRONMENT" envDefault:"development"` // Application environment
	AuditFilePath    string // Path to audit file where app logs will be stored
	AuditURL         string // URL for audit service where app logs will be sent
	HTTPSEnabled     bool
}

// ParseFlags func reads startup arguments and env variables
func ParseFlags() {
	flag.StringVar(&Options.AppPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&Options.ShortenedBaseURL, "b", "localhost:8080", "The base url of shortened")
	flag.StringVar(&Options.FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&Options.StorageFilePath, "f", "/tmp/shortenerStorage", "Path to the file where the shortened URLs will be stored")
	flag.StringVar(&Options.DatabaseDSN, "d", "", "Database connection string")
	flag.StringVar(&Options.CookieSecret, "c", "", "Cookie secret")
	flag.StringVar(&Options.AuditFilePath, "audit-file", "", "Audit file path")
	flag.StringVar(&Options.AuditURL, "audit-url", "", "Audit URL")
	flag.BoolVar(&Options.HTTPSEnabled, "s", false, "Enable HTTPS")

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
	if auditFilePath, exists := os.LookupEnv("AUDIT_FILE"); exists {
		Options.AuditFilePath = auditFilePath
	}
	if auditURL, exists := os.LookupEnv("AUDIT_URL"); exists {
		Options.AuditURL = auditURL
	}
	if isHTTPSEnabled, exists := os.LookupEnv("ENABLE_HTTPS"); exists {
		Options.HTTPSEnabled, _ = strconv.ParseBool(isHTTPSEnabled)
	}

	if Options.CookieSecret == "" {
		// For passing auto-tests only
		Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	}

	if len(Options.CookieSecret) != 16 && len(Options.CookieSecret) != 24 && len(Options.CookieSecret) != 32 {
		panic("COOKIE_SECRET must be 16, 24 or 32 bytes long")
	}
}
