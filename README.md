# mastodon-stream

Stream public timelines from Mastodon servers.

Run the server with

```sh
go run ./cmd/server -localhost
```

optionally specifying a port to listen on with the `-http-port` option.  Omit `-localhost` if you want to accept connections from beyond this machine.

Then, run the client with

```sh
go run ./cmd/client
```

optionally specifying the gRPC server to connect to with the `-grpc-server` option, and the Mastodon server to stream from with `-mastodon-server`.

## gRPC / Protocol Buffers

Run `go generate ./proto` after changing any `.proto` files.
