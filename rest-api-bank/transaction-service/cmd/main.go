package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"transaction-service/config"
	"transaction-service/handler"
	"transaction-service/middleware"
	kafkapkg "transaction-service/pkg/kafka"
	"transaction-service/pkg/logger"
	"transaction-service/pkg/metrics"
	"transaction-service/pkg/otel"
	"transaction-service/repository"
	"transaction-service/server"
	"transaction-service/service"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

func main() {
	godotenv.Load()

	logger.InitLogger()
	defer logger.Logger.Sync()

	shutdown := otel.InitTracer(context.Background())
	defer shutdown(context.Background())

	metrics.Init()

	config.InitRedis()

	database := config.InitDB()
	config.RunMigrations(database)

	// Kafka
	config.InitKafka()
	defer config.CloseKafka()
	config.EnsureTopics([]kafka.TopicConfig{
		{Topic: kafkapkg.TopicTransaction},
		{Topic: kafkapkg.TopicDeadLetter},
	})

	// gRPC Client to Account Service
	grpcConn := config.InitGRPCClient()
	defer grpcConn.Close()

	// Publisher
	publisher := kafkapkg.NewKafkaPublisher(config.KafkaWriter)

	// Repository
	transactionRepo := repository.NewTransactionRepository(database)

	// Service
	transferService := &service.TransferService{
		TransactionRepo: transactionRepo,
		Publisher:        publisher,
		AccountClient:   config.AccountClient,
	}

	// HTTP
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"transaction-service"}`))
	})

	handler.NewTransferHandler(mux, transferService).MapRoutes()

	handlerChain := middleware.Metrics(
		middleware.Observability(
			server.ApplicationMiddlewareResponse(
				middleware.Timeout(20*time.Second)(
					server.HandleRouteNotFound(mux),
				),
			),
		),
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      handlerChain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[HTTP] Transaction Service running on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("server forced shutdown:", err)
	}
	log.Println("Server exited gracefully")
}
