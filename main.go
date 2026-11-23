package main

import (
	"context"
	"fmt"
	"go-postgres/middleware"
	"go-postgres/router"
	"go-postgres/telemetry"
	"log"
	"log/slog"
	"net/http"
	"time"
)

func main() {
	ctx := context.Background()

	traceProvider, err := telemetry.InitTracing(ctx)
	if err != nil {
		log.Fatalf("failed to initialize OpenTelemetry tracing: %v", err)
	}

	logProvider, err := telemetry.InitLogging(ctx)
	if err != nil {
		log.Fatalf("failed to initialize OpenTelemetry logging: %v", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := telemetry.ShutdownTracing(
			shutdownCtx,
			traceProvider,
		); err != nil {
			log.Printf(
				"failed to shutdown OpenTelemetry tracing: %v",
				err,
			)
		}

		if err := telemetry.ShutdownLogging(
			shutdownCtx,
			logProvider,
		); err != nil {
			log.Printf(
				"failed to shutdown OpenTelemetry logging: %v",
				err,
			)
		}
	}()

	middleware.InitDatabase()

	defer func() {
		if err := middleware.CloseDatabase(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	r := router.Router()

	slog.Info("Starting server", "port", 8080)

	fmt.Println("Starting server on the port 8080...")

	log.Fatal(http.ListenAndServe(":8080", r))
}
