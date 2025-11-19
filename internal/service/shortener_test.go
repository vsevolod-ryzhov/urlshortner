package service

import (
	"strings"
	"testing"
)

func TestCreateShortURLAndGet(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "First URL",
			args: args{url: "https://ya.com"},
			want: "",
		},
		{
			name: "Second URL",
			args: args{url: "https://yandex.ru"},
			want: "",
		},
		{
			name: "URL with special characters",
			args: args{url: "https://someurl.com/somepath?query=param&other=value"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortened, _, _ := CreateShortURL(t.Context(), tt.args.url)

			if shortened == "" {
				t.Errorf("generateShortID(%q) returns empty string", tt.args.url)
			}

			restored, err := GetURL(shortened)

			if err != nil {
				t.Errorf("GetURL(%q) returns error: %v", shortened, err)
			}

			if tt.args.url != restored {
				t.Errorf("GetURL(%q) returns %q, want %q", shortened, restored, tt.args.url)
			}
		})
	}
}

func Test_generateShortID(t *testing.T) {
	type args struct {
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "First URL",
			args: args{originalURL: "https://ya.com"},
			want: "",
		},
		{
			name: "Second URL",
			args: args{originalURL: "https://yandex.ru"},
			want: "",
		},
		{
			name: "empty string",
			args: args{originalURL: ""},
			want: "",
		},
		{
			name: "URL with special characters",
			args: args{originalURL: "https://someurl.com/somepath?query=param&other=value"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateShortID(tt.args.originalURL)

			if result == "" {
				t.Errorf("generateShortID(%q) returns empty string", tt.args.originalURL)
			}

			if strings.HasSuffix(result, "=") {
				t.Errorf("generateShortID(%q) contains = suffix", tt.args.originalURL)
			}

			if !base64Save(result) {
				t.Errorf("generateShortID(%q) contains unsefe for base64 symbols", tt.args.originalURL)
			}
		})
	}
}

func base64Save(str string) bool {
	allowedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	for _, char := range str {
		if !strings.ContainsRune(allowedChars, char) {
			return false
		}
	}
	return true
}
