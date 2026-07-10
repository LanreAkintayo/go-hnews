package main

import (
	"net/http"
	"time"
)


func (app *application) serve() error{
	s := &http.Server{
		Addr: ":8080",
		Handler: app.routes(),
		ReadTimeout: 10*time.Second,
		WriteTimeout: 10*time.Second,
		IdleTimeout: 60*time.Second,
	}

	return s.ListenAndServe()
}