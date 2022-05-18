package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/marif226/bookings/internal/config"
	"github.com/marif226/bookings/internal/driver"
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
	db, err := run()
	if err != nil {
		log.Fatal(err)
	}

	defer db.SQL.Close()

	defer close(app.MailChan)

	fmt.Println("Starting mail lestener...")
	listenToMail()

	fmt.Printf("Starting application on port %s\n", portNumber)

	from := "me@here.com"
	auth := smtp.PlainAuth("", from, "", "localhost")
	err = smtp.SendMail("localhost:1025", auth, from, []string{"you@there.com"}, []byte("Hello, world!"))
	if err != nil {
		log.Println(err)
	}	

	serv := &http.Server{
		Addr: portNumber,
		Handler: routes(&app),
	}

	
	err = serv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func run() (*driver.DB, error) {
	// what am i going to put in the session

	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Restriction{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan


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

	// connect to database
	log.Println("Connecting to database...")
	db, err := driver.ConnectSQL("host=localhost port=5432 dbname=bookings user=postgres password=minecraft132")
	if err != nil {
		log.Fatal("Cannot connect to database! Dying...")
	}

	log.Println("Connected to database!")

	// create template cache
	templateCache, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("Cannot create template cache: ", err)
		return nil, err
	}

	// store template cache in application
	app.TemplateCache = templateCache
	app.UseCache = false

	// create new repository that holds app config
	repo := handlers.NewRepo(&app, db)
	// set this repository for handlers package
	handlers.NewHandlers(repo)

	// set app config for render package
	render.NewRenderer(&app)

	helpers.NewHelpers(&app)

	return db, nil
}