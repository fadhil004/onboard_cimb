package handler

import (
	"database-exercise/service"
	"encoding/json"
	"net/http"
	"strconv"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// GET /users
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {

	users, err := h.service.GetAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// GET /users?id=1
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	user, err := h.service.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// POST /users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {

	var body map[string]string

	json.NewDecoder(r.Body).Decode(&body)

	err := h.service.Create(body["name"], body["email"])
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	w.Write([]byte("User Created"))
}