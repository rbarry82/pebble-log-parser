package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/canonical/pebble/client"
	"github.com/go-logfmt/logfmt"
	"go.uber.org/zap"
)

type pebbleLogger struct {
	Service string
	client  *client.Client
}

type LogStreamer struct {
	SignalChan chan os.Signal
	Services   map[string]*pebbleLogger
	Context    *context.Context
}

var timeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05.999999999-0700", // Of course Grafana has to use their own. There will probably be more.
}

func NewLogStreamer(ctx context.Context, socket string, services []string) (*LogStreamer, error) {
	streamer := &LogStreamer{
		SignalChan: make(chan os.Signal, 1),
		Services:   make(map[string]*pebbleLogger),
		Context:    &ctx,
	}

	pebble, err := client.New(&client.Config{Socket: socket})
	if err != nil {
		zap.L().Fatal(
			"Could not spawn a pebble client!",
			zap.String("service", "logstreamer"),
		)
		return nil, err
	}

	for _, service := range services {
		zap.L().Info(
			"Starting logger",
			zap.String("service", service),
		)
		p := &pebbleLogger{
			Service: service,
			client:  pebble,
		}
		go p.stream(ctx)
	}
	return streamer, nil
}

func (pl *pebbleLogger) stream(ctx context.Context) {
	writeLogFunc := func(entry client.LogEntry) error {
		//data := make(map[string]string)
		values := make(map[string]string)
		dec := logfmt.NewDecoder(strings.NewReader(entry.Message))

		for dec.ScanRecord() {
			for dec.ScanKeyval() {
				k := dec.Key()

				if k != nil {
					// Don't relog times
					if isTimeEntry(string(k)) {
						continue
					}

					values[string(k)] = string(dec.Value())
				}
			}
		}
		zap.L().Info(
			"",
			append([]zap.Field{
				zap.String("timestamp", entry.Time.String()),
				zap.String("service", entry.Service)},
				mapToZap(values)...,
			)...,
		)
		return nil
	}

	pl.client.FollowLogs(ctx, &client.LogsOptions{
		Services: []string{pl.Service},
		WriteLog: writeLogFunc,
	})
	select {
	case <-ctx.Done():
		return
	}
}

func mapToZap(m map[string]string) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.String(k, v))
	}
	return fields
}

func isTimeEntry(value string) bool {
	for _, f := range timeFormats {
		_, err := time.Parse(f, value)
		if err == nil {
			return true
		}
	}
	return false
}
