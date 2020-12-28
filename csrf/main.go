package main

import (
	"encoding/json"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	//csrfMiddleware := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false))
	CSRF := csrf.Protect(
		[]byte("08SY058118B4DN7adZr5a77Omvp6v1vA"),
		csrf.FieldName("authenticity_token"),
		csrf.Secure(false),
		csrf.HttpOnly(false),
		csrf.Path("/"),
		csrf.MaxAge(12000),
	)
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/user/{id}", GetUser).Methods("GET")
	api.HandleFunc("/test", GetTest).Methods("POST")

	http.ListenAndServe(":8000", CSRF(r))

}

func GetUser(w http.ResponseWriter, r *http.Request) {
	// Authenticate the request, get the id from the route params,
	// and fetch the user from the DB, etc.

	// Get the token and pass it in the CSRF header. Our JSON-speaking client
	// or JavaScript framework can now read the header and return the token in
	// in its own "X-CSRF-Token" request header on the subsequent POST.
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	b, err := json.Marshal(struct {Number int; Text string}{42, "Hello world!"})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(b)
}
func GetTest(w http.ResponseWriter, r *http.Request) {

	b, err := json.Marshal(struct {Number int; Text string}{100, "Test Route!"})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(b)
}
