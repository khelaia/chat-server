package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type User struct {
	ID        int
	Username  string
	Conn      *websocket.Conn
	PartnerId int
}
type Online struct {
	Data   int  `json:"data"`
	Online bool `json:"online"`
}

var (
	users      = make(map[int]*User)
	usersMutex sync.Mutex
)

func main() {
	app := fiber.New()

	// WebSocket route
	app.Get("/ws", websocket.New(handleWebSocket))

	// Start the Fiber app
	err := app.Listen(":8001")
	if err != nil {
		log.Fatal(err)
	}
}

func handleWebSocket(c *websocket.Conn) {
	fmt.Println("WebSocket connection opened")

	// Create a new user
	user := &User{
		ID:        rand.Int(),
		Username:  fmt.Sprintf("User%d", rand.Intn(100)),
		Conn:      c,
		PartnerId: 0,
	}

	// Add the user to the pool
	usersMutex.Lock()
	users[user.ID] = user

	usersMutex.Unlock()
	sendOnlineUsersEvent()

	findRandomPair(user.ID)

	// Read and handle messages from the user's WebSocket connection
	for {
		messageType, p, err := c.ReadMessage()
		if err != nil {
			fmt.Println("WebSocket connection closed")
			break
		}

		// Handle different types of WebSocket messages
		switch messageType {
		case websocket.TextMessage:
			message := string(p)
			partnerUser := ""
			if user.PartnerId != 0 {
				partnerUser = users[user.PartnerId].Username
			}
			fmt.Printf("Received message from %s: %s to %s\n", user.Username, message, partnerUser)

			// Send the message to the paired user
			if pairedUser := findRandomPair(user.ID); pairedUser != nil {
				if err := pairedUser.Conn.WriteMessage(messageType, p); err != nil {
					fmt.Println("Error writing message to paired user:", err)
				}
			}
			// Handle other message types if needed
		}
	}

	// Remove the user from the pool when the connection is closed
	usersMutex.Lock()

	if user.PartnerId != 0 {
		users[user.PartnerId].PartnerId = 0
		if err := users[user.PartnerId].Conn.WriteMessage(websocket.TextMessage, []byte(`{"data":"partner_disconnected"}`)); err != nil {
			fmt.Println("Error writing message to paired user:", err)
		}
	}
	delete(users, user.ID)
	usersMutex.Unlock()
}

func sendOnlineUsersEvent() {
	for _, user := range users {
		data := Online{Data: len(users), Online: true}
		d, _ := json.Marshal(data)
		if err := user.Conn.WriteMessage(websocket.TextMessage, d); err != nil {
			fmt.Println("Error writing message to paired user:", err)
		}
	}
}
func findRandomPair(userID int) *User {
	usersMutex.Lock()
	defer usersMutex.Unlock()

	// Filter out the current user
	if users[userID].PartnerId != 0 {
		return users[users[userID].PartnerId]
	}
	var availableUsers []*User
	for _, u := range users {
		if u.ID != userID && u.PartnerId == 0 {
			availableUsers = append(availableUsers, u)
		}
	}

	// If there are available users, pick a random one
	if len(availableUsers) > 0 {
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(availableUsers))
		pairedUser := availableUsers[randomIndex]

		// Mark both users as unavailable
		users[userID].PartnerId = pairedUser.ID
		users[pairedUser.ID].PartnerId = userID

		if err := users[userID].Conn.WriteMessage(websocket.TextMessage, []byte(`{"data":"partner_connected"}`)); err != nil {
			fmt.Println("Error writing message to paired user:", err)
		}
		if err := users[pairedUser.ID].Conn.WriteMessage(websocket.TextMessage, []byte(`{"data":"partner_connected"}`)); err != nil {
			fmt.Println("Error writing message to paired user:", err)
		}

		return pairedUser
	}

	return nil
}
