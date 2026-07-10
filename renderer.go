package main

import (
	"bytes"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"sync"
)

type TemplateRenderer struct {
	cache       map[string]*template.Template
	mutex       sync.RWMutex
	dev         bool
	templateDir string
}

type templateData struct {
	Form            *Form
	Posts           []Post
	Metadata        Metadata
	IsAuthenticated bool
	Flash           string
	Comments        []Comment
	Post            *Post
	NextLink        string
	PrevLink        string
}

func NewTemplateRenderer(templateDir string, isDev bool) *TemplateRenderer {
	return &TemplateRenderer{
		cache:       make(map[string]*template.Template),
		dev:         isDev,
		templateDir: templateDir,
	}
}

func (t *TemplateRenderer) Render(w http.ResponseWriter, templateName string, data interface{}) {
	tmpl, err := t.getTemplate(templateName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write to a buffer first so errors don't corrupt the response
	buf := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(buf, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// No errors, so now it's safe to write to the browser
	buf.WriteTo(w)
}

func (t *TemplateRenderer) getTemplate(templateName string) (*template.Template, error) {
	// On production, try to get template from cache
	if !t.dev {
		t.mutex.RLock()
		tmpl, ok := t.cache[templateName]
		if ok {
			t.mutex.RUnlock()
			return tmpl, nil
		}
		t.mutex.RUnlock()
	}

	// Build path to template file
	tmpl, err := t.parseTemplate(templateName)
	if err != nil {
		return nil, err
	}

	if !t.dev {
		t.mutex.Lock()
		t.cache[templateName] = tmpl
		t.mutex.Unlock()
	}

	return tmpl, nil

}

func (t *TemplateRenderer) parseTemplate(templateName string) (*template.Template, error) {
	templatePath := path.Join(t.templateDir, templateName)

	files := []string{templatePath}

	layoutPath := path.Join(t.templateDir, "layouts/*.html")
	layouts, err := filepath.Glob(layoutPath)
	if err == nil {
		files = append(files, layouts...)
	}

	partialPath := path.Join(t.templateDir, "partials/*.html")
	partials, err := filepath.Glob(partialPath)
	if err == nil {
		files = append(files, partials...)
	}

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
