package main

import (
	"log"
	"net/http"
	"time"

	"rest-api-bank/config"
	"rest-api-bank/handler"
	"rest-api-bank/middleware"
	"rest-api-bank/repository"
	"rest-api-bank/server"
	"rest-api-bank/service"
)

func main() {
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

	handlerChain := server.ApplicationMiddlewareResponse(
						timeoutMiddleware(
								server.HandleRouteNotFound(mux),
					
							),
						)

	http.ListenAndServe(":8080", handlerChain)

	log.Println("Server running on port 8080")
}