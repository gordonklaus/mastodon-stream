package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
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

	httpServer := ServeHTTP(server, *localhost, *httpPort, cancel)

	<-ctx.Done()

	ShutdownHTTP(httpServer)

	slog.Info("Done.")
}
