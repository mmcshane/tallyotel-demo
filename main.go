package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mmcshane/tallyotel"
	"github.com/uber-go/tally"
	"github.com/uber-go/tally/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"
)

func main() {
	// Global Setup
	// ------------------------------------------------------------------------

	// Tally + Prometheus
	r := prometheus.NewReporter(prometheus.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:         "my_service",
		Tags:           map[string]string{},
		CachedReporter: r,
		Separator:      prometheus.DefaultSeparator,
	}, 1*time.Second)
	defer closer.Close()
	http.Handle("/metrics", r.HTTPHandler())
	go func() {
		fmt.Println("Listening on http://localhost:8080/metrics")
		http.ListenAndServe(":8080", nil)
	}()

	// install tallyotel as the default global MeterProvider
	global.SetMeterProvider(tallyotel.NewMeterProvider(scope))

	// Local metric instruments
	// ------------------------------------------------------------------------
	m := metric.Must(global.Meter("foooo"))
	ctr := m.NewInt64Counter("c1").Bind(attribute.Key("x").Int(1))
	hist := m.NewInt64Histogram("h1").Bind(attribute.Key("x").Int(1))
	durhist := m.NewFloat64Histogram("h2", metric.WithUnit(unit.Milliseconds))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		var i int64
		start := time.Now()
		for {
			ctr.Add(ctx, 1)
			hist.Record(ctx, i)
			durhist.Record(ctx, float64(time.Since(start).Milliseconds()))
			i++
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
}
