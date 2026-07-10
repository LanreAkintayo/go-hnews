package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type contextKey string

const (
	contextAuthKey contextKey = "isAuthKey"
	contextUserKey contextKey = "userKey"
)

func (app *application) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (app *application) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s \n -----------------------------", err))

			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			http.Redirect(w, r, fmt.Sprintf("/login?redirectTo=%s", r.URL.Path), http.StatusSeeOther)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := app.session.Exists(r, loggedInUserKey)
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		user, err := app.userRepo.GetUserByEmail(app.session.GetString(r, loggedInUserKey))
		if errors.Is(err, sql.ErrNoRows) {
			app.session.Remove(r, loggedInUserKey)
			next.ServeHTTP(w, r)
			return
		}

		if err != nil {
			app.serverError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextAuthKey, true)
		ctx = context.WithValue(ctx, contextUserKey, user)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuth, ok := r.Context().Value(contextAuthKey).(bool)
	if !ok {
		return false
	}
	return isAuth
}
