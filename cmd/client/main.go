package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bufbuild/connect-go"
	"github.com/gordonklaus/mastodon-stream/proto"
	"github.com/gordonklaus/mastodon-stream/proto/protoconnect"
)

func main() {
	var (
		grpcServer     = flag.String("grpc-server", "http://localhost:8080", "the gRPC server to connect to")
		mastodonServer = flag.String("mastodon-server", "https://mastodon.social", "the Mastodon server to stream from")
	)
	flag.Parse()

	ctx := context.Background()

	client := protoconnect.NewMastodonClient(http.DefaultClient, *grpcServer)

	stream, err := client.StreamTimeline(ctx, connect.NewRequest(&proto.StreamTimelineRequest{
		Server: *mastodonServer,
	}))
	if err != nil {
		slog.Error("StreamTimeline", "err", err)
		return
	}

	for stream.Receive() {
		msg := stream.Msg()
		fmt.Println(msg.Id, msg.CreatedAt.AsTime())
		fmt.Println(msg.Content)
		fmt.Println()
	}
	if err := stream.Err(); err != nil {
		slog.Error("stream.Err", "err", err)
		return
	}
}
