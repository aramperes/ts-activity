package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gtuk/discordwebhook"
	"github.com/multiplay/go-ts3"
)

// App holds the configuration
type App struct {
	discordURL         string
	discordUsername    string
	discordAvatarURL   *string
	tsQueryAddr        string
	tsQueryUser        string
	tsQueryPass        string
	tsQueryServerID    int
	spotLightGfxFormat string
	spotLightIDMap     map[string]int
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
	spotLightGfxFormat := os.Getenv("TS_SPOTLIGHT_GFX_FMT")

	// TODO: Load from environment variable
	spotLightIDMap := make(map[string]int)
	spotLightIDMap["rb+mT/4bh37gHzQYqTgBiEHG2IA="] = 0
	spotLightIDMap["sA3fHhvqmlSuFYtMoVYseRQI2DE="] = 0
	spotLightIDMap["9K6JV7kWaRU+4HFRkXrBZNjSmRA="] = 1
	spotLightIDMap["pFclzBx0w2UmwPd91VvaXJjYCYA="] = 2
	spotLightIDMap["tvjlpKqvcyQSCCVkT0TJ28uwdaQ="] = 3
	spotLightIDMap["SLLvtjVBmSoIzpMhlxnLa9CWoOU="] = 4
	spotLightIDMap["7EU/Up++D9+8SQk0sNchEuKPufw="] = 5
	spotLightIDMap["SyldxnLYWHJOUj3HnEsXGF6B0T4="] = 5
	spotLightIDMap["Mc/TdoNhddKdGtB55DSZrYk3NWc="] = 6
	spotLightIDMap["xOWMWG/TpkbV8XjahqqoQLsHHpA="] = 7
	spotLightIDMap["G4kg1LKJElM5LIpoeA6gN7DMl0c="] = 7
	spotLightIDMap["wuQ907NtzqL4uxhLk3P/TCpkXF0="] = 8

	return &App{
		discordURL:         discordURL,
		discordUsername:    discordUsername,
		discordAvatarURL:   discordAvatarURL,
		tsQueryAddr:        tsQueryAddr,
		tsQueryUser:        tsQueryUser,
		tsQueryPass:        tsQueryPass,
		tsQueryServerID:    tsQueryServerID,
		spotLightGfxFormat: spotLightGfxFormat,
		spotLightIDMap:     spotLightIDMap,
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
	clientDatabaseIDs := make(map[string]string)
	clientUniqueIDs := make(map[string]string)

	log.Println("Online clients:")
	for _, client := range cl {
		if client.Type != 0 {
			continue
		}
		log.Println("-", client)
		clientMap[strconv.Itoa(client.ID)] = client.Nickname
		clientDatabaseIDs[strconv.Itoa(client.ID)] = strconv.Itoa(client.DatabaseID)

		uid, err := getClientUniqueId(c, strconv.Itoa(client.DatabaseID))
		if err != nil {
			log.Fatal(err)
		}

		clientUniqueIDs[strconv.Itoa(client.ID)] = uid
		log.Println("  - UID:", uid)
	}

	// Update the banner on startup with the currently online users.
	app.updateSpotLight(c, mapValues(clientUniqueIDs))

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

			clientDBID, ok := event.Data["client_database_id"]
			if !ok {
				log.Println("User has no client database id", event.Data)
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

				clientDatabaseIDs[clientID] = clientDBID
				uid, err := getClientUniqueId(c, clientDBID)
				if err != nil {
					log.Fatal(err)
				}

				clientUniqueIDs[clientID] = uid

				app.updateSpotLight(c, mapValues(clientUniqueIDs))
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
			delete(clientDatabaseIDs, clientID)
			delete(clientUniqueIDs, clientID)
			app.updateSpotLight(c, mapValues(clientUniqueIDs))
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

func (app *App) updateSpotLight(c *ts3.Client, connectedUIDs []string) {
	if app.spotLightGfxFormat == "" {
		return
	}

	spotLightIDs := make([]int, 0)
	for _, uid := range connectedUIDs {
		spotLightID, ok := app.spotLightIDMap[uid]
		if ok {
			spotLightIDs = append(spotLightIDs, spotLightID)
		}
	}

	if len(spotLightIDs) == 0 {
		updateBanner(c, fmt.Sprintf(app.spotLightGfxFormat, "empty"))
		return
	}

	slices.Sort(spotLightIDs)
	slices.Compact(spotLightIDs)

	spotLightIDStrings := make([]string, len(spotLightIDs))
	for idx, id := range spotLightIDs {
		spotLightIDStrings[idx] = strconv.Itoa(id)
	}
	joined := strings.Join(spotLightIDStrings, "_")

	updateBanner(c, fmt.Sprintf(app.spotLightGfxFormat, joined))
}

func updateBanner(c *ts3.Client, gfx string) {
	err := c.Server.Edit(ts3.NewArg("virtualserver_hostbanner_gfx_url", gfx))
	if err != nil {
		log.Println("Failed to update banner:", gfx, err)
	} else {
		log.Println("Updated banner:", gfx)
	}
}

func getClientUniqueId(c *ts3.Client, dbID string) (string, error) {
	var uid = struct {
		UID string `ms:"cluid"`
	}{}
	_, err := c.ExecCmd(ts3.NewCmd("clientgetnamefromdbid").WithArgs(ts3.NewArg("cldbid", dbID)).WithResponse(&uid))

	if err != nil {
		return "", err
	}

	return uid.UID, nil
}

func mapValues(m map[string]string) []string {
	v := make([]string, 0, len(m))
	for _, val := range m {
		v = append(v, val)
	}
	return v
}
