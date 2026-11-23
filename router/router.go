package router

import (
	"go-postgres/middleware"
	"go-postgres/telemetry"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Router() *mux.Router {
	telemetry.InitMetrics()

	router := mux.NewRouter()

	router.HandleFunc("/healthz", middleware.Healthz).Methods("GET")
	router.HandleFunc("/readyz", middleware.Readyz).Methods("GET")

	router.HandleFunc("/api/user/{id}", middleware.GetUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/user", middleware.GetAllUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/newuser", middleware.CreateUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/user/{id}", middleware.UpdateUser).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/user/{id}", middleware.DeleteUser).Methods("DELETE", "OPTIONS")

	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	router.Use(middleware.Metrics)
	router.Use(middleware.Tracing)

	return router
}
