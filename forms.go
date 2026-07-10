package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)


type formErrors map[string][]string

var EmailRX = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (e formErrors) Add(field, message string) {
	e[field] = append(e[field], message)
}

func (e formErrors) Get(field string) string {
	if err, ok := e[field]; ok {
		return err[0]
	}
	return ""
}

type Form struct {
	url.Values
	Errors formErrors
}


func NewForm(data url.Values) *Form {
	return &Form{
		Values: data,
		Errors: make(formErrors),
	}
}

func (f *Form) Required(fields ...string) *Form{
	for _, field := range fields {
		value := f.Get(field)
		if strings.TrimSpace(value) == "" {
			f.Errors.Add(field, fmt.Sprintf("This field %s cannot be blank", field))
		}
	}

	return f
}

func (f *Form) Valid() bool{
	return len(f.Errors) == 0
}

func (f *Form) MaxLength(field string, count int) *Form {
	value := f.Get(field)
	if value == ""{
		return f
	}
	if utf8.RuneCountInString(value) > count {
		f.Errors.Add(field, fmt.Sprintf("This field %s is too long (maximum of %d characters)", field, count))
	}

	return f
}

func (f *Form) MinLength(field string, count int) *Form {
	value := f.Get(field)
	if value == ""{
		return f
	}
	if utf8.RuneCountInString(value) < count {
		f.Errors.Add(field, fmt.Sprintf("This field %s is too short (minimum of %d characters)", field, count))
	}

	return f
}

func (f *Form) Matches(field string, pattern *regexp.Regexp) *Form {
	value := f.Get(field)
	if value == ""{
		return f
	}
	if !pattern.MatchString(value) {
		f.Errors.Add(field, "This field does not match the pattern")
	}

	return f
}

func (f *Form) IsEmail(field string) *Form {
	value := f.Get(field)
	if value == ""{
		return f
	}

	if !EmailRX.MatchString(value){
		f.Errors.Add(field, "Please enter a valid email address")
	}
	return f
}