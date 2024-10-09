package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/mattn/go-mastodon"
)

func main() {
	log.Println("Starting.")

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	client := mastodon.NewClient(&mastodon.Config{
		Server: "https://mastodon.social",
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
			log.Println(err)
		}

		slices.SortFunc(timeline, func(x, y *mastodon.Status) int { return cmp.Compare(x.ID, y.ID) })
		for _, status := range timeline {
			fmt.Println(status.ID, status.CreatedAt)

			sinceID = status.ID
		}

		// One request per second is the maximum (average) rate before we will be rate limited, by default.
		time.Sleep(1 * time.Second)
	}

	log.Println("Done.")
}
