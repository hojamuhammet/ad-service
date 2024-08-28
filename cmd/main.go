package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-service/internal/config"
	"ad-service/internal/delivery/router"
	"ad-service/internal/infrastructure/cache"
	"ad-service/internal/infrastructure/metrics"
	"ad-service/internal/repository"
	"ad-service/internal/service"
	"ad-service/pkg/database"
	"ad-service/pkg/logger"
	"ad-service/pkg/utils"

	"github.com/go-chi/chi/v5"
	redisClient "github.com/go-redis/redis/v8"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	cfg := config.MustLoadConfig()

	loggers, err := logger.SetupLogger(cfg.Logger.Level)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}
	loggers.InfoLogger.Info("Logger initialized")

	db, cleanupDB := setupDatabase(cfg, loggers)
	defer cleanupDB()

	redisCache, cleanupRedis := setupRedis(cfg, loggers)
	defer cleanupRedis()

	tracerProvider := setupTracer(cfg, loggers)
	defer shutdownTracer(tracerProvider, loggers)

	handlerMetrics := metrics.NewHandlerMetrics()
	serviceMetrics := metrics.NewServiceMetrics()
	repositoryMetrics := metrics.NewRepositoryMetrics()
	loggers.InfoLogger.Info("Prometheus metrics initialized")

	adRepo := repository.NewMysqlAdRepository(db, redisCache, repositoryMetrics)
	adService := service.NewAdService(adRepo, serviceMetrics)
	loggers.InfoLogger.Info("Service and repository layers initialized")

	r := chi.NewRouter()
	router.SetupAdRoutes(r, adService, loggers, handlerMetrics)
	loggers.InfoLogger.Info("Router and routes initialized")

	r.Handle("/metrics", handlerMetrics.HTTPHandler())

	server := startServer(cfg, r, loggers)

	waitForShutdown(server, loggers)
}

func setupDatabase(cfg *config.Config, loggers *logger.Loggers) (*sql.DB, func()) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name)

	db, err := database.NewDatabase(dsn)
	if err != nil {
		loggers.ErrorLogger.Error("Failed to connect to database", utils.Err(err))
		os.Exit(1)
	}
	loggers.InfoLogger.Info("Connected to database")

	cleanup := func() {
		if err := db.Close(); err != nil {
			loggers.ErrorLogger.Error("Failed to close database connection", utils.Err(err))
		}
	}

	return db, cleanup
}

func setupRedis(cfg *config.Config, loggers *logger.Loggers) (cache.Cache, func()) {
	rdb := redisClient.NewClient(&redisClient.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		loggers.ErrorLogger.Error("Failed to connect to Redis", utils.Err(err))
		os.Exit(1)
	}
	loggers.InfoLogger.Info("Connected to Redis")

	cleanup := func() {
		if err := rdb.Close(); err != nil {
			loggers.ErrorLogger.Error("Failed to close Redis client", utils.Err(err))
		}
	}

	return cache.NewRedisCache(rdb), cleanup
}

func setupTracer(cfg *config.Config, loggers *logger.Loggers) *sdktrace.TracerProvider {
	tracerProvider := metrics.InitTracer(
		cfg.Tracing.ServiceName,
		cfg.Tracing.Environment,
		cfg.Tracing.Version,
		cfg.Tracing.Endpoint,
	)
	loggers.InfoLogger.Info("OpenTelemetry Tracer initialized")
	return tracerProvider
}

func shutdownTracer(tp *sdktrace.TracerProvider, loggers *logger.Loggers) {
	if err := tp.Shutdown(context.Background()); err != nil {
		loggers.ErrorLogger.Error("Failed to shut down tracer provider", utils.Err(err))
	}
}

func startServer(cfg *config.Config, handler http.Handler, loggers *logger.Loggers) *http.Server {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.Timeout,
		WriteTimeout: cfg.HTTP.Timeout,
	}

	go func() {
		loggers.InfoLogger.Info("Starting server", "port", cfg.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			loggers.ErrorLogger.Error("Failed to start server", utils.Err(err))
			os.Exit(1)
		}
	}()

	return server
}

func waitForShutdown(server *http.Server, loggers *logger.Loggers) {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	<-shutdownCh
	loggers.InfoLogger.Info("Shutdown signal received, shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		loggers.ErrorLogger.Error("Server forced to shutdown", utils.Err(err))
	} else {
		loggers.InfoLogger.Info("Server shutdown gracefully")
	}
}
