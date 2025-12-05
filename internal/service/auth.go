package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"go.uber.org/zap"
)

type contextKey string

const (
	cookieName               = "user_session"
	cookieExpires            = 30 * 24 * time.Hour
	userIDKey     contextKey = "userID"
)

var (
	ErrInvalidCookie = errors.New("invalid cookie")
)

type UserSession struct {
	UserID    string
	CreatedAt time.Time
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(cookieName); err == nil {
			session, err := decodeAndVerifyCookie(cookie.Value)
			if err != nil {
				logger.Log.Debug("Cookie verifying error", zap.Error(err))
			}
			if err == nil && session.UserID != "" {
				ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		session, err := createNewSession(w)
		if err != nil {
			logger.Log.Error("Failed to create new session", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func createNewSession(w http.ResponseWriter) (*UserSession, error) {
	userID := uuid.New().String()
	session := &UserSession{
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	cookieValue, err := encodeAndSignCookie(session)
	if err != nil {
		return nil, err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		Expires:  time.Now().Add(cookieExpires),
		HttpOnly: true,
		Secure:   config.Options.Environment == "production",
		SameSite: http.SameSiteLaxMode,
	})

	return session, nil
}

func encodeAndSignCookie(session *UserSession) (string, error) {
	data := fmt.Sprintf("%s|%d", session.UserID, session.CreatedAt.Unix())

	encrypted, err := encrypt([]byte(data))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(encrypted), nil
}

func decodeAndVerifyCookie(cookieValue string) (*UserSession, error) {
	encrypted, err := base64.URLEncoding.DecodeString(cookieValue)
	if err != nil {
		return nil, ErrInvalidCookie
	}

	decrypted, err := decrypt(encrypted)
	if err != nil {
		return nil, ErrInvalidCookie
	}

	parts := strings.Split(string(decrypted), "|")
	if len(parts) != 2 {
		return nil, ErrInvalidCookie
	}

	userID := parts[0]
	createdAtStr := parts[1]

	createdAt, err := strconv.ParseInt(createdAtStr, 10, 64)
	if err != nil {
		return nil, ErrInvalidCookie
	}

	if userID == "" {
		return nil, ErrInvalidCookie
	}

	return &UserSession{
		UserID:    userID,
		CreatedAt: time.Unix(createdAt, 0),
	}, nil
}

func encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(config.Options.CookieSecret))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(config.Options.CookieSecret))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
