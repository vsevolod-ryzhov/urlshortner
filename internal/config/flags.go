// Package config used for initialization and storing all available parameters in app
package config

import (
	"encoding/json"
	"flag"
	"fmt"
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
	HTTPSEnabled     bool   // Enables SSL support
	TrustedSubnet    string // For this subnet /api/internal/stats endpoint will be enabled
}

type ConfigurationFile struct {
	AppPort          string `json:"app_port"`
	ShortenedBaseURL string `json:"shortened_base_url"`
	FlagLogLevel     string `json:"log_level"`
	StorageFilePath  string `json:"storage_file_path"`
	DatabaseDSN      string `json:"database_dsn"`
	CookieSecret     string `json:"cookie_secret"`
	Environment      string `json:"environment"`
	AuditFilePath    string `json:"audit_file_path"`
	AuditURL         string `json:"audit_url"`
	HTTPSEnabled     bool   `json:"https_enabled"`
	TrustedSubnet    string `json:"trusted_subnet"`
}

// ParseFlags func reads startup arguments and env variables
func ParseFlags() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.StringVar(&Options.AppPort, "a", "localhost:8080", "The address to bind the app to")
	flag.StringVar(&Options.ShortenedBaseURL, "b", "localhost:8080", "The base url of shortened")
	flag.StringVar(&Options.FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&Options.StorageFilePath, "f", "/tmp/shortenerStorage", "Path to the file where the shortened URLs will be stored")
	flag.StringVar(&Options.DatabaseDSN, "d", "", "Database connection string")
	flag.StringVar(&Options.CookieSecret, "c", "", "Cookie secret")
	flag.StringVar(&Options.AuditFilePath, "audit-file", "", "Audit file path")
	flag.StringVar(&Options.AuditURL, "audit-url", "", "Audit URL")
	flag.BoolVar(&Options.HTTPSEnabled, "s", false, "Enable HTTPS")
	flag.StringVar(&Options.TrustedSubnet, "t", "", "Trusted subnet")

	flag.Parse()

	if configFile == "" {
		configFile = os.Getenv("CONFIG")
	}

	if configFile != "" {
		if err := loadConfigFromFile(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config file %s: %v\n", configFile, err)
		}
	}

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

	if trustedSubnet, exists := os.LookupEnv("TRUSTED_SUBNET"); exists {
		Options.TrustedSubnet = trustedSubnet
	}

	if Options.CookieSecret == "" {
		// For passing auto-tests only
		Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	}

	if len(Options.CookieSecret) != 16 && len(Options.CookieSecret) != 24 && len(Options.CookieSecret) != 32 {
		panic("COOKIE_SECRET must be 16, 24 or 32 bytes long")
	}
}

func loadConfigFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	var fileConfig ConfigurationFile
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	if Options.AppPort == "localhost:8080" && fileConfig.AppPort != "" {
		Options.AppPort = fileConfig.AppPort
	}

	if Options.ShortenedBaseURL == "localhost:8080" && fileConfig.ShortenedBaseURL != "" {
		Options.ShortenedBaseURL = fileConfig.ShortenedBaseURL
	}

	if Options.FlagLogLevel == "info" && fileConfig.FlagLogLevel != "" {
		Options.FlagLogLevel = fileConfig.FlagLogLevel
	}

	if Options.StorageFilePath == "/tmp/shortenerStorage" && fileConfig.StorageFilePath != "" {
		Options.StorageFilePath = fileConfig.StorageFilePath
	}

	if Options.DatabaseDSN == "" && fileConfig.DatabaseDSN != "" {
		Options.DatabaseDSN = fileConfig.DatabaseDSN
	}

	if Options.CookieSecret == "" && fileConfig.CookieSecret != "" {
		Options.CookieSecret = fileConfig.CookieSecret
	}

	if Options.Environment == "" && fileConfig.Environment != "" {
		Options.Environment = fileConfig.Environment
	}

	if Options.AuditFilePath == "" && fileConfig.AuditFilePath != "" {
		Options.AuditFilePath = fileConfig.AuditFilePath
	}

	if Options.AuditURL == "" && fileConfig.AuditURL != "" {
		Options.AuditURL = fileConfig.AuditURL
	}

	if !isFlagPassed("s") && fileConfig.HTTPSEnabled {
		Options.HTTPSEnabled = fileConfig.HTTPSEnabled
	}

	if Options.TrustedSubnet == "" && fileConfig.TrustedSubnet != "" {
		Options.TrustedSubnet = fileConfig.TrustedSubnet
	}

	return nil
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
