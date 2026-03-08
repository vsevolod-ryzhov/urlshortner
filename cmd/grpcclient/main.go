package main

import (
	"context"
	"fmt"
	"os"

	pb "github.com/vsevolod-ryzhov/urlshortner.git/api/proto"

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

	if err := SensRequests(ctx, c); err != nil {
		fmt.Println("error sending requests %w", err)
		os.Exit(1)
	}
}

func SensRequests(ctx context.Context, c pb.ShortenerServiceClient) error {
	links := []string{"https://ya.ru", "https://ya.com"}
	for _, link := range links {
		response, err := c.ShortenURL(ctx, pb.URLShortenRequest_builder{
			Url: &link,
		}.Build())

		if err != nil {
			return fmt.Errorf("shorten URL request error: %w", err)
		}

		fmt.Println(response.GetResult())
	}
	return nil
}
