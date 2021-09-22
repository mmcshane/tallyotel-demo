package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mmcshane/tallyotel"
	tally "github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"
)

func DoGlobalSetup() io.Closer {
	// Tally + Prometheus
	r := prometheus.NewReporter(prometheus.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:         "my_service",
		Tags:           map[string]string{"a": "b"},
		CachedReporter: r,
		Separator:      prometheus.DefaultSeparator,
	}, 1*time.Second)
	http.Handle("/metrics", r.HTTPHandler())
	go func() {
		fmt.Println("Listening on http://localhost:8080/metrics")
		http.ListenAndServe(":8080", nil)
	}()

	// install tallyotel as the default global MeterProvider
	global.SetMeterProvider(tallyotel.NewMeterProvider(scope))
	return closer
}

func main() {
	rand.Seed(time.Now().UnixNano())
	closer := DoGlobalSetup()
	defer closer.Close()

	// allocate some instruments
	m := metric.Must(global.Meter("foo.bar"))
	ctr := m.NewInt64Counter("loops").Bind(attribute.Key("x").Int(1))
	hist := m.NewInt64Histogram("numbers").Bind(attribute.Key("x").Int(1))
	durhist := m.NewFloat64Histogram("request_duration_seconds", metric.WithUnit(unit.Milliseconds))

	// use the instruments
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var i int64
	for {
		ctr.Add(ctx, 1)
		hist.Record(ctx, i)
		durhist.Record(ctx, (rand.Float64() * 1000.0))
		i++
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
