package render

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/justinas/nosurf"
	"github.com/marif226/bookings/internal/config"
	"github.com/marif226/bookings/internal/models"
)

var functions = template.FuncMap {
	"humanDate": HumanDate,
}

var app *config.AppConfig
var pathToTemplates = "./templates"

// NewRenderer sets the config for the template package
func NewRenderer(a *config.AppConfig) {
	app = a
}

// HumanDate returns time in dd-mm-yyyy format
func HumanDate(t time.Time) string {
	return t.Format("02-01-2006")
}

// AddDefaultData adds data for all templates
func AddDefaultData(templData *models.TemplateData, r *http.Request) *models.TemplateData {
	templData.Flash = app.Session.PopString(r.Context(), "flash")
	templData.Error = app.Session.PopString(r.Context(), "error")
	templData.Warning = app.Session.PopString(r.Context(), "warning")
	templData.CSRFToken = nosurf.Token(r)
	if app.Session.Exists(r.Context(), "user_id") {
		templData.IsAuthenticated = 1
	}
	return templData
}

// Template renders templates using html/template
func Template(w http.ResponseWriter, r *http.Request, tmpl string, tmplData *models.TemplateData) error {
	var templateCache map[string]*template.Template
	if app.UseCache {
		// get the template cache from the app config
		templateCache = app.TemplateCache
	} else {
		templateCache, _ = CreateTemplateCache()
	}


	t, ok := templateCache[tmpl]
	if !ok {
		return errors.New("can't get template from cache")
	}

	// holds bytes
	buf := new(bytes.Buffer)

	tmplData = AddDefaultData(tmplData, r)

	// store executed template in buf
	_ = t.Execute(buf, tmplData)

	_, err := buf.WriteTo(w)
	if err != nil {
		fmt.Println("Error writing template to browser: ", err)
		return err
	}

	return nil
}

// CreateTemplateCache creates template cache as a map
func CreateTemplateCache() (map[string]*template.Template, error) {
	// myCache holds all templates created at the start of the application
	myCache := map[string]*template.Template{}

	// get all *.page.html files in templates directory
	pages, err := filepath.Glob(fmt.Sprintf("%s/*.page.html", pathToTemplates))
	if err != nil {
		return myCache, err
	}

	for _, page := range pages {
		// extract actual name of the file from its path
		name := filepath.Base(page)

		templSet, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return myCache, err
		}

		// search for layout pages
		matches, err := filepath.Glob(fmt.Sprintf("%s/*.layout.html", pathToTemplates))
		if err != nil {
			return myCache, err
		}

		if len(matches) > 0 {
			//parse layout template
			templSet, err = templSet.ParseGlob(fmt.Sprintf("%s/*.layout.html", pathToTemplates))
			if err != nil {
				return myCache, err
			}
		}
		
		// hold parsed template in the cache
		myCache[name] = templSet
	}

	return myCache, nil
}