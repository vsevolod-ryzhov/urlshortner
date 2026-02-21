package config

import (
	"flag"
	"os"
	"testing"
)

func setup() {
	flag.CommandLine = flag.NewFlagSet("", flag.ExitOnError)
	Options = struct {
		AppPort          string
		ShortenedBaseURL string
		FlagLogLevel     string
		StorageFilePath  string
		DatabaseDSN      string
		CookieSecret     string
		Environment      string `env:"ENVIRONMENT" envDefault:"development"`
		AuditFilePath    string
		AuditURL         string
		HTTPSEnabled     bool
	}{}
}

func TestParseFlags_DefaultValues(t *testing.T) {
	setup()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd"}
	ParseFlags()

	if Options.AppPort != "localhost:8080" {
		t.Errorf("Expected default AppPort 'localhost:8080', got '%s'", Options.AppPort)
	}
	if Options.ShortenedBaseURL != "localhost:8080" {
		t.Errorf("Expected default ShortenedBaseURL 'localhost:8080', got '%s'", Options.ShortenedBaseURL)
	}
	if Options.FlagLogLevel != "info" {
		t.Errorf("Expected default FlagLogLevel 'info', got '%s'", Options.FlagLogLevel)
	}
}

func TestParseFlags_EnvironmentVariablesOverride(t *testing.T) {
	setup()

	os.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	os.Setenv("BASE_URL", "https://short.example.com")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/test_storage")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")

	defer func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("BASE_URL")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("DATABASE_DSN")
	}()

	os.Args = []string{"cmd", "-a", "should-be-overridden:8080"}
	ParseFlags()

	if Options.AppPort != "127.0.0.1:9090" {
		t.Errorf("Expected AppPort from env '127.0.0.1:9090', got '%s'", Options.AppPort)
	}
	if Options.ShortenedBaseURL != "https://short.example.com" {
		t.Errorf("Expected ShortenedBaseURL from env 'https://short.example.com', got '%s'", Options.ShortenedBaseURL)
	}
	if Options.FlagLogLevel != "debug" {
		t.Errorf("Expected FlagLogLevel from env 'debug', got '%s'", Options.FlagLogLevel)
	}
}

func TestParseFlags_CommandLineFlags(t *testing.T) {
	setup()

	os.Args = []string{"cmd",
		"-a", "192.168.1.1:8080",
		"-b", "https://custom.url",
		"-l", "error",
		"-f", "/custom/path",
		"-d", "custom_dsn",
	}
	ParseFlags()

	if Options.AppPort != "192.168.1.1:8080" {
		t.Errorf("Expected AppPort from flag '192.168.1.1:8080', got '%s'", Options.AppPort)
	}
	if Options.ShortenedBaseURL != "https://custom.url" {
		t.Errorf("Expected ShortenedBaseURL from flag 'https://custom.url', got '%s'", Options.ShortenedBaseURL)
	}
}
