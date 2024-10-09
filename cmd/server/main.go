package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gordonklaus/mastodon-stream/proto/protoconnect"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	slog.Info("Starting.")

	var (
		localhost = flag.Bool("localhost", false, "serve only on localhost")
		httpPort  = flag.String("http-port", "8080", "port for the HTTP server")
	)
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	server := NewServer()

	host := ""
	if *localhost {
		host = "localhost"
	}

	router := http.NewServeMux()
	router.Handle(protoconnect.NewMastodonHandler(server))
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, *httpPort),
		Handler: router,
	}
	go func() {
		defer cancel()
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("httpServer.ListenAndServe", "err", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	slog.Info("Done.")
}
