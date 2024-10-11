package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/gordonklaus/mastodon-stream/proto"
	"github.com/gordonklaus/mastodon-stream/proto/protoconnect"
)

func TestMastodonStream(t *testing.T) {
	if testing.Short() {
		return
	}

	server := NewServer()

	router := http.NewServeMux()
	router.Handle(protoconnect.NewMastodonHandler(server))
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("localhost:8080"),
		Handler: router,
	}
	go httpServer.ListenAndServe()

	client := protoconnect.NewMastodonClient(http.DefaultClient, "http://localhost:8080")
	stream, err := client.StreamTimeline(context.Background(), connect.NewRequest(&proto.StreamTimelineRequest{
		Server: "https://mastodon.social",
	}))
	if err != nil {
		t.Error("StreamTimeline", "err", err)
		return
	}

	deadline := time.Now().Add(5 * time.Second)
	var lastID string
	count := 0
	for time.Now().Before(deadline) && stream.Receive() {
		msg := stream.Msg()
		if msg.Id <= lastID {
			t.Fatalf("expected increasing IDs, got %s <= %s", msg.Id, lastID)
		}
		lastID = msg.Id
		count++
	}
	if err := stream.Err(); err != nil {
		t.Error("stream.Err", "err", err)
		return
	}

	if count < 50 {
		t.Fatalf("expected at least 50 events, got %d", count)
	}
}
