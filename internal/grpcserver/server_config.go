// internal/grpcserver/server_config.go
package grpcserver

import (
	"context"
	"fmt"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type tokenInfo struct {
	UserID string
}

type ServerConfig struct {
	Port   string
	Logger *zap.Logger
}

type Server struct {
	config     ServerConfig
	grpcServer *grpc.Server
	logger     *zap.Logger
}

func NewServer(config ServerConfig) *Server {
	return &Server{
		config: config,
		logger: config.Logger,
	}
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", s.config.Port)
	if err != nil {
		return fmt.Errorf("gRPC listener init error: %w", err)
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.authInterceptor),
	)

	shortenerServer := NewShortenerServer(s.logger)
	pb.RegisterShortenerServiceServer(s.grpcServer, shortenerServer)

	s.logger.Info("gRPC server started successfully", zap.String("port", s.config.Port))

	if err := s.grpcServer.Serve(listen); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC server stopped")
	}
}

func (s *Server) authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	publicMethods := map[string]bool{
		"/vsevolodryzhov.urlshortner.proto.ShortenerService/GetSessionToken": true,
	}

	if publicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	newCtx, err := s.grpcAuth(ctx)
	if err != nil {
		return nil, err
	}

	return handler(newCtx, req)
}

func (s *Server) grpcAuth(ctx context.Context) (context.Context, error) {
	token, err := auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	tokenInfo, err := s.parseToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	ctx = logging.InjectFields(ctx, logging.Fields{"user.id", tokenInfo.UserID})
	ctx = context.WithValue(ctx, UserIDKey, tokenInfo.UserID)

	return ctx, nil
}

func (s *Server) parseToken(token string) (*tokenInfo, error) {
	session, err := service.DecodeAndVerifyCookie(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cookie: %w", err)
	}

	return &tokenInfo{
		UserID: session.UserID,
	}, nil
}
