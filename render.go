package main

import (
	"net/http"
)

func (app *application) render(w http.ResponseWriter, r *http.Request, fileName string, data *templateData) {
	if app.tp == nil {
		http.Error(w, "Template renderer not initialized", http.StatusInternalServerError)
		return
	}

	mergedData := app.defaultTemplateData(data, r)
	app.tp.Render(w, fileName, mergedData)

}

func (app *application) defaultTemplateData(data *templateData, r *http.Request) *templateData {
	if data == nil {
		data = &templateData{}
	}

	data.Flash = app.session.PopString(r, "flash")
	data.IsAuthenticated = app.isAuthenticated(r)
	

	return data

}
