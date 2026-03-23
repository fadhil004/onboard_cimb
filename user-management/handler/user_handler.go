package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"user-management/models"
	"user-management/service"
)

type UserHandler struct {
	Service *service.UserService
}

func (h *UserHandler) Users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		users, _ := h.Service.GetUsers()
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		var u models.User
		json.NewDecoder(r.Body).Decode(&u)
		h.Service.CreateUser(u)
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UserHandler) UserByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, _ := strconv.Atoi(idStr)

	switch r.Method {

	case http.MethodGet:
		user, _ := h.Service.GetUser(id)
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		var u models.User
		json.NewDecoder(r.Body).Decode(&u)
		u.ID = id
		h.Service.UpdateUser(u)

	case http.MethodDelete:
		h.Service.DeleteUser(id)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}