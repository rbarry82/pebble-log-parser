package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"go.uber.org/zap"
)

type Context struct {
	Debug bool
}

type LogCmd struct {
	PebbleSocket string   `arg:"" name:"pebble-socket" help:"Path to the pebble socket"`
	Services     []string `arg:"" name:"services" help:"Services to stream logs from"`
}

func (l *LogCmd) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	streamer, err := NewLogStreamer(ctx, l.PebbleSocket, l.Services)
	if err != nil {
		os.Exit(1)
	}

	signal.Notify(streamer.SignalChan, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-streamer.SignalChan:
			cancel()
			zap.L().Info("Shutting down log streamer", zap.String("services", strings.Join(l.Services, ",")))
			zap.L().Sync()
			os.Exit(0)
		}
	}
}

var cli struct {
	Log LogCmd `cmd:"" help:"Convert Pebble service logs into structured output"`
}

func main() {
	app := kong.Parse(&cli)

	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	zap.ReplaceGlobals(zapLogger)

	err := app.Run()
	app.FatalIfErrorf(err)
}
