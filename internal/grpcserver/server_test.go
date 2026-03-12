package grpcserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func setupTestConfig() {
	config.Options.CookieSecret = "JVaB8G2m7tu9XSzQjLU3Vxf5X4uU3apR"
	config.Options.Environment = "development"
	config.Options.ShortenedBaseURL = "http://localhost:8080"
}

func TestMain(m *testing.M) {
	setupTestConfig()
	m.Run()
}

func TestShortenURL(t *testing.T) {
	logger := zap.NewNop()
	server := NewShortenerServer(logger)

	tokenResp, err := server.GetSessionToken(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)

	session, err := service.DecodeAndVerifyCookie(tokenResp.GetToken())
	require.NoError(t, err)

	t.Run("successful shorten", func(t *testing.T) {
		ctx := withUserID(context.Background(), session.UserID)

		req := pb.URLShortenRequest_builder{
			Url: proto.String("https://example.com"),
		}.Build()

		resp, err := server.ShortenURL(ctx, req)

		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetResult())
	})

	t.Run("missing user ID", func(t *testing.T) {
		ctx := context.Background()

		req := pb.URLShortenRequest_builder{
			Url: proto.String("https://example.com"),
		}.Build()

		resp, err := server.ShortenURL(ctx, req)

		require.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}

func TestExpandURL(t *testing.T) {
	logger := zap.NewNop()
	server := NewShortenerServer(logger)

	tokenResp, err := server.GetSessionToken(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)

	session, err := service.DecodeAndVerifyCookie(tokenResp.GetToken())
	require.NoError(t, err)

	ctx := withUserID(context.Background(), session.UserID)

	shortenReq := pb.URLShortenRequest_builder{
		Url: proto.String("https://example.com"),
	}.Build()

	shortenResp, err := server.ShortenURL(ctx, shortenReq)
	require.NoError(t, err)
	shortURL := shortenResp.GetResult()

	t.Run("successful expand", func(t *testing.T) {
		req := pb.URLExpandRequest_builder{
			Id: proto.String(shortURL),
		}.Build()

		resp, err := server.ExpandURL(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "https://example.com", resp.GetResult())
	})

	t.Run("missing user ID", func(t *testing.T) {
		unauthCtx := context.Background()

		req := pb.URLExpandRequest_builder{
			Id: proto.String(shortURL),
		}.Build()

		resp, err := server.ExpandURL(unauthCtx, req)

		require.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("non-existent ID", func(t *testing.T) {
		req := pb.URLExpandRequest_builder{
			Id: proto.String("nonexistent123"),
		}.Build()

		resp, err := server.ExpandURL(ctx, req)

		require.Error(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("empty ID", func(t *testing.T) {
		req := pb.URLExpandRequest_builder{
			Id: proto.String(""),
		}.Build()

		resp, err := server.ExpandURL(ctx, req)

		require.Error(t, err)
		assert.NotNil(t, resp)
	})
}

func TestGetSessionToken(t *testing.T) {
	setupTestConfig()
	logger := zap.NewNop()
	server := NewShortenerServer(logger)

	t.Run("successful token creation", func(t *testing.T) {
		resp, err := server.GetSessionToken(context.Background(), &emptypb.Empty{})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetToken())

		session, err := service.DecodeAndVerifyCookie(resp.GetToken())
		require.NoError(t, err)
		assert.NotEmpty(t, session.UserID)
		assert.False(t, session.CreatedAt.IsZero())
	})

	t.Run("multiple tokens are unique", func(t *testing.T) {
		resp1, err := server.GetSessionToken(context.Background(), &emptypb.Empty{})
		require.NoError(t, err)

		resp2, err := server.GetSessionToken(context.Background(), &emptypb.Empty{})
		require.NoError(t, err)

		assert.NotEqual(t, resp1.GetToken(), resp2.GetToken())
	})
}

func TestNewServer(t *testing.T) {
	logger := zap.NewNop()
	config := ServerConfig{
		Port:   ":0",
		Logger: logger,
	}

	server := NewServer(config)

	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
	assert.Equal(t, logger, server.logger)
	assert.Nil(t, server.grpcServer)
}

func TestServerStartStop(t *testing.T) {
	testLogger := zaptest.NewLogger(t)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	serverConfig := ServerConfig{
		Port:   listener.Addr().String(),
		Logger: testLogger,
	}

	server := NewServer(serverConfig)

	errChan := make(chan error, 1)

	started := make(chan struct{})

	go func() {
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(server.authInterceptor),
		)

		shortenerServer := NewShortenerServer(testLogger)
		pb.RegisterShortenerServiceServer(grpcServer, shortenerServer)

		server.grpcServer = grpcServer

		close(started)

		errChan <- grpcServer.Serve(listener)
	}()

	<-started

	time.Sleep(100 * time.Millisecond)

	server.Stop()

	select {
	case err := <-errChan:
		t.Logf("Server stopped with: %v", err)
		assert.NoError(t, err, "Server should stop without error")
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not stop")
	}
}

func TestParseToken(t *testing.T) {
	setupTestConfig()
	logger := zap.NewNop()
	server := NewServer(ServerConfig{Logger: logger})

	t.Run("valid token", func(t *testing.T) {
		tokenResp, err := server.getShortenerServer().GetSessionToken(context.Background(), &emptypb.Empty{})
		require.NoError(t, err)

		tokenInfo, err := server.parseToken(tokenResp.GetToken())

		require.NoError(t, err)
		assert.NotEmpty(t, tokenInfo.UserID)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := server.parseToken("invalid-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode cookie")
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := server.parseToken("")
		require.Error(t, err)
	})
}

func (s *Server) getShortenerServer() *ShortenerServer {
	return NewShortenerServer(s.logger)
}

func TestAuthInterceptor(t *testing.T) {
	setupTestConfig()
	logger := zap.NewNop()
	server := NewServer(ServerConfig{Logger: logger})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	t.Run("public method", func(t *testing.T) {
		info := &grpc.UnaryServerInfo{
			FullMethod: "/vsevolodryzhov.urlshortner.proto.ShortenerService/GetSessionToken",
		}
		ctx := context.Background()

		resp, err := server.authInterceptor(ctx, nil, info, handler)

		require.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("protected method without auth", func(t *testing.T) {
		info := &grpc.UnaryServerInfo{
			FullMethod: "/vsevolodryzhov.urlshortner.proto.ShortenerService/ShortenURL",
		}
		ctx := context.Background()

		resp, err := server.authInterceptor(ctx, nil, info, handler)

		require.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("protected method with auth", func(t *testing.T) {
		t.Skip("Requires metadata context with valid token")
	})
}
