package main

import (
	"context"
	"fmt"
	"os"

	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()

	conn, err := grpc.NewClient(`:3200`, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("server connection error %w", err)
		os.Exit(1)
	}
	defer conn.Close()
	c := pb.NewShortenerServiceClient(conn)

	tokenResp, err := c.GetSessionToken(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("failed to get session token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Got token: %s\n", tokenResp.GetToken())

	md := metadata.New(map[string]string{
		"authorization": "bearer " + tokenResp.GetToken(),
	})
	authCtx := metadata.NewOutgoingContext(ctx, md)

	testLinks := []string{"https://ya.ru", "https://ya.com"}
	if err := SendRequests(authCtx, c, testLinks); err != nil {
		fmt.Println("error sending requests %w", err)
		os.Exit(1)
	}
}

func SendRequests(ctx context.Context, c pb.ShortenerServiceClient, links []string) error {
	for _, link := range links {
		responseShorten, errShorten := c.ShortenURL(ctx, pb.URLShortenRequest_builder{
			Url: &link,
		}.Build())

		if errShorten != nil {
			return fmt.Errorf("shorten URL request error: %w", errShorten)
		}

		shortened := responseShorten.GetResult()
		fmt.Println(shortened)

		responseExpand, errExpand := c.ExpandURL(ctx, pb.URLExpandRequest_builder{
			Id: &shortened,
		}.Build())

		if errExpand != nil {
			return fmt.Errorf("expand URL request error: %w", errExpand)
		}

		fmt.Println(responseExpand.GetResult())
	}

	response, err := c.ListUserURLs(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("list user URLs request error: %w", err)
	}
	fmt.Println(response)
	return nil
}
