package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/4925k/usdl/chat/foundation/logger"
)

func main() {
	var log *logger.Logger

	tradeIDFn := func(ctx context.Context) string {
		return "" // TODO
	}

	log = logger.New(os.Stdout, logger.LevelInfo, "CAP", tradeIDFn)

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Startup

	log.Info(ctx, "startup", "status", "complete")
	defer log.Info(ctx, "shutdown", "status", "complete")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	return nil
}
