package main

import (
	"net/http"
	"user-management/config"
	"user-management/handler"
	"user-management/middleware"
	"user-management/repository"
	"user-management/service"
)

func main() {
	database := config.InitDB()

	repo := &repository.UserRepository{DB: database}
	service := &service.UserService{Repo: repo}
	handler := &handler.UserHandler{Service: service}

	mux := http.NewServeMux()
	mux.HandleFunc("/users", handler.Users)
	mux.HandleFunc("/users/", handler.UserByID)

	wrapped := middleware.Logging(mux)

	http.ListenAndServe(":8080", wrapped)
}