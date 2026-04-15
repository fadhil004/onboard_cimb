package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rest-api-bank/config"
	"rest-api-bank/handler"
	"rest-api-bank/middleware"
	kafkapkg "rest-api-bank/pkg/kafka"
	"rest-api-bank/pkg/kafka/consumer"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/pkg/metrics"
	"rest-api-bank/pkg/otel"
	"rest-api-bank/repository"
	"rest-api-bank/server"
	"rest-api-bank/service"

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
		{Topic: kafkapkg.TopicAccountCreation},
		{Topic: kafkapkg.TopicTransaction},
		{Topic: kafkapkg.TopicBalanceChange},
		{Topic: kafkapkg.TopicDeadLetter},
	})

	// Publisher — di-inject ke service
	publisher := kafkapkg.NewKafkaPublisher(config.KafkaWriter)

	// Repository
	accountRepo     := repository.NewAccountRepository(database)
	transactionRepo := repository.NewTransactionRepository(database)

	// Service — Publisher di-inject di sini
	accountService := &service.AccountService{
		Repo:      accountRepo,
		Publisher: publisher,
	}
	transferService := &service.TransferService{
		AccountRepo:     accountRepo,
		TransactionRepo: transactionRepo,
		Publisher:       publisher,
	}

	// HTTP
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	handler.NewAccountHandler(mux, accountService, transferService).MapRoutes()
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

	// Kafka Consumers
	consumersCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()
	startConsumers(consumersCtx)

	// HTTP server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handlerChain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")

	cancelConsumers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("server forced shutdown:", err)
	}
	log.Println("Server exited gracefully")
}

func startConsumers(ctx context.Context) {
	start := func(groupID, topic string, handler consumer.Handler) {
		go func() {
			cg := consumer.NewConsumerGroup(groupID, topic, handler)
			log.Println("[Kafka] Consumer started:", topic)
			cg.Start(ctx)
		}()
	}

	start("bank-account-creation-logger", kafkapkg.TopicAccountCreation, consumer.LogAccountCreated)
	start("bank-transaction-logger",      kafkapkg.TopicTransaction,     consumer.LogTransactionCreated)
	start("bank-balance-change-logger",   kafkapkg.TopicBalanceChange,   consumer.LogBalanceChanged)
}
