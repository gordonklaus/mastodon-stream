package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gordonklaus/mastodon-stream/proto/protoconnect"
)

func ServeHTTP(server *Server, localhost bool, port string, done func()) *http.Server {
	host := ""
	if localhost {
		host = "localhost"
	}

	router := http.NewServeMux()
	router.Handle(protoconnect.NewMastodonHandler(server))
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: router,
	}
	go func() {
		defer done()
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("httpServer.ListenAndServe", "err", err)
		}
	}()
	return httpServer
}

func ShutdownHTTP(httpServer *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
