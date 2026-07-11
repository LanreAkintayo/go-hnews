package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogin_GET_NotAuthenticated(t *testing.T) {
	// We will need to enable the session
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, testApp.isAuthenticated(req))

}

func TestLogin_GET_AlreadyAuthenticated(t *testing.T) {
	defer cleanupTestData(t)

	// Create a user in the database to authenticate against
	_, err := testApp.userRepo.CreateUser(
		"testuser",
		"test@example.com",
		"password",
		"avatar.png",
	)
	assert.NoError(t, err)

	// Generate a valid session cookie for the user
	setupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testApp.session.Put(r, loggedInUserKey, "test@example.com")
	})

	setupReq := httptest.NewRequest(http.MethodGet, "/setup", nil)
	setupRes := httptest.NewRecorder()
	testApp.session.Enable(setupHandler).ServeHTTP(setupRes, setupReq)

	cookies := setupRes.Result().Cookies()
	assert.NotEmpty(t, cookies)

	// Perform the actual GET request to /login, attaching the session cookie
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()

	// Run the request through the session, authenticate, and login chain
	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	handler.ServeHTTP(res, req)

	// Assert that authenticated users are redirected back to the homepage
	assert.Equal(t, http.StatusSeeOther, res.Code)
	assert.Equal(t, "/", res.Header().Get("Location"))
}


func TestLogin_POST_ValidCredentials(t *testing.T) {

	defer cleanupTestData(t)

	_, err := testApp.userRepo.CreateUser(
		"login",
		"login@test.com",
		"goodpassword",
		"avatar",
	)
	assert.NoError(t, err)

	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	formData := "email=login@test.com&password=goodpassword"
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/submit", w.Header().Get("Location"))

}

func TestLogin_POST_InvalidFormData(t *testing.T) {
	defer cleanupTestData(t)

	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	formData := "email=&password="
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "The data you submitted is not valid")
	assert.Contains(t, body, "This field email cannot be blank")
	assert.Contains(t, body, "This field password cannot be blank")
}

func TestLogin_POST_InvalidAuthenticationData(t *testing.T) {
	defer cleanupTestData(t)

	handler := testApp.session.Enable(testApp.authenticate(http.HandlerFunc(testApp.login)))
	formData := "email=test@test.com&password=goodpassword"
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, "Invalid credentials")

}

