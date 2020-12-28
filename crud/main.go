package main

import (
	"database/sql"
	"encoding/gob"
	"golang.org/x/crypto/bcrypt"
	"log"
	"context"
	"net/http"
	"text/template"
	"github.com/go-redis/redis/v8"
	_ "github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/rbcervilla/redisstore/v8"
	_ "github.com/go-sql-driver/mysql"
)
var store *redisstore.RedisStore
var tmpl * template.Template
func init() {
	 tmpl = template.Must(template.ParseGlob("/Users/local/go/src/github.com/GoWebApp/crud/form/*"))
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
		Domain: "http://localhost:8083",
		MaxAge: 5000,
	})

	gob.Register(SessionUser{})

}
type Employee struct {
	Id    int
	Name  string
	City string
}

type User struct {
	Id    int
	Name  string
	PassWord string
}

type SessionUser struct {
	Id    int
	Name  string
	Authenticated bool
}

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := "password"
	dbName := "ToDo"
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}


func List(w http.ResponseWriter, r *http.Request) {
	 CheckValidSession(w,r);
	db := dbConn()
	selDB, err := db.Query("SELECT * FROM Employee ORDER BY id DESC")
	if err != nil {
		panic(err.Error())
	}
	emp := Employee{}
	res := []Employee{}
	for selDB.Next() {
		var id int
		var name, city string
		err = selDB.Scan(&id, &name, &city)
		if err != nil {
			panic(err.Error())
		}
		emp.Id = id
		emp.Name = name
		emp.City = city
		res = append(res, emp)
	}
	tmpl.ExecuteTemplate(w, "Index", res)
	defer db.Close()
}

func Show(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	db := dbConn()
	nId := r.URL.Query().Get("id")
	selDB, err := db.Query("SELECT * FROM Employee WHERE id=?", nId)
	if err != nil {
		panic(err.Error())
	}
	emp := Employee{}
	for selDB.Next() {
		var id int
		var name, city string
		err = selDB.Scan(&id, &name, &city)
		if err != nil {
			panic(err.Error())
		}
		emp.Id = id
		emp.Name = name
		emp.City = city
	}
	tmpl.ExecuteTemplate(w, "Show", emp)
	defer db.Close()
}

func New(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	tmpl.ExecuteTemplate(w, "New", nil)
}

func Edit(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	db := dbConn()
	nId := r.URL.Query().Get("id")
	selDB, err := db.Query("SELECT * FROM Employee WHERE id=?", nId)
	if err != nil {
		panic(err.Error())
	}
	emp := Employee{}
	for selDB.Next() {
		var id int
		var name, city string
		err = selDB.Scan(&id, &name, &city)
		if err != nil {
			panic(err.Error())
		}
		emp.Id = id
		emp.Name = name
		emp.City = city
	}
	tmpl.ExecuteTemplate(w, "Edit", emp)
	defer db.Close()
}

func Insert(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	db := dbConn()
	if r.Method == "POST" {
		name := r.FormValue("name")
		city := r.FormValue("city")
		insForm, err := db.Prepare("INSERT INTO Employee(name, city) VALUES(?,?)")
		if err != nil {
			panic(err.Error())
		}
		insForm.Exec(name, city)
		log.Println("INSERT: Name: " + name + " | City: " + city)
	}
	defer db.Close()
	http.Redirect(w, r, "/", 301)
}

func Update(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	db := dbConn()
	if r.Method == "POST" {
		name := r.FormValue("name")
		city := r.FormValue("city")
		id := r.FormValue("uid")
		insForm, err := db.Prepare("UPDATE Employee SET name=?, city=? WHERE id=?")
		if err != nil {
			panic(err.Error())
		}
		insForm.Exec(name, city, id)
		log.Println("UPDATE: Name: " + name + " | City: " + city)
	}
	defer db.Close()
	http.Redirect(w, r, "/index", 301)
}

func Delete(w http.ResponseWriter, r *http.Request) {
	CheckValidSession(w,r);
	db := dbConn()
	emp := r.URL.Query().Get("id")
	delForm, err := db.Prepare("DELETE FROM Employee WHERE id=?")
	if err != nil {
		panic(err.Error())
	}
	delForm.Exec(emp)
	log.Println("DELETE")
	defer db.Close()
	http.Redirect(w, r, "/", 301)
}

func main() {
	log.Println("Server started on: http://localhost:8080")
	http.HandleFunc("/", Index)
	http.HandleFunc("/index", Menu)
	http.HandleFunc("/signup", SignUp)
	http.HandleFunc("/log", LoginCheck)
	http.HandleFunc("/login", Login)
	http.HandleFunc("/list", List)
	http.HandleFunc("/show", Show)
	http.HandleFunc("/new", New)
	http.HandleFunc("/edit", Edit)
	http.HandleFunc("/insert", Insert)
	http.HandleFunc("/update", Update)
	http.HandleFunc("/delete", Delete)
	http.ListenAndServe(":8083", nil)
}

func Menu(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "Menu", map[string]interface{}{
		"key": "SignUp to Website",
	})

}
func Index(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "SignUp", map[string]interface{}{
		"key": "SignUp to Website",
	})

}

func LoginCheck(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "Login", map[string]interface{}{
		"key": "SignUp to Website",
	})

}

func SignUp(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	if r.Method == "POST" {
		name := r.FormValue("uname")
		pwd := r.FormValue("psw")
		password, _ := HashPassword(pwd)
		insForm, err := db.Prepare("INSERT INTO User(name, password) VALUES(?,?)")
		if err != nil {
			panic(err.Error())
		}
		insForm.Exec(name, password)
		log.Println("INSERT: Name: " + name + " | City: " + password)
	}
	defer db.Close()
	http.Redirect(w, r, "/log", 301)

}


func Login(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	if r.Method == "POST" {
		name := r.FormValue("uname")
		password := r.FormValue("psw")
		selDB, err := db.Query("SELECT * FROM USER WHERE name=?", name)
		if err != nil {
			panic(err.Error())
		}
		emp := User{}
		for selDB.Next() {
			var id int
			var name, password string
			err = selDB.Scan(&id, &name, &password)
			if err != nil {
				panic(err.Error())
			}
			emp.Id = id
			emp.Name = name
			emp.PassWord = password
			break
		}
		isequal := CheckPasswordHash(password,emp.PassWord)
		session, err := store.Get(r, "cookie-name")
		if isequal {
			session.Values["user"] =  &SessionUser{
				Name: emp.Name,
				Id:   emp.Id,
				Authenticated: true,
			}
				err := session.Save(r, w)
				if err != nil {
					log.Println("session not saved: " + err.Error())
					http.Redirect(w, r, "/login", 301)
				}
				log.Println("session save succeed for user: " + session.ID, emp.Id)
				http.Redirect(w, r, "/list", 301)
			}else {
			http.Redirect(w, r, "/login", 301)
			log.Println("Invalid Login " + name + " | City: " + password)
			}

		log.Println("INSERT: Name: " + name + " | City: " + password)
		}

	defer db.Close()


}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func getUser(s *sessions.Session) SessionUser {
	val := s.Values["user"]
	var user = SessionUser{}
	user, ok := val.(SessionUser)
	if !ok {
		return SessionUser{Authenticated: false}
	}
	return user
}
func CheckValidSession(w http.ResponseWriter, r *http.Request)  {

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

}
//Reference : https://www.golangprograms.com/example-of-golang-crud-using-mysql-from-scratch.html
