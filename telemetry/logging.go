package telemetry

import (
	"context"
    "log/slog"

    "go.opentelemetry.io/contrib/bridges/otelslog"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
    sdklog "go.opentelemetry.io/otel/sdk/log"
    "go.opentelemetry.io/otel/sdk/resource"
)

func InitLogging(ctx context.Context) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpoint("otel-collector:4318"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", "go-postgres"),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(exporter),
		),
		sdklog.WithResource(res),
	)

	handler := otelslog.NewHandler(
		"go-postgres",
		otelslog.WithLoggerProvider(provider),
	)

	slog.SetDefault(slog.New(handler))

	return provider, nil
}

func ShutdownLogging(
	ctx context.Context,
	provider *sdklog.LoggerProvider,
) error {
	if provider == nil {
		return nil
	}

	return provider.Shutdown(ctx)
}