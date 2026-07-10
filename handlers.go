package main

// import "fmt"
import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
)

var htmlContent = `
	<!DOCTYPE html>
	<html>
	<head><title>%s</title></head>
	<body>
	%s
	</body>
	</html>
`

const (
	loggedInUserKey = "logged_in_user"
)

func (app *application) readIntWithDefault(r *http.Request, key string, def int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return value
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {

	filter := Filter{
		Query:    r.URL.Query().Get("q"),
		OrderBy:  r.URL.Query().Get("order_by"),
		Page:     app.readIntWithDefault(r, "page", 1),
		PageSize: app.readIntWithDefault(r, "page_size", 50),
	}

	posts, metadata, err := app.postRepo.GetAll(filter)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	app.infoLog.Printf("\nMetadata: %+v\n", metadata)

	app.render(w, r, "index.html", &templateData{
		Posts:    posts,
		Metadata: metadata,
		NextLink: fmt.Sprintf("/?q=%s&order_by=%s&page=%d&page_size=%d",
			filter.Query, filter.OrderBy, metadata.NextPage, filter.PageSize),
		PrevLink: fmt.Sprintf("/?q=%s&order_by=%s&page=%d&page_size=%d",
			filter.Query, filter.OrderBy, metadata.PrevPage, filter.PageSize),
	})
}
func (app *application) login(w http.ResponseWriter, r *http.Request) {
	if app.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.infoLog.Printf("Session key: %s", app.session.GetString(r, loggedInUserKey))
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		form := NewForm(r.PostForm)
		form.Required("email", "password").
			MaxLength("email", 100).
			MaxLength("password", 100).
			MinLength("password", 6).
			MinLength("email", 3).
			IsEmail("email")

		if !form.Valid() {
			form.Errors.Add("generic", "The data you submitted is not valid")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "login.html", &templateData{Form: form})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		_, err := app.userRepo.Authenticate(email, password)
		if err != nil {
			form.Errors.Add("generic", "Invalid credentials")
			app.render(w, r, "login.html", &templateData{Form: form})
			return
		}

		// We put the userID in session.
		app.session.Put(r, loggedInUserKey, email)
		app.session.Put(r, "flash", "You are logged in")

		app.infoLog.Printf("User %s logged in", email)

		redirectURL := r.FormValue("redirectTo")
		if redirectURL == "" {
			redirectURL = "/submit"
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	app.render(w, r, "login.html", &templateData{Form: NewForm(r.PostForm)})
}

func (app *application) logout(w http.ResponseWriter, r *http.Request) {

	app.session.Remove(r, loggedInUserKey)
	app.session.Put(r, "flash", "You are logged out")

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func (app *application) register(w http.ResponseWriter, r *http.Request) {
	if app.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		form := NewForm(r.PostForm)
		form.Required("name", "email", "password").
			MaxLength("name", 100).
			MaxLength("email", 100).
			MinLength("password", 6).
			MinLength("email", 3).
			IsEmail("email")
		if !form.Valid() {
			form.Errors.Add("generic", "The data you submitted is not valid")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "register.html", &templateData{Form: form})
			return
		}

		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")
		avatar := r.FormValue("avatar")

		_, err := app.userRepo.CreateUser(name, email, password, avatar)
		if err != nil {
			app.errorLog.Printf("Error creating user: %v", err.Error())
			form.Errors.Add("generic", "Failed to create account")
			app.render(w, r, "register.html", &templateData{Form: form})
			return
		}

		// app.session.Put(r, loggedInUserKey, userID)

		// app.infoLog.Printf("User %d registered", userID)
		app.session.Put(r, "flash", "You are registered")

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	app.render(w, r, "register.html", &templateData{Form: NewForm(r.PostForm)})
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "about.html", nil)
}

func (app *application) contact(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "contact.html", nil)
}
func (app *application) submit(w http.ResponseWriter, r *http.Request) {
	/*
		What I want to try and achieve in this function
		- If we are coming from post method
			- I want to parse the form
			- Wrap it in our own form
			- Extract the values from the form
			- Create a post with those values
			- If it is successful, then somehow activate the flash message
			- I think that's all
		- render the submit form and just parse the form stuff.
	*/

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		form := NewForm(r.PostForm)

		form.Required("url", "title").
			MinLength("title", 5).
			MaxLength("url", 2048).
			MaxLength("title", 255)

		if !form.Valid() {
			form.Errors.Add("generic", "The data you submitted is not valid")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "submit.html", &templateData{Form: form})
			return
		}

		title := r.FormValue("title")
		url := r.FormValue("url")
		user := app.getUserFromContext(r.Context())

		_, err := app.postRepo.CreatePost(title, url, user.ID)
		if err != nil {
			app.errorLog.Printf("Error creating post: %v", err.Error())
			form.Errors.Add("generic", "Failed to create post")
			app.render(w, r, "submit.html", &templateData{Form: form})
			return
		}

		app.infoLog.Println("Post created successfully")

		app.session.Put(r, "flash", "Post created successfully")

		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}
	app.render(w, r, "submit.html", &templateData{Form: NewForm(r.PostForm)})

}

func (app *application) vote(w http.ResponseWriter, r *http.Request) {
	/*
			type Vote struct {
			UserID    int       `json:"user_id"`
			PostID    int       `json:"post_id"`
			CreatedAt time.Time `json:"created_at"`
		}
	*/

	postId := app.readIntWithDefault(r, "post_id", 0)
	user := app.getUserFromContext(r.Context())

	err := app.postRepo.AddVote(user.ID, postId)
	if err != nil {
		app.errorLog.Printf("Error voting: %v", err.Error())
		app.session.Put(r, "flash", fmt.Sprintf("You may have already voted for post %d", postId))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.infoLog.Printf("Vote added successfully for %d", postId)

	app.session.Put(r, "flash", fmt.Sprintf("Vote added successfully for post %d", postId))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) comments(w http.ResponseWriter, r *http.Request) {
	postId := app.readIntWithDefault(r, "post_id", 0)

	post, err := app.postRepo.GetByID(postId)
	if err != nil {
		app.infoLog.Printf("No post found for %d", postId)
		app.session.Put(r, "flash", fmt.Sprintf("No post found for %d", postId))
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		form := NewForm(r.PostForm)
		form.Required("comment").
			MinLength("comment", 1).
			MaxLength("comment", 1000)

		if !form.Valid() {
			comments, _ := app.postRepo.GetComments(postId)
			form.Errors.Add("generic", "The data you submitted is not valid")
			app.errorLog.Printf("Invalid form: %+v", form.Errors)
			app.render(w, r, "comments.html", &templateData{Form: form, Post: post, Comments: comments})
			return
		}

		comment := r.FormValue("comment")
		user := app.getUserFromContext(r.Context())

		_, err = app.postRepo.AddComment(user.ID, postId, comment)
		if err != nil {
			app.errorLog.Printf("Error creating comment: %v", err.Error())
			form.Errors.Add("generic", "Failed to create comment")
			app.render(w, r, "comments.html", &templateData{Form: form, Post: post})
			return
		}

		app.infoLog.Printf("Comment added successfully for %d", postId)

		app.session.Put(r, "flash", fmt.Sprintf("Comment added successfully for post %d", postId))

		http.Redirect(w, r, fmt.Sprintf("/comments?post_id=%d", postId), http.StatusSeeOther)

		return
	}

	comments, err := app.postRepo.GetComments(postId)
	if err != nil {
		if err == sql.ErrNoRows {
			app.infoLog.Printf("No comments found for post %d", postId)
			app.session.Put(r, "flash", fmt.Sprintf("No comments found for post %d", postId))
		} else {
			app.errorLog.Printf("Error getting comments: %v", err.Error())
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	app.render(w, r, "comments.html", &templateData{Comments: comments, Post: post})

}
