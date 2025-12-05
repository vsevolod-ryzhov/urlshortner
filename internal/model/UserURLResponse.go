package model

import "time"

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type UserURL struct {
	ShortURL    string
	OriginalURL string
	UserID      string
	CreatedAt   time.Time
}
