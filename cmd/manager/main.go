package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/nats-io/nats.go"

	"deepbooru"
	"deepbooru/internal/authorizer/http"
)

var authorizerUrl = ""
var databaseUrl = ""
var natsUrl = nats.DefaultURL
var minAccessLevel = deepbooru.Anonymous

func getenv(name, defaultValue string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}

	return defaultValue
}

func init() {
	flag.StringVar(&authorizerUrl, "a", getenv("AUTHORIZER_URL", authorizerUrl), "Authorizer URL")
	flag.StringVar(&databaseUrl, "d", getenv("DATABASE_URL", databaseUrl), "Database URL")
	flag.StringVar(&natsUrl, "n", getenv("NATS_URL", natsUrl), "NATS URL")
	flag.Parse()
}

type CloserFunc func()

func noop() {}

func getAuthorizer(url string) (deepbooru.Authorizer, CloserFunc) {
	if url == "" {
		return deepbooru.NoopAuthorizer(), noop
	}

	if url[:7] == "http://" || url[:8] == "https://" {
		return http_authorizer.New(url), noop
	}

	panic("invalid authorizer url")
}

func getStorage(url string) (deepbooru.Storage, CloserFunc) {
	return nil, noop
}

func getBus(url string) (deepbooru.BusFactory, CloserFunc) {
	return nil, noop
}

func main() {
	authorizer, closeAuthorizer := getAuthorizer(authorizerUrl)
	storage, closeStorage := getStorage(databaseUrl)
	bus, closeBus := getBus(natsUrl)

	defer closeAuthorizer()
	defer closeStorage()
	defer closeBus()

	manager := deepbooru.NewManager(authorizer, bus, storage)

	sigs := make(chan os.Signal, 1)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	err := manager.Run(ctx)

	if err != nil {
		panic(err)
	}

	signal.Notify(sigs, os.Interrupt, os.Kill)

	<-sigs
	cancel()
}
