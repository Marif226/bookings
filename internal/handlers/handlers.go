package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/marif226/bookings/internal/config"
	"github.com/marif226/bookings/internal/driver"
	"github.com/marif226/bookings/internal/forms"
	"github.com/marif226/bookings/internal/helpers"
	"github.com/marif226/bookings/internal/models"
	"github.com/marif226/bookings/internal/render"
	"github.com/marif226/bookings/internal/repository"
	"github.com/marif226/bookings/internal/repository/dbrepo"
)

// Repo the repositpry used by the handlers
var Repo *Repository

// Repository is the repository type
type Repository struct {
	App *config.AppConfig
	DB repository.DatabaseRepo
}

// NewRepo creates a new repository
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB: dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

// NewTestRepo creates a new repository for test
func NewTestRepo(a *config.AppConfig) *Repository {
	return &Repository{
		App: a,
		DB: dbrepo.NewTestingRepo(a),
	}
}

// NewHandlers sets the repository for the handlers
func NewHandlers(r *Repository) {
	Repo = r
}

// Home is the home page handler
func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	
	render.Template(w, r, "home.page.html", &models.TemplateData{})
}

// About is the about page handler
func (m *Repository) About(w http.ResponseWriter, r *http.Request) {
	// send the data to the template
	render.Template(w, r, "about.page.html", &models.TemplateData{})
}

// Generals renders the generals room page
func (m *Repository) Generals(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "generals.page.html", &models.TemplateData{})
}

// Majors renders the majors room page
func (m *Repository) Majors(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "majors.page.html", &models.TemplateData{})
}

// Availability renders the search availability room page
func (m *Repository) Availability(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "search-availability.page.html", &models.TemplateData{})
}

// Reservation renders the make a reservation page and displays form
func (m *Repository) Reservation(w http.ResponseWriter, r *http.Request) {
	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "cannot get reservation from session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	room, err := m.DB.GetRoomByID(res.RoomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot find room")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	res.Room.RoomName = room.RoomName

	m.App.Session.Put(r.Context(), "reservation", res)

	sd := res.StartDate.Format("02-01-2006")
	ed := res.EndDate.Format("02-01-2006")

	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	data := make(map[string]interface{})
	data["reservation"] = res
	
	render.Template(w, r, "make-reservation.page.html", &models.TemplateData{
		Form: forms.New(nil),
		Data: data,
		StringMap: stringMap,
	})
}

// PostReservation handles the posting of the reservation form
func (m *Repository) PostReservation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot parse form!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	sd := r.Form.Get("start_date")
	ed := r.Form.Get("end_date")

	// 02-01-2006

	layout := "02-01-2006"

	startDate, err := time.Parse(layout, sd)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot parse start date!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	endDate, err := time.Parse(layout, ed)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot parse end date!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	roomID, err := strconv.Atoi(r.Form.Get("room_id"))
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "invalid data!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// add this to fix invalid data error
	room, err := m.DB.GetRoomByID(roomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find room!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	reservation := models.Reservation {
		FirstName: r.Form.Get("first_name"),
		LastName: r.Form.Get("last_name"),
		Phone: r.Form.Get("phone"),
		Email: r.Form.Get("email"),
		StartDate: startDate,
		EndDate: endDate,
		RoomID: roomID,
		Room: room,
	}

	form := forms.New(r.PostForm)

	form.Required("first_name", "last_name", "email")
	form.MinLength("first_name", 3)
	form.MinLength("last_name", 3)
	form.IsEmail("email")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["reservation"] = reservation
		http.Error(w, "my own error message", http.StatusSeeOther)

		render.Template(w, r, "make-reservation.page.html", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}

	newReservationID, err := m.DB.InsertReservation(reservation)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot insert reservation into database!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	restriction := models.RoomRestriction{
		StartDate: startDate,
		EndDate: endDate,
		RoomID: roomID,
		ReservationID: newReservationID,
		RestrictionID: 1,
	}

	err = m.DB.InsertRoomRestriction(restriction)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "cannot insert room restriction!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// send notification - to the guest
	htmlMessage := fmt.Sprintf(`
		<strong>Reservation Confirmation</strong><br>
		Dear %s: <br>
		This is confirm your reservation from %s to %s.
	`, reservation.FirstName, reservation.StartDate.Format("02-01-2006"), reservation.EndDate.Format("02-01-2006"))

	msg := models.MailData {
		To: reservation.Email,
		From: "me@here.com",
		Subject: "Reservation Confirmation",
		Content: htmlMessage,
		Template: "basic.html",
	}

	m.App.MailChan <- msg

	// send notification - to the owner
	htmlMessage = fmt.Sprintf(`
		<strong>Reservation Notification</strong><br>
		A reservation has been made for %s from %s to %s.
	`, reservation.FirstName, reservation.StartDate.Format("02-01-2006"), reservation.EndDate.Format("02-01-2006"))

	msg = models.MailData {
		To: "me@here.com",
		From: "me@here.com",
		Subject: "Reservation Notification",
		Content: htmlMessage,
	}

	m.App.MailChan <- msg

	m.App.Session.Put(r.Context(), "reservation", reservation)

	http.Redirect(w, r, "/reservation-summary", http.StatusSeeOther)
}

// PostAvailability renders the search availability room page
func (m *Repository) PostAvailability(w http.ResponseWriter, r *http.Request) {
	start := r.Form.Get("start")
	end := r.Form.Get("end")

	// 02-01-2006

	layout := "02-01-2006"
	startDate, err := time.Parse(layout, start)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}
	endDate, err := time.Parse(layout, end)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	rooms, err := m.DB.SearchAvailabilityForAllRooms(startDate, endDate)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	if len(rooms) == 0 {
		// no availability
		m.App.Session.Put(r.Context(), "error", "No availability!")
		http.Redirect(w, r, "/search-availability", http.StatusSeeOther)
		return
	} 

	data := make(map[string]interface{})
	data["rooms"] = rooms

	res := models.Reservation{
		StartDate: startDate,
		EndDate: endDate,
	}

	m.App.Session.Put(r.Context(), "reservation", res)

	render.Template(w, r, "choose-room.page.html", &models.TemplateData{
		Data: data,
	})
}

type jsonResponse struct {
	OK 			bool	`json:"ok"`
	Message 	string	`json:"message"`
	RoomID		string	`json:"room_id"`
	StartDate	string	`json:"start_date"`
	EndDate		string	`json:"end_date"`
}

// AvailabiltyJSON handles request for availability and sends JSON responds
func (m *Repository) AvailabilityJSON(w http.ResponseWriter, r *http.Request) {
	// need to parse form
	err := r.ParseForm()
	if err != nil {
		// cannot parse form, return appropriate json
		resp := jsonResponse{
			OK: false,
			Message: "Internal server error",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	sd := r.Form.Get("start")
	ed := r.Form.Get("end")

	layout := "02-01-2006"

	startDate, _ := time.Parse(layout, sd)
	endDate, _ := time.Parse(layout, ed)

	roomID, _ := strconv.Atoi(r.Form.Get("room_id"))

	available, err := m.DB.SearchAvailabilityByDatesByRoomID(startDate, endDate, roomID)
	if err != nil {
		// cannot parse form, return appropriate json
		resp := jsonResponse{
			OK: false,
			Message: "Error connecting to database",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	resp := jsonResponse {
		OK: available,
		Message: "",
		StartDate: sd,
		EndDate: ed,
		RoomID: strconv.Itoa(roomID),
	}

	// json response is constructed manually, so error checking is redundant
	out, _ := json.MarshalIndent(resp, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// Contact renders the contact page
func (m *Repository) Contact(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "contact.page.html", &models.TemplateData{})
}

// ReservationSummary displays the reservation summary page
func (m *Repository) ReservationSummary(w http.ResponseWriter, r *http.Request) {
	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.ErrorLog.Println("Can't get error from session")
		m.App.Session.Put(r.Context(), "error", "Cannot get reservation from session!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	m.App.Session.Remove(r.Context(), "reservation")

	data := make(map[string]interface{})
	data["reservation"] = reservation

	sd := reservation.StartDate.Format("02-01-2006")
	ed := reservation.EndDate.Format("02-01-2006")
	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	render.Template(w, r, "reservation-summary.page.html", &models.TemplateData{
		Data: data,
		StringMap: stringMap,
	})
}

// ChooseRoom displays list of available rooms
func (m *Repository) ChooseRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		helpers.ServerError(w, err)
		return
	}

	res.RoomID = roomID

	m.App.Session.Put(r.Context(), "reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusSeeOther)
}

// BookRoom takes URL parameters, build a sessional variable and takes user to make res screen
func (m *Repository) BookRoom(w http.ResponseWriter, r *http.Request) {
	// id, s, e
	roomID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	sd := r.URL.Query().Get("s")
	ed := r.URL.Query().Get("e")

	layout := "02-01-2006"
	startDate, _ := time.Parse(layout, sd)
	endDate, _ := time.Parse(layout, ed)

	var res models.Reservation

	room, err := m.DB.GetRoomByID(roomID)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	res.Room.RoomName = room.RoomName
	res.RoomID = roomID
	res.StartDate = startDate
	res.EndDate = endDate

	m.App.Session.Put(r.Context(),"reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusTemporaryRedirect)
}

// ShowLogin shows the login screen
func (m *Repository) ShowLogin(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "login.page.html", &models.TemplateData{
		Form: forms.New(nil),
	})
}

// PostShowLogin hadnles logging the user in
func (m *Repository) PostShowLogin(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	
	form := forms.New(r.PostForm)
	form.Required("email", "password")
	form.IsEmail("email")
	if !form.Valid() {
		render.Template(w, r, "login.page.html", &models.TemplateData{
			Form: form,
		})
		return
	}

	id, _, err := m.DB.Authenticate(email, password)
	if err != nil {
		log.Println(err)

		m.App.Session.Put(r.Context(), "error", "Invalid login credentials")
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "user_id", id)
	m.App.Session.Put(r.Context(), "flash", "Logged in successfully")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout logs a user out
func (m *Repository) Logout(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.Destroy(r.Context())
	_ = m.App.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (m *Repository) AdminDashBoard(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "admin-dashboard.page.html", &models.TemplateData{})
}

// AdminNewReservations shows all new reservations in admin tool
func (m *Repository) AdminNewReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.AllNewReservations()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	data := make(map[string]interface{})
	data["reservations"] = reservations

	render.Template(w, r, "admin-new-reservations.page.html", &models.TemplateData{
		Data: data,
	})
}

// AdminNewReservations shows all reservations in admin tool
func (m *Repository) AdminAllReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.AllReservations()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	data := make(map[string]interface{})
	data["reservations"] = reservations

	render.Template(w, r, "admin-all-reservations.page.html", &models.TemplateData{
		Data: data,
	})
}

// AdminShowReservation shows the reservation in the admin tool
func (m *Repository) AdminShowReservation(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src

	// get reservation from database
	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	data := make(map[string]interface{})
	data["reservation"] = res

	render.Template(w, r, "admin-reservations-show.page.html", &models.TemplateData{
		StringMap: stringMap,
		Data: data,
		Form: forms.New(nil),
	})
}

func (m *Repository) AdminPostShowReservation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src

	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	res.FirstName = r.Form.Get("first_name")
	res.LastName = r.Form.Get("last_name")
	res.Email = r.Form.Get("email")
	res.Phone = r.Form.Get("phone")

	err = m.DB.UpdateReservation(res)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
}

// AdminReservationCalendar displays the reservation calendar
func (m *Repository) AdminReservationsCalendar(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "admin-reservations-calendar.page.html", &models.TemplateData{})
}