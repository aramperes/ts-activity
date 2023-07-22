package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gtuk/discordwebhook"
	"github.com/multiplay/go-ts3"
)

// App holds the configuration
type App struct {
	discordURL       string
	discordUsername  string
	discordAvatarURL *string
	tsQueryAddr      string
	tsQueryUser      string
	tsQueryPass      string
	tsQueryServerID  int
}

func appFromEnv() (*App, error) {
	discordURL := os.Getenv("TS_DISCORD_WEBHOOK")
	if discordURL == "" {
		return nil, errors.New("must configure: TS_DISCORD_WEBHOOK")
	}
	discordUsername := os.Getenv("TS_DISCORD_USERNAME")
	if discordUsername == "" {
		discordUsername = "TeamSpeak"
	}

	var discordAvatarURL *string = nil
	if val, ok := os.LookupEnv("TS_DISCORD_AVATAR"); ok {
		discordAvatarURL = &val
	}

	tsQueryAddr := os.Getenv("TS_QUERY_ADDR")
	if tsQueryAddr == "" {
		return nil, errors.New("must configure: TS_QUERY_ADDR")
	}
	tsQueryUser := os.Getenv("TS_QUERY_USER")
	if tsQueryUser == "" {
		return nil, errors.New("must configure: TS_QUERY_USER")
	}
	tsQueryPass := os.Getenv("TS_QUERY_PASS")
	if tsQueryPass == "" {
		return nil, errors.New("must configure: TS_QUERY_PASS")
	}
	tsQueryServerID := 1
	if val, ok := os.LookupEnv("TS_QUERY_SERVER_ID"); ok {
		val, err := strconv.Atoi(val)
		if err == nil {
			tsQueryServerID = val
		} else {
			return nil, errors.New("invalid TS_QUERY_SERVER_ID, must be int")
		}
	}

	return &App{
		discordURL:       discordURL,
		discordUsername:  discordUsername,
		discordAvatarURL: discordAvatarURL,
		tsQueryAddr:      tsQueryAddr,
		tsQueryUser:      tsQueryUser,
		tsQueryPass:      tsQueryPass,
		tsQueryServerID:  tsQueryServerID,
	}, nil
}

func main() {
	app, err := appFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Connect and login
	c, err := ts3.NewClient(app.tsQueryAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.Login(app.tsQueryUser, app.tsQueryPass); err != nil {
		log.Fatal(err)
	}

	if err := c.Use(app.tsQueryServerID); err != nil {
		log.Fatal(err)
	}

	if v, err := c.Whoami(); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Hello, ts-activity:", *v)
	}

	// Subscribe to server events (e.g. client connections)
	if err := c.Register(ts3.ServerEvents); err != nil {
		log.Fatal(err)
	}

	// List current clients
	cl, err := c.Server.ClientList()
	if err != nil {
		log.Fatal(err)
	}

	clientMap := make(map[string]string)

	log.Println("Online clients:")
	for _, client := range cl {
		if client.Type != 0 {
			continue
		}
		log.Println("-", client)
		clientMap[strconv.Itoa(client.ID)] = client.Nickname
	}

	// Listen for client updates
	notifs := c.Notifications()

	for {
		event := <-notifs

		if event.Type == "cliententerview" {
			if event.Data["client_type"] != "0" {
				continue
			}

			clientID, ok := event.Data["clid"]
			if !ok {
				log.Println("User has no client id", event.Data)
				continue
			}

			clientNick, ok := event.Data["client_nickname"]
			if !ok {
				log.Println("User has no nickname:", clientID)
				continue
			}

			_, previous := clientMap[clientID]
			clientMap[clientID] = clientNick

			if !previous {
				app.clientConnected(clientNick)
			}
		} else if event.Type == "clientleftview" {
			clientID, ok := event.Data["clid"]
			if !ok {
				log.Println("User has no client id", event.Data)
				continue
			}

			clientNick, ok := clientMap[clientID]
			if !ok {
				log.Println("Unknown user left:", clientID)
				continue
			}

			delete(clientMap, clientID)
			app.clientDisconnected(clientNick)
		}
	}
}

func (app *App) sendWebhook(content string) {
	message := discordwebhook.Message{
		Username:  &app.discordUsername,
		Content:   &content,
		AvatarUrl: app.discordAvatarURL,
	}

	if err := discordwebhook.SendMessage(app.discordURL, message); err != nil {
		log.Println("Failed to log Discord message:", err)
	}
}

func (app *App) clientConnected(nick string) {
	content := fmt.Sprintf("Client connected: %s", nick)
	app.sendWebhook(content)
}

func (app *App) clientDisconnected(nick string) {
	content := fmt.Sprintf("Client disconnected: %s", nick)
	app.sendWebhook(content)
}
