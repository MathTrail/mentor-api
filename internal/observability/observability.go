package observability

import (
	"context"
	"os"

	"github.com/grafana/pyroscope-go"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/MathTrail/mentor-api/internal/config"
)

// InitTracer initialises the global OTLP gRPC trace exporter and TracerProvider.
// It also sets the global W3C TraceContext + Baggage propagator so that
// traceparent headers injected by the Dapr sidecar are automatically
// extracted and the Go spans appear as children of the Dapr spans in Tempo.
// Call the returned shutdown function during graceful shutdown to flush pending spans.
func InitTracer(cfg *config.Config) (shutdown func(context.Context) error, err error) {
	conn, err := grpc.NewClient(
		cfg.OTelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.K8SPodName(os.Getenv("POD_NAME")),
			semconv.K8SNamespaceName(os.Getenv("NAMESPACE")),
		),
		resource.WithFromEnv(), // picks up OTEL_RESOURCE_ATTRIBUTES if set
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent — compatible with Dapr
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// InitMetrics initialises the global Prometheus MeterProvider.
// The Prometheus exporter registers with prometheus.DefaultRegisterer so
// promhttp.Handler() in the router automatically exposes all OTel metrics.
// Call the returned shutdown function during graceful shutdown.
func InitMetrics() (shutdown func(context.Context) error, err error) {
	exporter, err := promexporter.New()
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

// InitPyroscope starts the Pyroscope continuous profiling agent.
// It collects CPU, allocation and in-use heap profiles and pushes them to the
// Pyroscope server. Call profiler.Stop() during graceful shutdown.
func InitPyroscope(cfg *config.Config) (*pyroscope.Profiler, error) {
	return pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   cfg.PyroscopeEndpoint,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
}
