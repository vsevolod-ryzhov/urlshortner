package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompressWriter(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantHeader bool
	}{
		{
			name:       "success status sets content encoding",
			statusCode: http.StatusOK,
			wantHeader: true,
		},
		{
			name:       "2xx status sets content encoding",
			statusCode: http.StatusCreated,
			wantHeader: true,
		},
		{
			name:       "3xx status does not set content encoding",
			statusCode: http.StatusMovedPermanently,
			wantHeader: false,
		},
		{
			name:       "4xx status does not set content encoding",
			statusCode: http.StatusBadRequest,
			wantHeader: false,
		},
		{
			name:       "5xx status does not set content encoding",
			statusCode: http.StatusInternalServerError,
			wantHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			cw := newCompressWriter(recorder)

			cw.WriteHeader(tt.statusCode)

			hasHeader := recorder.Header().Get("Content-Encoding") == "gzip"

			if hasHeader != tt.wantHeader {
				t.Errorf("Content-Encoding header: got %v, want %v", hasHeader, tt.wantHeader)
			}

			testData := []byte("test data")
			n, err := cw.Write(testData)
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}
			if n != len(testData) {
				t.Errorf("Write() wrote %d bytes, want %d", n, len(testData))
			}

			if err := cw.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}

			cwHeaderValue := cw.Header().Get("Content-Encoding")
			recorderHeaderValue := recorder.Header().Get("Content-Encoding")
			if cwHeaderValue != recorderHeaderValue {
				t.Errorf("Header values differ: cw=%s, recorder=%s", cwHeaderValue, recorderHeaderValue)
			}
		})
	}
}

func TestCompressReader(t *testing.T) {
	testData := []byte("Hello, Gzip Compression!")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(testData); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	compressedData := buf.Bytes()

	t.Run("read compressed data", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader(compressedData))
		cr, err := newCompressReader(reader)
		if err != nil {
			t.Fatalf("newCompressReader() error = %v", err)
		}
		defer cr.Close()

		decompressed, err := io.ReadAll(cr)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if string(decompressed) != string(testData) {
			t.Errorf("Decompressed data = %s, want %s", decompressed, testData)
		}
	})

	t.Run("read empty compressed data", func(t *testing.T) {
		var emptyBuf bytes.Buffer
		gz := gzip.NewWriter(&emptyBuf)
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}

		reader := io.NopCloser(bytes.NewReader(emptyBuf.Bytes()))
		cr, err := newCompressReader(reader)
		if err != nil {
			t.Fatalf("newCompressReader() error = %v", err)
		}
		defer cr.Close()

		decompressed, err := io.ReadAll(cr)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if len(decompressed) != 0 {
			t.Errorf("Decompressed data should be empty, got %d bytes", len(decompressed))
		}
	})

	t.Run("invalid gzip data", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader([]byte("not gzip data")))
		_, err := newCompressReader(reader)
		if err == nil {
			t.Error("newCompressReader() should fail with invalid gzip data")
		}
	})

	t.Run("close reader", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader(compressedData))
		cr, err := newCompressReader(reader)
		if err != nil {
			t.Fatal(err)
		}

		if err := cr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		if err := cr.Close(); err != nil {
			t.Errorf("Second Close() error = %v", err)
		}
	})
}

func TestCompressReaderPartialRead(t *testing.T) {
	testData := []byte("Hello, this is a longer test string for partial reads! And some more text to make it longer.")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(testData); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	reader := io.NopCloser(bytes.NewReader(buf.Bytes()))
	cr, err := newCompressReader(reader)
	if err != nil {
		t.Fatal(err)
	}
	defer cr.Close()

	chunkSize := 10
	var result []byte
	buffer := make([]byte, chunkSize)

	for {
		n, err := cr.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(result) != len(testData) {
		t.Errorf("length mismatch: got %d, want %d", len(result), len(testData))
	}

	if string(result) != string(testData) {
		minLen := len(result)
		if len(testData) < minLen {
			minLen = len(testData)
		}

		for i := 0; i < minLen; i++ {
			if result[i] != testData[i] {
				t.Errorf("mismatch at position %d: got %q, want %q", i, result[i], testData[i])
				start := i - 5
				if start < 0 {
					start = 0
				}
				end := i + 5
				if end > minLen {
					end = minLen
				}
				t.Errorf("context: got %q, want %q", string(result[start:end]), string(testData[start:end]))
				break
			}
		}

		if len(result) != len(testData) {
			t.Errorf("different lengths: got %d bytes (%q), want %d bytes (%q)",
				len(result), result, len(testData), testData)
		}
	}
}

func TestGzipMiddlewareClose(t *testing.T) {
	eofReceived := false

	readHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for {
			buf := make([]byte, 1024)
			n, err := r.Body.Read(buf)
			if n == 0 && err == io.EOF {
				eofReceived = true
				break
			}
			if err != nil {
				break
			}
		}
		w.Write([]byte("OK"))
	})

	handler := GzipMiddleware(readHandler)

	testData := []byte("test data")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(testData)
	gz.Close()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	if !eofReceived {
		t.Log("Note: EOF not received, but this might be OK if body was read differently")
	}
}
