package main

import (
	"database-exercise/config"
	"database-exercise/handler"
	"database-exercise/repository"
	"database-exercise/service"
	"log"
	"net/http"
)

func main() {

	// INIT DB
	config.InitSQLX()
	// config.InitGORM()

	// PILIH REPOSITORY
	repo := repository.NewUserSQLX(config.DB)
	// repo := repository.NewUserGORM(config.GORM)

	// DEPENDENCY INJECTION
	userService := service.NewUserService(repo)
	userHandler := handler.NewUserHandler(userService)

	// ROUTES
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			userHandler.GetUsers(w, r)
		} else if r.Method == "POST" {
			userHandler.CreateUser(w, r)
		}
	})

	http.HandleFunc("/user", userHandler.GetUser)

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}