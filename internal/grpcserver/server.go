// Package grpcserver
package grpcserver

import (
	"context"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"go.uber.org/zap"
)

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

	userID, ok := service.GetUserIDFromContext(ctx)
	if !ok {
		userID = "unknown"
	}
	shortened, _, errCreation := service.CreateShortURL(ctx, in.GetUrl(), userID)
	if errCreation != nil {
		logger.Log.Debug("Shortened result was not saved to file", zap.Error(errCreation))
	}

	response.SetResult(shortened)

	return &response, nil
}

func (s *ShortenerServer) ExpandURL(ctx context.Context, in *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	var response pb.URLExpandResponse

	return &response, nil
}

func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	var response pb.UserURLsResponse

	return &response, nil
}
