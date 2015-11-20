package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/l8nite/hipchat-cinema/util"

	"github.com/gorilla/mux"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

const Version = "0.0.0"

// RoomConfig holds information to send messages to a specific room
type RoomConfig struct {
	token          *hipchat.OAuthAccessToken
	hc             *hipchat.Client
	name           string
	isMoviePlaying bool
}

// BotContext holds the base URL that the bot is running under
// and a map of rooms that we've been installed into
type BotContext struct {
	baseURL string
	rooms   map[string]*RoomConfig
}

// GET /atlassian-connect.json
// Fetches the configuration descriptor for this bot
func (c *BotContext) atlassianConnect(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("./static", "atlassian-connect.json")
	vals := map[string]string{
		"BaseUrl": c.baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		log.Fatalf("%v", err)
	}
	tmpl.ExecuteTemplate(w, "atlassian-connect", vals)
}

// POST /installable
// Callback received when the bot is installed in a room
func (c *BotContext) install(w http.ResponseWriter, r *http.Request) {
	authPayload, err := util.DecodePostJSON(r, true)

	if err != nil {
		log.Fatalf("Parse of installation auth data failed:%v\n", err)
		return
	}

	credentials := hipchat.ClientCredentials{
		ClientID:     authPayload["oauthId"].(string),
		ClientSecret: authPayload["oauthSecret"].(string),
	}

	roomName := strconv.Itoa(int(authPayload["roomId"].(float64)))

	newClient := hipchat.NewClient("")

	tok, _, err := newClient.GenerateToken(credentials, []string{hipchat.ScopeSendNotification})

	if err != nil {
		log.Fatalf("Client.GetAccessToken returns an error %v", err)
	}

	rc := &RoomConfig{
		name: roomName,
		hc:   tok.CreateClient(),
	}

	c.rooms[roomName] = rc

	util.PrintDump(w, r, false)
	json.NewEncoder(w).Encode([]string{"OK"})
}

// DELETE /installable
// Callback received when the user wants to uninstall the bot from their channel
func (c *BotContext) uninstall(w http.ResponseWriter, r *http.Request) {
	// TODO: parse out roomID, remove from configured rooms map
	w.WriteHeader(204)
}

// POST /hook
// Callback received when the user types a command our bot recognizes
func (c *BotContext) hook(w http.ResponseWriter, r *http.Request) {
	util.PrintDump(w, r, true)

	payLoad, err := util.DecodePostJSON(r, true)

	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}

	roomID := strconv.Itoa(int((payLoad["item"].(map[string]interface{}))["room"].(map[string]interface{})["id"].(float64)))

	log.Printf("Received play request to roomID: %s\n", roomID)

	if _, ok := c.rooms[roomID]; !ok {
		log.Print("Room is not registered!")
		return
	}

	var message string

	if c.rooms[roomID].isMoviePlaying {
		message = "Movie is already playing!"
	} else {
		message = "Enjoy the show!"
		c.rooms[roomID].isMoviePlaying = true
	}

	notifRq := &hipchat.NotificationRequest{
		Message:       message,
		MessageFormat: "html",
		Color:         "red",
		From:          "God",
	}

	if _, ok := c.rooms[roomID]; ok {
		_, err = c.rooms[roomID].hc.Room.Notification(roomID, notifRq)
		if err != nil {
			log.Printf("Failed to notify HipChat channel:%v\n", err)
		}
	} else {
		log.Printf("Room is not registered correctly:%v\n", c.rooms)
	}
}

// Install http handler routes
func (c *BotContext) routes() *mux.Router {
	r := mux.NewRouter()

	r.Path("/").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/atlassian-connect.json").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/installable").Methods("POST").HandlerFunc(c.install)
	r.Path("/installable/{token}").Methods("DELETE").HandlerFunc(c.uninstall)
	r.Path("/hook").Methods("POST").HandlerFunc(c.hook)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))
	return r
}

// Start up, parse command line flags, handle http requests
func main() {
	var (
		port    = flag.String("port", "8080", "web server port")
		baseURL = flag.String("baseURL", os.Getenv("BASE_URL"), "server base url")
	)

	flag.Parse()

	c := &BotContext{
		baseURL: *baseURL,
		rooms:   make(map[string]*RoomConfig),
	}

	log.Printf("HipChat Cinema v%s - Listening on port %v", Version, *port)

	r := c.routes()
	http.Handle("/", r)
	http.ListenAndServe(":"+*port, nil)
}
