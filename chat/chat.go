package chat

import (
	"chat-app-server/models" // Import the models package
	"chat-app-server/mongo"  // Import the mongo package
	"chat-app-server/redis"  // Import the redis package
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)
var mu sync.Mutex

// Handle incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Register the client
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	// Send stored messages to the newly connected client
	messages, err := mongo.GetAllMessagesFromDB()
	if err != nil {
    	log.Println("Failed to retrieve messages:", err)
	} else {
   		for _, msg := range messages {
        	messageData, err := json.Marshal(map[string]interface{}{
            	"text":      msg.Text,
            	"timestamp": msg.Timestamp,
        })
        if err != nil {
            log.Println("Failed to marshal message:", err)
            continue
        }
        if err := conn.WriteMessage(websocket.TextMessage, messageData); err != nil {
            log.Println("Failed to send message:", err)
            conn.Close()
            mu.Lock()
            delete(clients, conn)
            mu.Unlock()
            return
        }
    	}
	}


	// Listen for incoming messages from WebSocket clients
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}

		// Save the message to MongoDB
		msg := models.Message{
			UserID:    "user123",        // Replace with actual user ID
			RoomID:    "chat-room-1",    // Replace with actual room ID
			Text:      string(message),
			Timestamp: time.Now(),
		}
		mongo.SaveMessageToDB(msg) // Save to MongoDB

		// Broadcast the message to all other connected clients
    messageData, err := json.Marshal(map[string]interface{}{
        "text":      msg.Text,
        "timestamp": msg.Timestamp,
    })
    if err != nil {
        log.Println("Failed to marshal message:", err)
        continue
    }
    for client := range clients {
        if err := client.WriteMessage(messageType, messageData); err != nil {
            log.Println(err)
            client.Close()
            mu.Lock()
            delete(clients, client)
            mu.Unlock()
        }
    }

		// Publish the message to Redis
		redis.PublishMessage("chat-room-1", string(message))
	}
}

// Start the WebSocket server
func StartServer() {
	http.HandleFunc("/ws", handleConnections)
	go func() {
		for {
			msg := <-broadcast
			redis.PublishMessage("chat-room-1", msg) // Publish to Redis channel
		}
	}()
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}