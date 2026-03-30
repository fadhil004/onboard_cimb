package main

import (
	"net/http"

	"rest-api-bank/db"
	"rest-api-bank/handler"
	"rest-api-bank/repository"
	"rest-api-bank/server"
	"rest-api-bank/service"
)

func main() {
	database := db.InitDB()

	// ✅ pakai constructor, bukan struct literal
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

	http.ListenAndServe(":8080",
		server.ApplicationMiddlewareResponse(
			server.HandleRouteNotFound(mux),
		),
	)
}