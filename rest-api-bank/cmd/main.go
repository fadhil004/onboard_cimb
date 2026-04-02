package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"rest-api-bank/config"
	"rest-api-bank/handler"
	"rest-api-bank/middleware"
	"rest-api-bank/pkg/logger"
	"rest-api-bank/pkg/otel"
	"rest-api-bank/repository"
	"rest-api-bank/server"
	"rest-api-bank/service"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	logger.InitLogger()
	defer logger.Logger.Sync()

	shutdown := otel.InitTracer(context.Background())
	defer shutdown(context.Background())

	config.InitRedis()
	database := config.InitDB()
	config.RunMigrations(database)

	//  pakai constructor, bukan struct literal
	accountRepo := repository.NewAccountRepository(database)
	transactionRepo := repository.NewTransactionRepository(database)

	accountService := &service.AccountService{
		Repo: accountRepo,
	}

	transferService := &service.TransferService{
		AccountRepo:     accountRepo,
		TransactionRepo: transactionRepo,
	}

	mux := http.NewServeMux()

	accountHandler := handler.NewAccountHandler(mux, accountService, transferService)
	accountHandler.MapRoutes()

	transferRoutes := handler.NewTransferHandler(mux, transferService)
	transferRoutes.MapRoutes()

	timeoutMiddleware := middleware.Timeout(20 * time.Second) 

	handlerChain := middleware.Observability(
		server.ApplicationMiddlewareResponse(
			timeoutMiddleware(
				server.HandleRouteNotFound(mux),
			),
		),
	)

	http.ListenAndServe(":8080", handlerChain)

	log.Println("Server running on port 8080")
}