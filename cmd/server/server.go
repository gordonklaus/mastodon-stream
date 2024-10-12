package main

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/gordonklaus/mastodon-stream/proto"
	"github.com/mattn/go-mastodon"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	mu      sync.Mutex
	streams map[string][]chan []*mastodon.Status
}

func NewServer() *Server {
	return &Server{
		streams: map[string][]chan []*mastodon.Status{},
	}
}

func (s *Server) StreamTimeline(ctx context.Context, req *connect.Request[proto.StreamTimelineRequest], stream *connect.ServerStream[proto.StreamTimelineResponse]) error {
	for timeline := range s.subscribe(req.Msg.Server) {
		for _, status := range timeline {
			if err := stream.Send(&proto.StreamTimelineResponse{
				Id:        string(status.ID),
				CreatedAt: timestamppb.New(status.CreatedAt),
				Content:   status.Content,
			}); err != nil {
				if !errors.Is(err, syscall.EPIPE) { // ignore "broken pipe" errors
					slog.Error("stream.Send", "err", err)
				}
				return nil
			}
		}
	}
	return nil
}

func (s *Server) subscribe(mastodonServer string) chan []*mastodon.Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan []*mastodon.Status, 1)
	s.streams[mastodonServer] = append(s.streams[mastodonServer], ch)
	if len(s.streams[mastodonServer]) == 1 {
		go s.streamTimeline(mastodonServer)
	}
	return ch
}

func (s *Server) streamTimeline(mastodonServer string) {
	client := mastodon.NewClient(&mastodon.Config{
		Server: mastodonServer,
	})

	var sinceID mastodon.ID
	for {
		// The public streaming API (client.StreamingPublic) can no longer be used without authentication (https://github.com/mastodon/mastodon/pull/23989), so instead we resort to polling.
		// Because of limits on request rates and the number of events returned per request, if more than 40 Mastodon status events occur per request then we will miss the intervening ones; but we will always get those most recent.
		timeline, err := client.GetTimelinePublic(context.Background(), false, &mastodon.Pagination{
			SinceID: sinceID,
			Limit:   40, // 40 is the default maximum limit.
		})
		if err != nil {
			slog.Error("client.GetTimelinePublic", "err", err)
		}

		slices.SortFunc(timeline, func(x, y *mastodon.Status) int { return cmp.Compare(x.ID, y.ID) })
		if !s.publish(mastodonServer, timeline) {
			return
		}
		if len(timeline) > 0 {
			sinceID = timeline[len(timeline)-1].ID
		}

		// One request per second is the maximum (average) rate before we will be rate limited, by default.
		time.Sleep(1 * time.Second)
	}
}

func (s *Server) publish(mastodonServer string, timeline []*mastodon.Status) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ch := range slices.Clone(s.streams[mastodonServer]) {
		select {
		case ch <- timeline:
		default:
			close(ch)
			i := slices.Index(s.streams[mastodonServer], ch)
			s.streams[mastodonServer] = slices.Delete(s.streams[mastodonServer], i, i+1)
			if len(s.streams[mastodonServer]) == 0 {
				delete(s.streams, mastodonServer)
				return false
			}
		}
	}
	return true
}
