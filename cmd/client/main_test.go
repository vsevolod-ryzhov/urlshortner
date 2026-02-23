package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	oldStdin := os.Stdin
	oldStdout := os.Stdout

	code := m.Run()

	os.Stdin = oldStdin
	os.Stdout = oldStdout

	os.Exit(code)
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	return string(out)
}

func withStdin(input string, f func()) {
	oldStdin := os.Stdin

	r, w, _ := os.Pipe()
	_, _ = w.Write([]byte(input))
	w.Close()
	os.Stdin = r

	f()

	os.Stdin = oldStdin
}

func TestClient_SuccessfulRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", string(body))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/abc123"))
	}))
	defer server.Close()

	oldEndpoint := endpoint
	endpoint = server.URL + "/"
	defer func() { endpoint = oldEndpoint }()

	output := captureOutput(func() {
		withStdin("https://example.com\n", func() {
			main()
		})
	})

	assert.Contains(t, output, "Enter URL to be shortened:")
	assert.Contains(t, output, "Response status  201 Created")
	assert.Contains(t, output, "http://localhost:8080/abc123")
}

func TestClient_EmptyInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	oldEndpoint := endpoint
	endpoint = server.URL + "/"
	defer func() { endpoint = oldEndpoint }()

	output := captureOutput(func() {
		withStdin("\n", func() {
			main()
		})
	})

	assert.Contains(t, output, "Enter URL to be shortened:")
	assert.Contains(t, output, "Response status  400 Bad Request")
}

func TestClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	oldEndpoint := endpoint
	endpoint = server.URL + "/"
	defer func() { endpoint = oldEndpoint }()

	output := captureOutput(func() {
		withStdin("https://example.com\n", func() {
			main()
		})
	})

	assert.Contains(t, output, "Enter URL to be shortened:")
	assert.Contains(t, output, "Response status  500 Internal Server Error")
}

func TestClient_InvalidURL(t *testing.T) {
	oldEndpoint := endpoint
	endpoint = "http://invalid-url-that-does-not-exist:8080/"
	defer func() { endpoint = oldEndpoint }()

	output := captureOutput(func() {
		withStdin("https://example.com\n", func() {
			main()
		})
	})

	assert.Contains(t, output, "Enter URL to be shortened:")
	assert.Contains(t, output, "no such host")
}

func TestClient_ReadStringError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	oldEndpoint := endpoint
	endpoint = server.URL + "/"
	defer func() { endpoint = oldEndpoint }()

	output := captureOutput(func() {
		withStdin("", func() {
			main()
		})
	})

	assert.Contains(t, output, "Enter URL to be shortened:")
	assert.Contains(t, output, "EOF")
}

func TestClient_DirectRestyCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "https://test.com", string(body))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/test123"))
	}))
	defer server.Close()

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		SetBody("https://test.com").
		Post(server.URL + "/")

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode())
	assert.Equal(t, "http://localhost:8080/test123", string(resp.Body()))
}

func TestClient_InputWithNewline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with newline",
			input:    "https://example.com\n",
			expected: "https://example.com",
		},
		{
			name:     "with windows newline",
			input:    "https://example.com\r\n",
			expected: "https://example.com",
		},
		{
			name:     "without newline",
			input:    "https://example.com",
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				actual := string(body)
				actual = strings.TrimSuffix(actual, "\r")

				assert.Equal(t, tt.expected, actual)
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			oldEndpoint := endpoint
			endpoint = server.URL + "/"
			defer func() { endpoint = oldEndpoint }()

			_ = captureOutput(func() {
				withStdin(tt.input, func() {
					main()
				})
			})
		})
	}
}

func TestClient_NewlineTrimming(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://example.com\n",
			expected: "https://example.com",
		},
		{
			input:    "https://example.com\r\n",
			expected: "https://example.com",
		},
		{
			input:    "https://example.com\r",
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			long := tt.input
			long = strings.TrimSuffix(long, "\n")
			long = strings.TrimSuffix(long, "\r")

			assert.Equal(t, tt.expected, long)
		})
	}
}
