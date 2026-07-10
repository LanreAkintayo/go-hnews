package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	defaultMiddleware := alice.New(app.recover, app.logger)
	secureMiddleware := alice.New(app.session.Enable, app.authenticate)

	fileServer := http.FileServer(http.Dir(app.publicPath))
	mux.Handle("/public/", http.StripPrefix("/public/", fileServer))
	mux.Handle("/", secureMiddleware.ThenFunc(app.home))
	mux.Handle("/about", secureMiddleware.ThenFunc(app.about))
	mux.Handle("/contact", secureMiddleware.ThenFunc(app.contact))
	mux.Handle("/login", secureMiddleware.ThenFunc(app.login))
	mux.Handle("/logout", secureMiddleware.ThenFunc(app.logout))
	mux.Handle("/register", secureMiddleware.ThenFunc(app.register))
	mux.Handle("/submit", secureMiddleware.Append(app.requireAuth).ThenFunc(app.submit))
	mux.Handle("/vote", secureMiddleware.Append(app.requireAuth).ThenFunc(app.vote))
	mux.Handle("/comments", secureMiddleware.Append(app.requireAuth).ThenFunc(app.comments))

	handler := defaultMiddleware.Then(mux)

	return handler
}
