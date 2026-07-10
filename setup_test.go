package main

import (
	"database/sql"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golangcollege/sessions"
	"github.com/stretchr/testify/assert"
)

var testDB *sql.DB
var testApp *application

func TestMain(m *testing.M) {
	// Set up the database and the schemas
	var err error

	// Prepare a connection pool with sqlite3
	testDB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Ping the database to be sure of the connection
	if err := testDB.Ping(); err != nil {
		panic(err)
	}

	// Setup applications
	testApp = setupApp(testDB)

	// Setup schemas
	if err := setupTestSchema(testDB); err != nil {
		panic(err)
	}

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

func setupApp(db *sql.DB) *application {

	sess := sessions.New([]byte("super-secret-key"))
	sess.Lifetime = 24 * time.Hour

	return &application{
		userRepo: NewSQLUserRepository(db),
		postRepo: NewSQLPostRepository(db),
		session: sess,
		errorLog: log.New(io.Discard, "",0),
		infoLog: log.New(io.Discard, "", 0),
		templateDir: "./templates",
		publicPath: "./public",
	}

}

func setupTestSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE users (
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   name TEXT NOT NULL,
   email TEXT NOT NULL UNIQUE,
   hashed_password TEXT NOT NULL,
   created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE profile (
     user_id INTEGER PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
     avatar TEXT NOT NULL,
     created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE posts (
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   url TEXT NOT NULL,
   title TEXT NOT NULL UNIQUE,
   user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
   created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  body TEXT NOT NULL,
  user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
  post_id INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE votes (
   user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
   post_id INTEGER REFERENCES posts(post_id) ON DELETE CASCADE,
   created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (user_id, post_id)
);

	`
	_, err := db.Exec(schema)
	return err
}

func cleanupTestData(t *testing.T) {
	tables := []string{
		"profile",
		"votes",
		"comments",
		"posts",
		"users",
	}
	for _, table := range tables {
		_, err := testDB.Exec("DELETE FROM " + table)
		assert.NoError(t, err)
	}

}
