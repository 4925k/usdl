package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/4925k/usdl/chat/foundation/logger"
	"github.com/4925k/usdl/chat/foundation/web"
)

func main() {
	var log *logger.Logger

	ctx := context.Background()

	tradeIDFn := func(ctx context.Context) string {
		return web.GetTraceID(ctx).String()
	}

	log = logger.New(os.Stdout, logger.LevelInfo, "CAP", tradeIDFn)

	// -------------------------------------------------------------------------

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
