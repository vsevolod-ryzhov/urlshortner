// internal/grpcserver/server_test.go
package grpcserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func setupTestConfig() {
	config.Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	config.Options.Environment = "development"
}

func TestShortenURL(t *testing.T) {
	setupTestConfig()

	logger := zap.NewNop()
	server := NewShortenerServer(logger)

	tokenResp, err := server.GetSessionToken(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)

	session, err := service.DecodeAndVerifyCookie(tokenResp.GetToken())
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), UserIDKey, session.UserID)

	req := pb.URLShortenRequest_builder{
		Url: proto.String("https://example.com"),
	}.Build()

	resp, err := server.ShortenURL(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetResult())
}
