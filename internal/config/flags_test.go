package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
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
		TrustedSubnet    string
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
	os.Setenv("COOKIE_SECRET", "01234567890123456789012345678901")
	os.Setenv("AUDIT_FILE", "/tmp/test_audit.log")
	os.Setenv("AUDIT_URL", "https://audit.example.com/audit.log")
	os.Setenv("ENABLE_HTTPS", "true")

	defer func() {
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("BASE_URL")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("COOKIE_SECRET")
		os.Unsetenv("AUDIT_FILE")
		os.Unsetenv("AUDIT_URL")
		os.Unsetenv("ENABLE_HTTPS")
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
	if Options.CookieSecret != "01234567890123456789012345678901" {
		t.Errorf("Expected CookieSecret from env '01234567890123456789012345678901', got '%s'", Options.FlagLogLevel)
	}
	if Options.AuditFilePath != "/tmp/test_audit.log" {
		t.Errorf("Expected AuditFilePath from env '/tmp/test_audit.log', got '%s'", Options.FlagLogLevel)
	}
	if Options.AuditURL != "https://audit.example.com/audit.log" {
		t.Errorf("Expected CookieSecret from env 'https://audit.example.com/audit.log', got '%s'", Options.FlagLogLevel)
	}
	if Options.HTTPSEnabled != true {
		t.Errorf("Expected HTTPSEnabled eq true, got '%s'", Options.FlagLogLevel)
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

func TestParseFlags_CookieSecretPanic(t *testing.T) {
	tests := []struct {
		name         string
		cookieSecret string
		wantPanic    bool
	}{
		{
			name:         "valid 16 bytes",
			cookieSecret: "1234567890123456",
			wantPanic:    false,
		},
		{
			name:         "valid 24 bytes",
			cookieSecret: "123456789012345678901234",
			wantPanic:    false,
		},
		{
			name:         "valid 32 bytes",
			cookieSecret: "12345678901234567890123456789012",
			wantPanic:    false,
		},
		{
			name:         "invalid 8 bytes",
			cookieSecret: "12345678",
			wantPanic:    true,
		},
		{
			name:         "invalid 20 bytes",
			cookieSecret: "12345678901234567890",
			wantPanic:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup()

			if tt.cookieSecret != "" {
				os.Setenv("COOKIE_SECRET", tt.cookieSecret)
				defer os.Unsetenv("COOKIE_SECRET")
			}

			os.Args = []string{"cmd"}

			defer func() {
				r := recover()
				if tt.wantPanic && r == nil {
					t.Errorf("Expected panic for cookie secret length %d, but got none", len(tt.cookieSecret))
				}
				if !tt.wantPanic && r != nil {
					t.Errorf("Did not expect panic for cookie secret length %d, but got %v", len(tt.cookieSecret), r)
				}
			}()

			ParseFlags()
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tests := []struct {
		name          string
		configContent interface{}
		initialOpts   struct {
			appPort          string
			shortenedBaseURL string
			flagLogLevel     string
			storageFilePath  string
			databaseDSN      string
			cookieSecret     string
			environment      string
			auditFilePath    string
			auditURL         string
			httpsEnabled     bool
		}
		expectedOpts struct {
			appPort          string
			shortenedBaseURL string
			flagLogLevel     string
			storageFilePath  string
			databaseDSN      string
			cookieSecret     string
			environment      string
			auditFilePath    string
			auditURL         string
			httpsEnabled     bool
		}
		wantErr bool
	}{
		{
			name: "load all fields",
			configContent: ConfigurationFile{
				AppPort:          "config:8080",
				ShortenedBaseURL: "https://config.url",
				FlagLogLevel:     "debug",
				StorageFilePath:  "/config/storage",
				DatabaseDSN:      "config_dsn",
				CookieSecret:     "config_secret",
				Environment:      "production",
				AuditFilePath:    "/config/audit.log",
				AuditURL:         "https://config.audit",
				HTTPSEnabled:     true,
			},
			initialOpts: struct {
				appPort          string
				shortenedBaseURL string
				flagLogLevel     string
				storageFilePath  string
				databaseDSN      string
				cookieSecret     string
				environment      string
				auditFilePath    string
				auditURL         string
				httpsEnabled     bool
			}{
				appPort:          "localhost:8080",
				shortenedBaseURL: "localhost:8080",
				flagLogLevel:     "info",
				storageFilePath:  "/tmp/shortenerStorage",
				databaseDSN:      "",
				cookieSecret:     "",
				environment:      "",
				auditFilePath:    "",
				auditURL:         "",
				httpsEnabled:     false,
			},
			expectedOpts: struct {
				appPort          string
				shortenedBaseURL string
				flagLogLevel     string
				storageFilePath  string
				databaseDSN      string
				cookieSecret     string
				environment      string
				auditFilePath    string
				auditURL         string
				httpsEnabled     bool
			}{
				appPort:          "config:8080",
				shortenedBaseURL: "https://config.url",
				flagLogLevel:     "debug",
				storageFilePath:  "/config/storage",
				databaseDSN:      "config_dsn",
				cookieSecret:     "config_secret",
				environment:      "production",
				auditFilePath:    "/config/audit.log",
				auditURL:         "https://config.audit",
				httpsEnabled:     true,
			},
			wantErr: false,
		},
		{
			name: "dont override non-default values",
			configContent: ConfigurationFile{
				AppPort:          "config:8080",
				ShortenedBaseURL: "https://config.url",
				FlagLogLevel:     "debug",
				StorageFilePath:  "/config/storage",
			},
			initialOpts: struct {
				appPort          string
				shortenedBaseURL string
				flagLogLevel     string
				storageFilePath  string
				databaseDSN      string
				cookieSecret     string
				environment      string
				auditFilePath    string
				auditURL         string
				httpsEnabled     bool
			}{
				appPort:          "custom:8080",
				shortenedBaseURL: "https://custom.url",
				flagLogLevel:     "warn",
				storageFilePath:  "/custom/storage",
				databaseDSN:      "",
				cookieSecret:     "",
				environment:      "",
				auditFilePath:    "",
				auditURL:         "",
				httpsEnabled:     false,
			},
			expectedOpts: struct {
				appPort          string
				shortenedBaseURL string
				flagLogLevel     string
				storageFilePath  string
				databaseDSN      string
				cookieSecret     string
				environment      string
				auditFilePath    string
				auditURL         string
				httpsEnabled     bool
			}{
				appPort:          "custom:8080",
				shortenedBaseURL: "https://custom.url",
				flagLogLevel:     "warn",
				storageFilePath:  "/custom/storage",
				databaseDSN:      "",
				cookieSecret:     "",
				environment:      "",
				auditFilePath:    "",
				auditURL:         "",
				httpsEnabled:     false,
			},
			wantErr: false,
		},
		{
			name:          "file not found",
			configContent: nil,
			wantErr:       true,
		},
		{
			name:          "invalid json",
			configContent: "invalid json content",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			if tt.configContent != nil {
				var data []byte
				var err error

				switch content := tt.configContent.(type) {
				case ConfigurationFile:
					data, err = json.Marshal(content)
				case string:
					data = []byte(content)
				}

				if err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				configPath = filepath.Join(tmpDir, "nonexistent.json")
			}

			setup()
			Options.AppPort = tt.initialOpts.appPort
			Options.ShortenedBaseURL = tt.initialOpts.shortenedBaseURL
			Options.FlagLogLevel = tt.initialOpts.flagLogLevel
			Options.StorageFilePath = tt.initialOpts.storageFilePath
			Options.DatabaseDSN = tt.initialOpts.databaseDSN
			Options.CookieSecret = tt.initialOpts.cookieSecret
			Options.Environment = tt.initialOpts.environment
			Options.AuditFilePath = tt.initialOpts.auditFilePath
			Options.AuditURL = tt.initialOpts.auditURL
			Options.HTTPSEnabled = tt.initialOpts.httpsEnabled

			err := loadConfigFromFile(configPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if Options.AppPort != tt.expectedOpts.appPort {
					t.Errorf("AppPort = %v, want %v", Options.AppPort, tt.expectedOpts.appPort)
				}
				if Options.ShortenedBaseURL != tt.expectedOpts.shortenedBaseURL {
					t.Errorf("ShortenedBaseURL = %v, want %v", Options.ShortenedBaseURL, tt.expectedOpts.shortenedBaseURL)
				}
				if Options.FlagLogLevel != tt.expectedOpts.flagLogLevel {
					t.Errorf("FlagLogLevel = %v, want %v", Options.FlagLogLevel, tt.expectedOpts.flagLogLevel)
				}
				if Options.StorageFilePath != tt.expectedOpts.storageFilePath {
					t.Errorf("StorageFilePath = %v, want %v", Options.StorageFilePath, tt.expectedOpts.storageFilePath)
				}
				if Options.DatabaseDSN != tt.expectedOpts.databaseDSN {
					t.Errorf("DatabaseDSN = %v, want %v", Options.DatabaseDSN, tt.expectedOpts.databaseDSN)
				}
				if Options.CookieSecret != tt.expectedOpts.cookieSecret {
					t.Errorf("CookieSecret = %v, want %v", Options.CookieSecret, tt.expectedOpts.cookieSecret)
				}
				if Options.Environment != tt.expectedOpts.environment {
					t.Errorf("Environment = %v, want %v", Options.Environment, tt.expectedOpts.environment)
				}
				if Options.AuditFilePath != tt.expectedOpts.auditFilePath {
					t.Errorf("AuditFilePath = %v, want %v", Options.AuditFilePath, tt.expectedOpts.auditFilePath)
				}
				if Options.AuditURL != tt.expectedOpts.auditURL {
					t.Errorf("AuditURL = %v, want %v", Options.AuditURL, tt.expectedOpts.auditURL)
				}
				if Options.HTTPSEnabled != tt.expectedOpts.httpsEnabled {
					t.Errorf("HTTPSEnabled = %v, want %v", Options.HTTPSEnabled, tt.expectedOpts.httpsEnabled)
				}
			}
		})
	}
}
