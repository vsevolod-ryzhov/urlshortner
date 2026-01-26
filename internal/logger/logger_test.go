package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitialize_Success(t *testing.T) {
	originalLog := Log
	defer func() { Log = originalLog }()

	levels := []string{
		"debug",
		"info",
		"warn",
		"error",
		"dpanic",
		"panic",
		"fatal",
	}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			err := Initialize(level)
			assert.NoError(t, err, "Initialize should not return error for level %s", level)
			assert.NotNil(t, Log, "Logger should be initialized")
			assert.NotEqual(t, zap.NewNop(), Log, "Logger should not be nop logger")
		})
	}
}

func TestInitialize_MultipleCalls(t *testing.T) {
	originalLog := Log
	defer func() { Log = originalLog }()

	err := Initialize("info")
	require.NoError(t, err)
	firstLogger := Log

	err = Initialize("debug")
	require.NoError(t, err)
	secondLogger := Log

	assert.NotEqual(t, firstLogger, secondLogger, "New logger should be created on re-initialization")
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	mockResponseWriter := httptest.NewRecorder()
	rd := &responseData{
		status: 0,
		size:   0,
	}

	lw := loggingResponseWriter{
		ResponseWriter: mockResponseWriter,
		responseData:   rd,
	}

	testData := []byte("Hello, World!")
	n, err := lw.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, len(testData), rd.size)
	assert.Equal(t, testData, mockResponseWriter.Body.Bytes())
}

func TestLoggingResponseWriter_WriteMultiple(t *testing.T) {
	mockResponseWriter := httptest.NewRecorder()
	rd := &responseData{
		status: 0,
		size:   0,
	}

	lw := loggingResponseWriter{
		ResponseWriter: mockResponseWriter,
		responseData:   rd,
	}

	chunks := [][]byte{
		[]byte("First chunk"),
		[]byte(" "),
		[]byte("Second chunk"),
		[]byte(" "),
		[]byte("Third chunk"),
	}

	totalSize := 0
	for _, chunk := range chunks {
		n, err := lw.Write(chunk)
		assert.NoError(t, err)
		totalSize += n
	}

	assert.Equal(t, totalSize, rd.size)
	assert.Equal(t, "First chunk Second chunk Third chunk", mockResponseWriter.Body.String())
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	mockResponseWriter := httptest.NewRecorder()
	rd := &responseData{
		status: 0,
		size:   0,
	}

	lw := loggingResponseWriter{
		ResponseWriter: mockResponseWriter,
		responseData:   rd,
	}

	statusCode := http.StatusNotFound
	lw.WriteHeader(statusCode)

	assert.Equal(t, statusCode, rd.status)
	assert.Equal(t, statusCode, mockResponseWriter.Code)
}

func TestLoggingResponseWriter_WriteAfterWriteHeader(t *testing.T) {
	mockResponseWriter := httptest.NewRecorder()
	rd := &responseData{
		status: 0,
		size:   0,
	}

	lw := loggingResponseWriter{
		ResponseWriter: mockResponseWriter,
		responseData:   rd,
	}

	lw.WriteHeader(http.StatusCreated)

	data := []byte("Created resource")
	n, err := lw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rd.size)
	assert.Equal(t, http.StatusCreated, rd.status)
	assert.Equal(t, http.StatusCreated, mockResponseWriter.Code)
}
