package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
)

type additionalContextKey string

const (
	additionalKey additionalContextKey = "additionalKey"
)

func setupTestConfig() {
	config.Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	config.Options.Environment = "development"
}

func TestAuthMiddleware_NewSession(t *testing.T) {
	setupTestConfig()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok, "UserID should be in context")
		assert.NotEmpty(t, userID, "UserID should not be empty")

		_, err := uuid.Parse(userID)
		assert.NoError(t, err, "UserID should be a valid UUID")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := AuthMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	require.Len(t, cookies, 1, "Should set one cookie")

	cookie := cookies[0]
	assert.Equal(t, cookieName, cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.Expires.After(time.Now()), "Cookie should have future expiration")
	assert.True(t, cookie.HttpOnly, "Cookie should be HttpOnly")
	assert.False(t, cookie.Secure, "Cookie should not be Secure in development")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)

	session, err := DecodeAndVerifyCookie(cookie.Value)
	require.NoError(t, err)
	assert.NotEmpty(t, session.UserID)
	assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)
}

func TestAuthMiddleware_ExistingSession(t *testing.T) {
	setupTestConfig()

	expectedUserID := uuid.New().String()
	session := &UserSession{
		UserID:    expectedUserID,
		CreatedAt: time.Now(),
	}

	cookieValue, err := encodeAndSignCookie(session)
	require.NoError(t, err)

	var capturedUserID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		require.True(t, ok)
		capturedUserID = userID
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: cookieValue,
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, expectedUserID, capturedUserID)

	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	assert.Len(t, cookies, 0, "Should not set new cookie for existing session")
}

func TestDecodeAndVerifyCookie_Invalid(t *testing.T) {
	setupTestConfig()

	testCases := []struct {
		name        string
		cookieValue string
		expectError bool
	}{
		{
			name:        "EmptyString",
			cookieValue: "",
			expectError: true,
		},
		{
			name:        "InvalidBase64",
			cookieValue: "not@base64#string",
			expectError: true,
		},
		{
			name: "WrongNumberOfParts",
			cookieValue: func() string {
				data := "only-one-part"
				encrypted, _ := encrypt([]byte(data))
				return base64.URLEncoding.EncodeToString(encrypted)
			}(),
			expectError: true,
		},
		{
			name: "InvalidTimestamp",
			cookieValue: func() string {
				data := "user-id|not-a-number"
				encrypted, _ := encrypt([]byte(data))
				return base64.URLEncoding.EncodeToString(encrypted)
			}(),
			expectError: true,
		},
		{
			name: "EmptyUserID",
			cookieValue: func() string {
				data := "|1234567890"
				encrypted, _ := encrypt([]byte(data))
				return base64.URLEncoding.EncodeToString(encrypted)
			}(),
			expectError: true,
		},
		{
			name: "TamperedData",
			cookieValue: func() string {
				session := &UserSession{
					UserID:    "test-user",
					CreatedAt: time.Now(),
				}
				encoded, _ := encodeAndSignCookie(session)
				return encoded[:len(encoded)-10] + "XXXXXXXXXX"
			}(),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session, err := DecodeAndVerifyCookie(tc.cookieValue)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	testCases := []struct {
		name          string
		contextValue  interface{}
		expectedID    string
		expectedFound bool
	}{
		{
			name:          "ValidUserID",
			contextValue:  "user-123",
			expectedID:    "user-123",
			expectedFound: true,
		},
		{
			name:          "EmptyUserID",
			contextValue:  "",
			expectedID:    "",
			expectedFound: true,
		},
		{
			name:          "WrongType",
			contextValue:  123,
			expectedID:    "",
			expectedFound: false,
		},
		{
			name:          "NilValue",
			contextValue:  nil,
			expectedID:    "",
			expectedFound: false,
		},
		{
			name:          "NoValue",
			contextValue:  nil,
			expectedID:    "",
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.contextValue != nil {
				ctx = context.WithValue(ctx, userIDKey, tc.contextValue)
			}

			userID, found := GetUserIDFromContext(ctx)

			assert.Equal(t, tc.expectedFound, found)
			if tc.expectedFound {
				assert.Equal(t, tc.expectedID, userID)
			}
		})
	}
}

func TestCookieAttributes(t *testing.T) {
	setupTestConfig()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]

	assert.Equal(t, cookieName, cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly, "Should be HttpOnly for security")
	assert.False(t, cookie.Secure, "Should not be Secure in development")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.True(t, cookie.Expires.After(time.Now()), "Should have future expiration")
	assert.True(t, cookie.Expires.Before(time.Now().Add(cookieExpires+time.Hour)), "Should not expire too far in future")
}

func TestAuthMiddleware_ProductionEnvironment(t *testing.T) {
	setupTestConfig()
	config.Options.Environment = "production"

	defer func() {
		config.Options.Environment = "development" // Восстанавливаем
	}()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()

	require.Len(t, cookies, 1)
	assert.True(t, cookies[0].Secure, "Cookie should be Secure in production")
}

func TestConcurrentSessions(t *testing.T) {
	setupTestConfig()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.NotEmpty(t, userID)
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(testHandler)

	concurrency := 10
	done := make(chan bool, concurrency)
	userIDs := make(map[string]bool)
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/test/%d", id), nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			cookies := result.Cookies()

			if len(cookies) > 0 {
				cookie := cookies[0]
				session, err := DecodeAndVerifyCookie(cookie.Value)
				if err == nil {
					mu.Lock()
					userIDs[session.UserID] = true
					mu.Unlock()
				}
			}

			done <- true
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}

	assert.Equal(t, concurrency, len(userIDs), "All sessions should have unique user IDs")
}

func TestInvalidSecretLength(t *testing.T) {
	originalSecret := config.Options.CookieSecret
	defer func() { config.Options.CookieSecret = originalSecret }()

	testCases := []struct {
		name       string
		secret     string
		shouldWork bool
	}{
		{"Empty", "", false},
		{"TooShort", "short", false},
		{"16Bytes", "16-bytes-key!!!!", true},                    // AES-128
		{"24Bytes", "24-bytes-long-key-here!!", true},            // AES-192
		{"32Bytes", "32-bytes-long-key-for-aes-256bit", true},    // AES-256
		{"33Bytes", "33-bytes-long-key-for-aes-256bit!!", false}, // too long
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config.Options.CookieSecret = tc.secret

			data := []byte("test-data")
			encrypted, err := encrypt(data)

			if tc.shouldWork {
				require.NoError(t, err)
				require.NotEmpty(t, encrypted)

				decrypted, errDecrypt := decrypt(encrypted)
				require.NoError(t, errDecrypt)
				assert.Equal(t, data, decrypted)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	setupTestConfig()

	calls := []string{}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
		w.WriteHeader(http.StatusOK)
	})

	anotherMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "another-middleware-before")
			next.ServeHTTP(w, r)
			calls = append(calls, "another-middleware-after")
		})
	}

	handler := anotherMiddleware(AuthMiddleware(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expectedCalls := []string{
		"another-middleware-before",
		"handler",
		"another-middleware-after",
	}
	assert.Equal(t, expectedCalls, calls)
}

func TestContextPropagation(t *testing.T) {
	setupTestConfig()

	valueAddingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), additionalKey, "additionalValue")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.NotEmpty(t, userID)

		additionalValue, ok := r.Context().Value(additionalKey).(string)
		assert.True(t, ok)
		assert.Equal(t, "additionalValue", additionalValue)

		w.WriteHeader(http.StatusOK)
	})

	handler := valueAddingMiddleware(AuthMiddleware(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_MissingCookieName(t *testing.T) {
	setupTestConfig()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "other_cookie",
		Value: "some_value",
	})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	result := rr.Result()
	defer result.Body.Close()
	cookies := result.Cookies()

	require.Len(t, cookies, 1)
	assert.Equal(t, cookieName, cookies[0].Name)
}
