package main

import (
	"context"
	"encoding/gob"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/rbcervilla/redisstore/v8"
	"html/template"
	"io/ioutil"
	"net/http"
	"github.com/rs/cors"
	"github.com/gorilla/mux"
	_ "github.com/gorilla/securecookie"
	"log"
)

// User holds a users account information
type User struct {
	Username      string
	Authenticated bool
}

// store will hold all session data
var store *redisstore.RedisStore

// tpl holds all parsed templates
var tpl *template.Template

func init() {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
	})
	var err error
	// New default RedisStore
	store, err = redisstore.NewRedisStore(context.Background(), client)
	if err != nil {
		log.Fatal("failed to create redis store: ", err)
	}

	// Example changing configuration for sessions
	store.KeyPrefix("session_")
	store.Options(sessions.Options{
		Path:   "/",
		Domain: "http://localhost",
		MaxAge: 30,
	})


	//authKeyOne := securecookie.GenerateRandomKey(64)
	//encryptionKeyOne := securecookie.GenerateRandomKey(32)
	//
	//store = sessions.NewCookieStore(
	//	authKeyOne,
	//	encryptionKeyOne,
	//)
	//
	//store.Options = &sessions.Options{
	//	Path:     "/",
	//	MaxAge:   60 * 15,
	//	HttpOnly: true,
	//}

	gob.Register(User{})

	tpl = template.Must(template.ParseGlob("/Users/local/go/src/github.com/GoWebApp/RedisStore/templates/*.gohtml"))
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/login", login)
	router.HandleFunc("/logout", logout)
	router.HandleFunc("/forbidden", forbidden)
	router.HandleFunc("/secret", secret)
	router.HandleFunc("/data", DataGet)
	router.HandleFunc("/cors", CorsExample)
	//http.ListenAndServe(":8002",
	//	csrf.Protect([]byte("32-byte-long-auth-key"),csrf.Secure(false))(router))
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://foo.com", "http://foo.com:8080"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})

	http.ListenAndServe(":8002", c.Handler(router))
}

// index serves the index html file
func index(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if session.IsNew == false {

		user := getUser(session)
		tpl.ExecuteTemplate(w, "home.gohtml", user)
	} else {
		w.Header().Set("X-CSRF-Token", csrf.Token(r))
		user := getUser(session)
		tpl.ExecuteTemplate(w, "login.gohtml",  map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"user" : user,
		})
	}
}

// login authenticates the user
func login(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("psw") != "code" {
		if r.FormValue("code") == "" {
			session.AddFlash("Must enter a code")
		}
		session.AddFlash("The code was incorrect")
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/forbidden", http.StatusFound)
		return
	}

	username := r.FormValue("uname")

	user := &User{
		Username:      username,
		Authenticated: true,
	}

	session.Values["user"] = user

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/secret", http.StatusFound)
}

// logout revokes authentication for a user
func logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["user"] = User{}
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// secret displays the secret message for authorized users
func secret(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := getUser(session)

	if auth := user.Authenticated; !auth {
		session.AddFlash("You don't have access!")
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/forbidden", http.StatusFound)
		return
	}

	tpl.ExecuteTemplate(w, "secret.gohtml", user.Username)
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	flashMessages := session.Flashes()
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tpl.ExecuteTemplate(w, "forbidden.gohtml", flashMessages)
}

// getUser returns a user from session s
// on error returns an empty user
func getUser(s *sessions.Session) User {
	val := s.Values["user"]
	var user = User{}
	user, ok := val.(User)
	if !ok {
		return User{Authenticated: false}
	}
	return user
}

// secret displays the secret message for authorized users
func DataGet(w http.ResponseWriter, r *http.Request) {
	resp,err := http.Get("http://localhost:8200/data");
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		//w.Write(bodyBytes)
		bodyString := string(bodyBytes)
		merge :=bodyString + " This is from other service";

		w.Write([]byte(merge))
	}

}

func CorsExample(w http.ResponseWriter, r *http.Request) {

	tpl.ExecuteTemplate(w, "test.gohtml", nil)
}
