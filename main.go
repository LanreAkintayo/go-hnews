package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golangcollege/sessions"
	_ "github.com/mattn/go-sqlite3"
)

type application struct {
	errorLog    *log.Logger
	infoLog     *log.Logger
	userRepo    UserRepository
	postRepo    PostRepository
	templateDir string
	publicPath  string
	tp          *TemplateRenderer
	session     *sessions.Session
}

func main() {

	db, err := connectToDatabase("users.db")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	session := sessions.New([]byte("secretkey"))
	session.Lifetime = 24 * time.Hour
	session.Secure = true
	session.SameSite = http.SameSiteLaxMode

	app := &application{
		errorLog:    log.New(os.Stderr, "ERROR\t", log.Ltime|log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
		infoLog:     log.New(os.Stdout, "INFO\t", log.Ltime|log.LstdFlags),
		userRepo:    NewSQLUserRepository(db),
		postRepo:    NewSQLPostRepository(db),
		templateDir: "./templates",
		publicPath:  "./public",
		session:     session,
	}
	app.tp = NewTemplateRenderer(app.templateDir, true)

	fmt.Println("Starting server on :8080")
	err = app.serve()
	if err != nil {
		log.Fatal(err)
	}

}

func connectToDatabase(dbName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
