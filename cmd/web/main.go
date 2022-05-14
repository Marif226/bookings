package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/marif226/bookings/internal/config"
	"github.com/marif226/bookings/internal/handlers"
	"github.com/marif226/bookings/internal/helpers"
	"github.com/marif226/bookings/internal/models"
	"github.com/marif226/bookings/internal/render"
)

const portNumber = ":8080"
var app 		config.AppConfig
var session 	*scs.SessionManager
var infoLog 	*log.Logger
var errorLog 	*log.Logger

// Main application function
func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}

	serv := &http.Server{
		Addr: portNumber,
		Handler: routes(&app),
	}

	fmt.Printf("Starting application on port %s\n", portNumber)
	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// what am i going to put in the session

	gob.Register(models.Reservation{})

	// change to true when in production
	app.InProduction = false

	// set up loggers
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session

	// create template cache
	templateCache, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("Cannot create template cache: ", err)
		return err
	}

	// store template cache in application
	app.TemplateCache = templateCache
	app.UseCache = false

	// create new repository that holds app config
	repo := handlers.NewRepo(&app)
	// set this repository for handlers package
	handlers.NewHandlers(repo)

	// set app config for render package
	render.NewTemplates(&app)

	helpers.NewHelpers(&app)

	return nil
}