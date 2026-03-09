// Package grpcserver
package grpcserver

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"go.uber.org/zap"
)

type grpcContextKey string

const (
	UserIDKey grpcContextKey = "userID"
)

func (s *ShortenerServer) getUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	logger *zap.Logger
}

func NewShortenerServer(logger *zap.Logger) *ShortenerServer {
	return &ShortenerServer{
		logger: logger,
	}
}

func (s *ShortenerServer) ShortenURL(ctx context.Context, in *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	var response pb.URLShortenResponse

	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	shortened, _, errCreation := service.CreateShortURL(ctx, in.GetUrl(), userID)
	if errCreation != nil {
		logger.Log.Debug("Shortened result was not saved to file", zap.Error(errCreation))
		return &response, errCreation
	}

	response.SetResult(shortened)

	return &response, nil
}

func (s *ShortenerServer) ExpandURL(ctx context.Context, in *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	var response pb.URLExpandResponse

	_, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	url, err := service.GetURL(in.GetId())
	if err != nil {
		return &response, err
	}

	response.SetResult(url)

	return &response, nil
}

func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	var response pb.UserURLsResponse

	userID, err := s.getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	urls, err := service.GetUserURLs(ctx, userID)
	if err != nil {
		return &response, errors.New("internal server error")
	}

	data := make([]*pb.URLData, len(urls))
	for _, url := range urls {
		ud := pb.URLData_builder{
			ShortUrl:    proto.String(url.ShortURL),
			OriginalUrl: proto.String(url.OriginalURL),
		}.Build()
		data = append(data, ud)
	}
	response.SetUrl(data)

	return &response, nil
}

func (s *ShortenerServer) GetSessionToken(ctx context.Context, _ *emptypb.Empty) (*pb.SessionTokenResponse, error) {
	var response pb.SessionTokenResponse

	userID := uuid.New().String()
	session := &service.UserSession{
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	token, err := service.EncodeSession(session)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session token: %v", err)
	}

	response.SetToken(token)

	return &response, nil
}
