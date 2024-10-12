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
	"golang.org/x/sync/errgroup"
)

func TestMastodonStream(t *testing.T) {
	if testing.Short() {
		return
	}

	ctx := context.Background()

	server := NewServer()

	httpServer := ServeHTTP(server, true, "8080", func() {})
	defer ShutdownHTTP(httpServer)

	// Give the server a moment to come up.
	time.Sleep(time.Second)

	client := protoconnect.NewMastodonClient(http.DefaultClient, "http://localhost:8080")
	stream, err := client.StreamTimeline(ctx, connect.NewRequest(&proto.StreamTimelineRequest{
		Server: "https://mastodon.social",
	}))
	if err != nil {
		t.Error("StreamTimeline", err)
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
		t.Error("stream.Err", err)
		return
	}

	if count < 45 {
		t.Fatalf("expected at least 45 events, got %d", count)
	}
}

// Too many requests toward the same Mastodon server will result in rate limiting.  This test verifies that our implementation makes only as many requests as necessary.
func TestMastodonStream_ConcurrentStreams(t *testing.T) {
	if testing.Short() {
		return
	}

	ctx := context.Background()

	server := NewServer()

	httpServer := ServeHTTP(server, true, "8080", func() {})
	defer ShutdownHTTP(httpServer)

	// Give the server a moment to come up.
	time.Sleep(time.Second)

	client := protoconnect.NewMastodonClient(http.DefaultClient, "http://localhost:8080")

	var eg errgroup.Group
	for range 100 {
		eg.Go(func() error {
			stream, err := client.StreamTimeline(ctx, connect.NewRequest(&proto.StreamTimelineRequest{
				Server: "https://mastodon.social",
			}))
			if err != nil {
				return fmt.Errorf("StreamTimeline: %v", err)
			}

			deadline := time.Now().Add(5 * time.Second)
			var lastID string
			count := 0
			for time.Now().Before(deadline) && stream.Receive() {
				msg := stream.Msg()
				if msg.Id <= lastID {
					return fmt.Errorf("expected increasing IDs, got %s <= %s", msg.Id, lastID)
				}
				lastID = msg.Id
				count++
			}
			if err := stream.Err(); err != nil {
				return fmt.Errorf("stream.Err: %v", err)
			}

			if count < 45 {
				return fmt.Errorf("expected at least 45 events, got %d", count)
			}

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		t.Error(err)
	}
}
