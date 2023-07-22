package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gtuk/discordwebhook"
	"github.com/multiplay/go-ts3"
)

func main() {
	discord := os.Getenv("TS_DISCORD_WEBHOOK")
	if discord == "" {
		log.Fatal("Must configure: TS_DISCORD_WEBHOOK")
	}

	// Connect and login
	c, err := ts3.NewClient(os.Getenv("TS_QUERY_ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	user := os.Getenv("TS_QUERY_USER")
	pass := os.Getenv("TS_QUERY_PASS")
	if err := c.Login(user, pass); err != nil {
		log.Fatal(err)
	}

	if err := c.Use(1); err != nil {
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

	clientMap := make(map[int]string)

	log.Println("Online clients:")
	for _, client := range cl {
		if client.Type != 0 {
			continue
		}
		log.Println("-", client)
		clientMap[client.ID] = client.Nickname
	}

	// Listen for client updates
	notifs := c.Notifications()

	for {
		event := <-notifs
		log.Println("=>", event)

		if event.Type == "cliententerview" {
			if event.Data["client_type"] != "0" {
				continue
			}

			clientId, err := strconv.Atoi(event.Data["clid"])
			if err != nil {
				log.Println("Failed to get client ID:", err)
				continue
			}

			clientNick, ok := event.Data["client_nickname"]
			if !ok {
				log.Println("User has no nickname:", clientId)
				continue
			}

			_, previous := clientMap[clientId]
			clientMap[clientId] = clientNick

			if !previous {
				ClientConnected(discord, clientNick)
			}
		} else if event.Type == "clientleftview" {
			clientId, err := strconv.Atoi(event.Data["clid"])
			if err != nil {
				log.Println("Failed to get client ID:", err)
				continue
			}

			clientNick, ok := clientMap[clientId]
			if !ok {
				log.Println("Unknown user left:", clientId)
				continue
			}

			delete(clientMap, clientId)
			ClientDisconnected(discord, clientNick)
		}
	}
}

func ClientConnected(discord string, nick string) {
	bot := os.Getenv("TS_DISCORD_USERNAME")
	if bot == "" {
		bot = "Jeff"
	}

	content := fmt.Sprintf("Client connected: %s", nick)
	message := discordwebhook.Message{
		Username: &bot,
		Content:  &content,
	}

	if err := discordwebhook.SendMessage(discord, message); err != nil {
		log.Println("Failed to log Discord message:", err)
	}
}

func ClientDisconnected(discord string, nick string) {
	bot := os.Getenv("TS_DISCORD_USERNAME")
	if bot == "" {
		bot = "Jeff"
	}

	content := fmt.Sprintf("Client disconnected: %s", nick)
	message := discordwebhook.Message{
		Username: &bot,
		Content:  &content,
	}

	if err := discordwebhook.SendMessage(discord, message); err != nil {
		log.Println("Failed to log Discord message:", err)
	}
}
