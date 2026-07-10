package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewForm(t *testing.T) {
	values := url.Values{}
	values.Add("email", "test@test.com")

	form := NewForm(values)
	assert.NotNil(t, form)
	assert.Equal(t, "test@test.com", form.Get("email"))
	assert.NotNil(t, form.Errors)
	assert.Len(t, form.Errors, 0)
}

func TestForm_Required(t *testing.T){
	values := url.Values{}
	values.Add("email", "test@test.com")
	values.Add("empty", "  ")

	form := NewForm(values)
	form.Required("email", "password", "empty")

	assert.NotNil(t, form)
	assert.Equal(t, "", form.Errors.Get("email"))
	assert.Contains(t, form.Errors.Get("password"), "cannot be blank")
	assert.Contains(t, form.Errors.Get("empty"), "cannot be blank")
}