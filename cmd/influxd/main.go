package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/influxdata/influxdb/cmd/influxd/launcher"
	"github.com/influxdata/influxdb/kit/signals"
	_ "github.com/influxdata/influxdb/query/builtin"
	_ "github.com/influxdata/influxdb/tsdb/tsi1"
	_ "github.com/influxdata/influxdb/tsdb/tsm1"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// exit with SIGINT and SIGTERM
	ctx := context.Background()
	ctx = signals.WithStandardSignals(ctx)

	m := launcher.NewLauncher()
	m.SetBuild(version, commit, date)
	if err := m.Run(ctx, os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else if !m.Running() {
		os.Exit(1)
	}

	if !m.ReportingDisabled() {
		go m.ReportUsageStats(ctx, 8*time.Hour)
	}

	<-ctx.Done()

	// Attempt clean shutdown.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	m.Shutdown(ctx)
}
