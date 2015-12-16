package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/l8nite/hipchat-cinema/cinema"
	"github.com/l8nite/hipchat-cinema/util"

	"github.com/gorilla/mux"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

const Version = "0.0.0"

var validMovieTitleFor = map[string]bool{
	"back_to_the_future": true,
	"bill_and_ted":       true,
	"hackers":            true,
	"the_holy_grail":     true,
	"the_princess_bride": true,
}

// RoomConfig holds information to send messages to a specific room
type RoomConfig struct {
	token          *hipchat.OAuthAccessToken
	hc             *hipchat.Client
	name           string
	isMoviePlaying bool
	stop           chan bool
}

func (r *RoomConfig) playMovie(movie *cinema.Movie) {
	r.isMoviePlaying = true
	r.stop = make(chan bool)

	for _, scene := range movie.Scenes {
		for _, line := range scene.Lines {
			select {
			case <-r.stop:
				r.isMoviePlaying = false
				return
			default:
				r.act(scene, line)
			}
		}
	}
}

func (r *RoomConfig) act(scene cinema.Scene, line cinema.Line) {
	notifRq := &hipchat.NotificationRequest{
		Message:       line.Text,
		MessageFormat: "html",
		Color:         scene.Actors[line.Actor],
		From:          line.Actor,
	}

	r.hc.Room.Notification(r.name, notifRq)

	time.Sleep(line.Delay)
}

// BotContext holds the base URL that the bot is running under
// and a map of client identifiers to rooms we're installed in
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
	authPayload, err := util.DecodePostJSON(r, false)

	if err != nil {
		log.Fatalf("Parse of installation auth data failed:%v\n", err)
		return
	}

	clientID := authPayload["oauthId"].(string)

	log.Printf("Received install request for clientID: %s\n", clientID)

	credentials := hipchat.ClientCredentials{
		ClientID:     clientID,
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

	c.rooms[clientID] = rc

	json.NewEncoder(w).Encode([]string{"OK"})
}

// DELETE /installable/token
// Callback received when the user wants to uninstall the bot from their channel
func (c *BotContext) uninstall(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	clientID := params["clientID"]

	// return a 204 for every request to this API, regardless of what happens
	defer func() {
		w.WriteHeader(204)
	}()

	log.Printf("Received uninstall request for clientID: %s\n", clientID)

	if _, ok := c.rooms[clientID]; !ok {
		log.Print("Not a registered clientID")
		return
	}

	delete(c.rooms, clientID)
}

// POST /hook
// Callback received when the user types a command our bot recognizes
func (c *BotContext) hook(w http.ResponseWriter, r *http.Request) {
	payLoad, err := util.DecodePostJSON(r, true)

	if err != nil {
		log.Fatalf("Parsed auth data failed: %v\n", err)
	}

	clientID := payLoad["oauth_client_id"].(string)
	item := payLoad["item"].(map[string]interface{})
	msgOuter := item["message"].(map[string]interface{})
	message := msgOuter["message"].(string)

	// must match regex from atlassian-connect.json
	re, err := regexp.Compile(`^/(play|stop)(?:\s+(.+)\s*)?`)
	res := re.FindStringSubmatch(message)
	command := res[1]
	movieTitle := res[2]

	log.Printf("Received %s request to clientID: %s\n", command, clientID)

	if _, ok := c.rooms[clientID]; !ok {
		log.Print("clientID is not registered!")
		return
	}

	room := c.rooms[clientID]

	switch command {
	case "stop":
		if room.isMoviePlaying {
			room.stop <- true
			reply(room, "Movie stopped")
		} else {
			reply(room, "Movie is not playing!")
		}
	case "play":
		if !room.isMoviePlaying {
			_, isValidMovieTitle := validMovieTitleFor[movieTitle]

			if !isValidMovieTitle {
				allowedMovies := make([]string, len(validMovieTitleFor))
				i := 0
				for k := range validMovieTitleFor {
					allowedMovies[i] = k
					i++
				}
				reply(room, fmt.Sprintf("Allowed movies are %v", allowedMovies))
			} else {
				movie, err := cinema.ParseMovieFile(movieTitle)
				if err != nil {
					replyError(room, "Error parsing movie file!")
					log.Fatal(err)
					return
				}

				reply(room, fmt.Sprintf("Got it, now playing \"%s\"", movie.Title))
				go room.playMovie(movie)
			}
		} else {
			// FEATURE: allow movies to be queued
			// FEATURE: remember requestor, tag them when movie starts
			reply(room, "Movie is already playing!")
		}
	default:
		reply(room, fmt.Sprintf("Unknown command: %s", command))
	}
}

func reply(r *RoomConfig, message string) {
	notifRq := &hipchat.NotificationRequest{
		Message:       message,
		MessageFormat: "html",
		Color:         "green",
		From:          "Hipchat Cinema",
	}

	r.hc.Room.Notification(r.name, notifRq)
}

func replyError(r *RoomConfig, message string) {
	notifRq := &hipchat.NotificationRequest{
		Message:       message,
		MessageFormat: "html",
		Color:         "red",
		From:          "Hipchat Cinema",
	}

	r.hc.Room.Notification(r.name, notifRq)
}

// Install http handler routes
func (c *BotContext) routes() *mux.Router {
	r := mux.NewRouter()

	r.Path("/").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/atlassian-connect.json").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/installable").Methods("POST").HandlerFunc(c.install)
	r.Path("/installable/{clientID}").Methods("DELETE").HandlerFunc(c.uninstall)
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
