package handler

import (
	"net/http"

	"github.com/GoWebApp/internal/middleware"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "hello, " + name})
}

func WhoAmI(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"user": user})
}
