package main

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/gordonklaus/mastodon-stream/proto"
	"github.com/mattn/go-mastodon"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) StreamTimeline(ctx context.Context, req *connect.Request[proto.StreamTimelineRequest], stream *connect.ServerStream[proto.StreamTimelineResponse]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	timeline := make(chan *mastodon.Status)
	go s.streamTimeline(ctx, req.Msg.Server, timeline)
	for status := range timeline {
		if err := stream.Send(&proto.StreamTimelineResponse{
			Id:        string(status.ID),
			CreatedAt: timestamppb.New(status.CreatedAt),
			Content:   status.Content,
		}); err != nil {
			slog.Error("stream.Send", "err", err)
			return nil
		}
	}
	return nil
}

func (s *Server) streamTimeline(ctx context.Context, mastodonServer string, timelineChan chan<- *mastodon.Status) {
	defer close(timelineChan)

	client := mastodon.NewClient(&mastodon.Config{
		Server: mastodonServer,
	})

	var sinceID mastodon.ID
	for {
		// The public streaming API (client.StreamingPublic) can no longer be used without authentication (https://github.com/mastodon/mastodon/pull/23989), so instead we resort to polling.
		// Because of limits on request rates and the number of events returned per request, if more than 40 Mastodon status events occur per request then we will miss the intervening ones; but we will always get those most recent.
		timeline, err := client.GetTimelinePublic(ctx, false, &mastodon.Pagination{
			SinceID: sinceID,
			Limit:   40, // 40 is the default maximum limit.
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			slog.Error("client.GetTimelinePublic", "err", err)
		}

		slices.SortFunc(timeline, func(x, y *mastodon.Status) int { return cmp.Compare(x.ID, y.ID) })
		for _, status := range timeline {
			timelineChan <- status

			sinceID = status.ID
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		// One request per second is the maximum (average) rate before we will be rate limited, by default.
		time.Sleep(1 * time.Second)
	}
}
