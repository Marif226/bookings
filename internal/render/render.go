package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/justinas/nosurf"
	"github.com/marif226/bookings/pkg/config"
	"github.com/marif226/bookings/pkg/models"
)

var functions = template.FuncMap {

}

var app *config.AppConfig

// NewTemplates sets the config for the template package
func NewTemplates(a *config.AppConfig) {
	app = a
}

func AddDefaultData(templData *models.TemplateData, r *http.Request) *models.TemplateData {
	templData.CSRFToken = nosurf.Token(r)

	return templData
}

func RenderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, tmplData *models.TemplateData) {
	var templateCache map[string]*template.Template
	if app.UseCache {
		// get the template cache from the app config
		templateCache = app.TemplateCache
	} else {
		templateCache, _ = CreateTemplateCache()
	}


	t, ok := templateCache[tmpl]
	if !ok {
		log.Fatal("Could not get template from template cache")
	}

	// holds bytes
	buf := new(bytes.Buffer)

	tmplData = AddDefaultData(tmplData, r)

	// store executed template in buf
	_ = t.Execute(buf, tmplData)

	_, err := buf.WriteTo(w)
	if err != nil {
		fmt.Println("Error writing template to browser: ", err)
	}
}

// CreateTemplateCache creates template cache as a map
func CreateTemplateCache() (map[string]*template.Template, error) {
	// myCache holds all templates created at the start of the application
	myCache := map[string]*template.Template{}

	// get all *.page.html files in templates directory
	pages, err := filepath.Glob("./templates/*.page.html")
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
		matches, err := filepath.Glob("./templates/*.layout.html")
		if err != nil {
			return myCache, err
		}

		if len(matches) > 0 {
			//parse layout template
			templSet, err = templSet.ParseGlob("./templates/*.layout.html")
			if err != nil {
				return myCache, err
			}
		}
		
		// hold parsed template in the cache
		myCache[name] = templSet
	}

	return myCache, nil
}