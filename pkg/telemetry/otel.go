// Package telemetry wires up OpenTelemetry metrics (exported via Prometheus)
// and a no-op trace provider. Swap the trace provider for an OTLP exporter
// when you have a collector (Jaeger, Tempo, etc.) in your stack.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Provider holds the initialised SDK providers.
type Provider struct {
	Meter *metric.MeterProvider
	Trace *sdktrace.TracerProvider
}

// Setup initialises OTel with a Prometheus metrics exporter and a no-op tracer.
// Call the returned cleanup func on shutdown (it flushes and stops both providers).
func Setup(serviceName string) (*Provider, func(), error) {
	// Prometheus exporter — metrics are scraped from /metrics.
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry: prometheus exporter: %w", err)
	}

	mp := metric.NewMeterProvider(metric.WithReader(promExporter))
	otel.SetMeterProvider(mp)

	// No-op tracer — replace with an OTLP exporter when you have a collector.
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cleanup := func() {
		_ = mp.Shutdown(context.Background())
		_ = tp.Shutdown(context.Background())
	}

	return &Provider{Meter: mp, Trace: tp}, cleanup, nil
}
