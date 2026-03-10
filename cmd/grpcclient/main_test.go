package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
)

type MockShortenerServiceClient struct {
	mock.Mock
	pb.ShortenerServiceClient
}

func (m *MockShortenerServiceClient) GetSessionToken(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*pb.SessionTokenResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.SessionTokenResponse), args.Error(1)
}

func (m *MockShortenerServiceClient) ShortenURL(ctx context.Context, in *pb.URLShortenRequest, opts ...grpc.CallOption) (*pb.URLShortenResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.URLShortenResponse), args.Error(1)
}

func (m *MockShortenerServiceClient) ExpandURL(ctx context.Context, in *pb.URLExpandRequest, opts ...grpc.CallOption) (*pb.URLExpandResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.URLExpandResponse), args.Error(1)
}

func (m *MockShortenerServiceClient) ListUserURLs(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*pb.UserURLsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.UserURLsResponse), args.Error(1)
}

func TestSendRequests(t *testing.T) {
	tests := []struct {
		name          string
		links         []string
		setupMock     func(*MockShortenerServiceClient)
		expectedError bool
	}{
		{
			name:  "successful requests",
			links: []string{"https://ya.ru", "https://ya.com"},
			setupMock: func(m *MockShortenerServiceClient) {
				m.On("ShortenURL", mock.Anything,
					pb.URLShortenRequest_builder{Url: proto.String("https://ya.ru")}.Build()).
					Return(pb.URLShortenResponse_builder{Result: proto.String("short1")}.Build(), nil).Once()

				m.On("ExpandURL", mock.Anything,
					pb.URLExpandRequest_builder{Id: proto.String("short1")}.Build()).
					Return(pb.URLExpandResponse_builder{Result: proto.String("https://ya.ru")}.Build(), nil).Once()

				m.On("ShortenURL", mock.Anything,
					pb.URLShortenRequest_builder{Url: proto.String("https://ya.com")}.Build()).
					Return(pb.URLShortenResponse_builder{Result: proto.String("short2")}.Build(), nil).Once()

				m.On("ExpandURL", mock.Anything,
					pb.URLExpandRequest_builder{Id: proto.String("short2")}.Build()).
					Return(pb.URLExpandResponse_builder{Result: proto.String("https://ya.com")}.Build(), nil).Once()

				m.On("ListUserURLs", mock.Anything, &emptypb.Empty{}).
					Return(pb.UserURLsResponse_builder{
						Url: []*pb.URLData{
							pb.URLData_builder{
								ShortUrl:    proto.String("short1"),
								OriginalUrl: proto.String("https://ya.ru"),
							}.Build(),
							pb.URLData_builder{
								ShortUrl:    proto.String("short2"),
								OriginalUrl: proto.String("https://ya.com"),
							}.Build(),
						},
					}.Build(), nil).Once()
			},
			expectedError: false,
		},
		{
			name:  "shorten URL error",
			links: []string{"https://ya.ru"},
			setupMock: func(m *MockShortenerServiceClient) {
				m.On("ShortenURL", mock.Anything,
					pb.URLShortenRequest_builder{Url: proto.String("https://ya.ru")}.Build()).
					Return((*pb.URLShortenResponse)(nil), errors.New("shorten error")).Once()
			},
			expectedError: true,
		},
		{
			name:  "expand URL error",
			links: []string{"https://ya.ru"},
			setupMock: func(m *MockShortenerServiceClient) {
				m.On("ShortenURL", mock.Anything,
					pb.URLShortenRequest_builder{Url: proto.String("https://ya.ru")}.Build()).
					Return(pb.URLShortenResponse_builder{Result: proto.String("short1")}.Build(), nil).Once()
				m.On("ExpandURL", mock.Anything,
					pb.URLExpandRequest_builder{Id: proto.String("short1")}.Build()).
					Return((*pb.URLExpandResponse)(nil), errors.New("expand error")).Once()
			},
			expectedError: true,
		},
		{
			name:  "list URLs error",
			links: []string{"https://ya.ru"},
			setupMock: func(m *MockShortenerServiceClient) {
				m.On("ShortenURL", mock.Anything,
					pb.URLShortenRequest_builder{Url: proto.String("https://ya.ru")}.Build()).
					Return(pb.URLShortenResponse_builder{Result: proto.String("short1")}.Build(), nil).Once()
				m.On("ExpandURL", mock.Anything,
					pb.URLExpandRequest_builder{Id: proto.String("short1")}.Build()).
					Return(pb.URLExpandResponse_builder{Result: proto.String("https://ya.ru")}.Build(), nil).Once()
				m.On("ListUserURLs", mock.Anything, &emptypb.Empty{}).
					Return((*pb.UserURLsResponse)(nil), errors.New("list error")).Once()
			},
			expectedError: true,
		},
		{
			name:  "empty links slice",
			links: []string{},
			setupMock: func(m *MockShortenerServiceClient) {
				m.On("ListUserURLs", mock.Anything, &emptypb.Empty{}).
					Return(pb.UserURLsResponse_builder{}.Build(), nil).Once()
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockShortenerServiceClient)
			tt.setupMock(mockClient)

			ctx := context.Background()
			err := SendRequests(ctx, mockClient, tt.links)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetSessionToken(t *testing.T) {
	mockClient := new(MockShortenerServiceClient)

	expectedToken := pb.SessionTokenResponse_builder{
		Token: proto.String("test-token-123"),
	}.Build()

	mockClient.On("GetSessionToken", mock.Anything, &emptypb.Empty{}).
		Return(expectedToken, nil)

	ctx := context.Background()
	tokenResp, err := mockClient.GetSessionToken(ctx, &emptypb.Empty{})

	require.NoError(t, err)
	assert.Equal(t, expectedToken.GetToken(), tokenResp.GetToken())
	mockClient.AssertExpectations(t)
}
