package service

import (
	"context"
	"fmt"
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
			shortened, _, _ := CreateShortURL(t.Context(), tt.args.url, "1")

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

func ExampleCreateShortURL() {
	shortened1, _, _ := CreateShortURL(context.Background(), "https://ya.ru", "1")
	fmt.Println(shortened1)

	shortened2, _, _ := CreateShortURL(context.Background(), "https://ya.com", "1")
	fmt.Println(shortened2)

	shortened3, _, _ := CreateShortURL(context.Background(), "https://ya.net", "1")
	fmt.Println(shortened3)

	// Output:
	// fpCk-cMLTn4
	// ZOiLcK9R_6I
	// gUbMZ5KfCnQ
}

func ExampleGetURL() {
	shortened1, _, _ := CreateShortURL(context.Background(), "https://ya.ru", "1")
	originalURL1, _ := GetURL(shortened1)
	fmt.Println(originalURL1)

	shortened2, _, _ := CreateShortURL(context.Background(), "https://ya.com", "1")
	originalURL2, _ := GetURL(shortened2)
	fmt.Println(originalURL2)

	shortened3, _, _ := CreateShortURL(context.Background(), "https://ya.net", "1")
	originalURL3, _ := GetURL(shortened3)
	fmt.Println(originalURL3)

	// Output:
	// https://ya.ru
	// https://ya.com
	// https://ya.net
}
