package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microservices-bank/account-service/config"
	grpcserver "microservices-bank/account-service/grpc"
	"microservices-bank/account-service/handler"
	"microservices-bank/account-service/middleware"
	kafkapkg "microservices-bank/account-service/pkg/kafka"
	"microservices-bank/account-service/pkg/logger"
	"microservices-bank/account-service/pkg/metrics"
	"microservices-bank/account-service/pkg/otel"
	"microservices-bank/account-service/repository"
	"microservices-bank/account-service/server"
	"microservices-bank/account-service/service"
	pb "microservices-bank/proto/accountpb"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
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
		{Topic: kafkapkg.TopicBalanceChange},
		{Topic: kafkapkg.TopicDeadLetter},
	})

	// Publisher
	publisher := kafkapkg.NewKafkaPublisher(config.KafkaWriter)

	// Repository
	accountRepo := repository.NewAccountRepository(database)

	// Service
	accountService := &service.AccountService{
		Repo:      accountRepo,
		Publisher: publisher,
	}

	// gRPC Server 
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterAccountServiceServer(grpcSrv, grpcserver.NewAccountGRPCServer(accountRepo))

	go func() {
		log.Printf("[gRPC] Server listening on :%s", grpcPort)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			log.Fatal("[gRPC] server error:", err)
		}
	}()

	// HTTP Server
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"account-service"}`))
	})

	handler.NewAccountHandler(mux, accountService).MapRoutes()

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
		log.Printf("[HTTP] Server running on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")

	grpcSrv.GracefulStop()
	log.Println("[gRPC] Server stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("server forced shutdown:", err)
	}
	log.Println("Server exited gracefully")
}
