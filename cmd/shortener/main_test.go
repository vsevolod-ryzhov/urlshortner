package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/grpcserver"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestMain(m *testing.M) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	code := m.Run()
	os.Exit(code)
}

func TestBuildInfo(t *testing.T) {
	oldStdout := os.Stdout
	oldBuildVersion := buildVersion
	oldBuildDate := buildDate
	oldBuildCommit := buildCommit

	defer func() {
		os.Stdout = oldStdout
		buildVersion = oldBuildVersion
		buildDate = oldBuildDate
		buildCommit = oldBuildCommit
	}()

	buildVersion = "v1.0.0"
	buildDate = "2024-01-01"
	buildCommit = "abc123"

	r, w, _ := os.Pipe()
	os.Stdout = w

	printBuildInfo()

	w.Close()
	out, _ := io.ReadAll(r)
	output := string(out)

	assert.Contains(t, output, "Build version: v1.0.0")
	assert.Contains(t, output, "Build date: 2024-01-01")
	assert.Contains(t, output, "Build commit: abc123")
}

func TestMainInitialization(t *testing.T) {
	oldArgs := os.Args
	oldEnv := os.Getenv("SERVER_ADDRESS")
	defer func() {
		os.Args = oldArgs
		os.Setenv("SERVER_ADDRESS", oldEnv)
	}()

	os.Args = []string{"cmd", "-a", "localhost:9090", "-l", "debug"}
	os.Setenv("SERVER_ADDRESS", "")

	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config.Options.AuditFilePath = tmpFile.Name()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Test panicked: %v", r)
		}
	}()

	done := make(chan bool)
	go func() {
		config.ParseFlags()
		err := logger.Initialize(config.Options.FlagLogLevel)
		assert.NoError(t, err)

		auditObserver := audit.NewAuditMessenger()
		assert.NotNil(t, auditObserver)

		repo, repoErr := repository.NewRepository()
		assert.NoError(t, repoErr)
		if repo != nil {
			repo.Close()
		}
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Initialization timeout")
	}
}

func TestCreateServer(t *testing.T) {
	tests := []struct {
		name         string
		httpsEnabled bool
		appPort      string
	}{
		{
			name:         "HTTP server",
			httpsEnabled: false,
			appPort:      "localhost:8080",
		},
		{
			name:         "HTTPS server",
			httpsEnabled: true,
			appPort:      "localhost:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHTTPS := config.Options.HTTPSEnabled
			oldPort := config.Options.AppPort
			defer func() {
				config.Options.HTTPSEnabled = oldHTTPS
				config.Options.AppPort = oldPort
			}()

			config.Options.HTTPSEnabled = tt.httpsEnabled
			config.Options.AppPort = tt.appPort

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			server := createHTTPServer(handler)

			assert.Equal(t, tt.appPort, server.Addr)
			assert.NotNil(t, server.Handler)
			assert.Equal(t, 5*time.Second, server.ReadTimeout)
			assert.Equal(t, 10*time.Second, server.WriteTimeout)

			if tt.httpsEnabled {
				assert.NotNil(t, server.TLSConfig)
			} else {
				assert.Nil(t, server.TLSConfig)
			}
		})
	}
}

func TestStartServer(t *testing.T) {
	tests := []struct {
		name         string
		httpsEnabled bool
		wantErr      bool
	}{
		{
			name:         "Start HTTP server",
			httpsEnabled: false,
			wantErr:      false,
		},
		{
			name:         "Start HTTPS server",
			httpsEnabled: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHTTPS := config.Options.HTTPSEnabled
			defer func() { config.Options.HTTPSEnabled = oldHTTPS }()

			config.Options.HTTPSEnabled = tt.httpsEnabled

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			server := &http.Server{
				Addr:    "localhost:0",
				Handler: handler,
			}

			errChan := make(chan error, 1)
			go func() {
				errChan <- startServer(server)
			}()

			time.Sleep(100 * time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			server.Shutdown(ctx)

			err := <-errChan
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.ErrorIs(t, err, http.ErrServerClosed)
			}
		})
	}
}

func TestStartPprofServer(t *testing.T) {
	go startPprofServer()

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	resp, err := client.Get("http://localhost:6060/debug/pprof/")
	if err != nil {
		t.Logf("Pprof server not available: %v", err)
	} else {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

func TestGracefulShutdown(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    "localhost:8081",
		Handler: handler,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	time.Sleep(100 * time.Millisecond)

	go func() {
		resp, err := http.Get("http://localhost:8081/")
		if err == nil {
			resp.Body.Close()
		}
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	startTime := time.Now()
	err := server.Shutdown(ctx)
	shutdownTime := time.Since(startTime)

	wg.Wait()

	assert.GreaterOrEqual(t, shutdownTime, 400*time.Millisecond,
		"Shutdown should wait for ongoing request to complete")
	assert.NoError(t, err)
}

func TestLoggerInitialization(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		wantErr  bool
	}{
		{
			name:     "Valid log level",
			logLevel: "debug",
			wantErr:  false,
		},
		{
			name:     "Invalid log level",
			logLevel: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldLevel := config.Options.FlagLogLevel
			defer func() { config.Options.FlagLogLevel = oldLevel }()

			config.Options.FlagLogLevel = tt.logLevel

			err := logger.Initialize(tt.logLevel)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger.Log)
			}
		})
	}
}

func TestAuditSetup(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var msg audit.AuditMessage
		err = json.Unmarshal(body, &msg)
		assert.NoError(t, err)
		assert.Equal(t, "test message", msg.Data)

		w.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()

	tests := []struct {
		name              string
		auditFile         string
		auditURL          string
		expectedObservers int
	}{
		{
			name:              "No observers",
			auditFile:         "",
			auditURL:          "",
			expectedObservers: 0,
		},
		{
			name:              "File observer only",
			auditFile:         tmpFile.Name(),
			auditURL:          "",
			expectedObservers: 1,
		},
		{
			name:              "HTTP observer only",
			auditFile:         "",
			auditURL:          httpServer.URL,
			expectedObservers: 1,
		},
		{
			name:              "Both observers",
			auditFile:         tmpFile.Name(),
			auditURL:          httpServer.URL,
			expectedObservers: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFile := config.Options.AuditFilePath
			oldURL := config.Options.AuditURL
			defer func() {
				config.Options.AuditFilePath = oldFile
				config.Options.AuditURL = oldURL
			}()

			config.Options.AuditFilePath = tt.auditFile
			config.Options.AuditURL = tt.auditURL

			observer := audit.NewAuditMessenger()
			assert.NotNil(t, observer)

			if tt.auditFile != "" {
				observer.RegisterObserver(&audit.FileObserver{
					FilePath: tt.auditFile,
				})
			}

			if tt.auditURL != "" {
				observer.RegisterObserver(&audit.HTTPObserver{
					URL: tt.auditURL,
				})
			}

			testMsg := audit.AuditMessage{Data: "test message"}
			observer.Audit(testMsg)

			time.Sleep(100 * time.Millisecond)

			if tt.auditFile != "" {
				content, err := os.ReadFile(tt.auditFile)
				assert.NoError(t, err)
				assert.Contains(t, string(content), "test message")
			}
		})
	}
}

func TestHandlerChain(t *testing.T) {
	auditObserver := audit.NewAuditMessenger()

	handlerChain := logger.WithLogging(
		handler.GzipMiddleware(
			service.AuthMiddleware(
				handler.MakeHandler(auditObserver),
			),
		),
	)

	assert.NotNil(t, handlerChain)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handlerChain.ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()

	assert.NotNil(t, result)
}

func TestSignalHandling(t *testing.T) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		time.Sleep(100 * time.Millisecond)
		sigChan <- syscall.SIGINT
	}()

	select {
	case <-sigChan:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Signal not received")
	}
}

func TestRun_WithAudit(t *testing.T) {
	resetFlags()
	tmpFile, err := os.CreateTemp("", "audit_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	oldArgs := os.Args
	oldAuditFile := config.Options.AuditFilePath
	oldPort := config.Options.AppPort

	defer func() {
		os.Args = oldArgs
		config.Options.AuditFilePath = oldAuditFile
		config.Options.AppPort = oldPort
	}()

	os.Args = []string{"cmd", "-a", "localhost:0", "-l", "debug"}
	config.Options.AuditFilePath = tmpFile.Name()
	config.Options.AppPort = "localhost:0"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
		content, _ := os.ReadFile(tmpFile.Name())
		t.Logf("Audit content: %s", string(content))
	case <-time.After(3 * time.Second):
		t.Fatal("Run timed out")
	}
}

func TestRun_Success(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	oldEnv := os.Getenv("SERVER_ADDRESS")
	oldPort := config.Options.AppPort
	oldAuditFile := config.Options.AuditFilePath
	oldAuditURL := config.Options.AuditURL

	defer func() {
		os.Args = oldArgs
		os.Setenv("SERVER_ADDRESS", oldEnv)
		config.Options.AppPort = oldPort
		config.Options.AuditFilePath = oldAuditFile
		config.Options.AuditURL = oldAuditURL
	}()

	os.Args = []string{"cmd", "-a", "localhost:0", "-l", "debug"}
	os.Setenv("SERVER_ADDRESS", "")
	config.Options.AuditFilePath = ""
	config.Options.AuditURL = ""

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "Run should complete without error")
	case <-time.After(3 * time.Second):
		t.Fatal("Run timed out")
	}
}

func TestRun_InvalidLoggerLevel(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	oldLevel := config.Options.FlagLogLevel

	defer func() {
		os.Args = oldArgs
		config.Options.FlagLogLevel = oldLevel
	}()

	os.Args = []string{"cmd", "-l", "invalid_level"}

	ctx := context.Background()
	err := Run(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logger initialization failed")
}

func TestShutdownServers(t *testing.T) {
	resetFlags()
	httpServer := &http.Server{
		Addr:    "localhost:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	grpcServer := grpcserver.NewServer(grpcserver.ServerConfig{
		Port:   ":0",
		Logger: zap.NewNop(),
	})

	go func() {
		_ = httpServer.ListenAndServe()
	}()
	time.Sleep(100 * time.Millisecond)

	err := shutdownServers(httpServer, grpcServer)
	assert.NoError(t, err)
}

func TestSetupAudit(t *testing.T) {
	resetFlags()
	tests := []struct {
		name        string
		filePath    string
		url         string
		expectError bool
	}{
		{
			name:        "no observers",
			filePath:    "",
			url:         "",
			expectError: false,
		},
		{
			name:        "file observer",
			filePath:    "/tmp/test.log",
			url:         "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFile := config.Options.AuditFilePath
			oldURL := config.Options.AuditURL

			defer func() {
				config.Options.AuditFilePath = oldFile
				config.Options.AuditURL = oldURL
			}()

			config.Options.AuditFilePath = tt.filePath
			config.Options.AuditURL = tt.url

			observer := setupAudit()
			assert.NotNil(t, observer)
		})
	}
}

func TestRun_RepositoryError(t *testing.T) {
	resetFlags()
	oldDSN := os.Getenv("DATABASE_DSN")
	defer os.Setenv("DATABASE_DSN", oldDSN)

	os.Setenv("DATABASE_DSN", "postgres://invalid:invalid@localhost:9999/invalid")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-d", "postgres://invalid:invalid@localhost:9999/invalid"}

	ctx := context.Background()
	err := Run(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create repository")
}

func TestRun_WithCancel(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-a", "localhost:0", "-l", "debug"}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "Run should exit cleanly on cancel")
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}
