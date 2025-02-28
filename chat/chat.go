package chat

import (
	"chat-app-server/auth" // Import the auth package
	"chat-app-server/middleware"
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
//Creates a broadcast channel to distribute messages to all connected clients, a channel is a communication mechanism that allows one goroutine to send values to another goroutine
var broadcast = make(chan string)
// Create a mutex to synchronize access to the clients map, mutex a mutual exclusion lock, used to synchronize access to shared resources
var mu sync.Mutex

// Handle incoming WebSocket connections, remains running for as long as the individual client is connected
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
	
	log.Printf("New WebSocket connection established: %s", conn.RemoteAddr().String())

	//Defer is a function that is executed when the surrounding function returns, ie. when the client disconnects
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
				//Using the message struct to create a JSON object
				"user_id": msg.UserID,
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

		    // Unmarshal the incoming message to extract the username and text
        var incomingMessage struct {
            Text     string `json:"text"`
            UserID string `json:"user_id"`
        }
        if err := json.Unmarshal(message, &incomingMessage); err != nil {
            log.Println("Failed to unmarshal message:", err)
            continue
        }

		// Save the message to MongoDB
		msg := models.Message{
			UserID:    incomingMessage.UserID,
			RoomID:    "chat-room-1",    // Replace with actual room ID
			Text:      incomingMessage.Text,
			Timestamp: time.Now(),
		}
		mongo.SaveMessageToDB(msg) // Save to MongoDB

		// Broadcast the message to all other connected clients
    messageData, err := json.Marshal(map[string]interface{}{
		//Using the message struct to create a JSON object
		"user_id": msg.UserID,
        "text":      msg.Text,
        "timestamp": msg.Timestamp,
    })
    if err != nil {
        log.Println("Failed to marshal message:", err)
        continue
    }

	// Call the broadcastMessage function to send the message to all connected clients
    broadcastMessage(messageType, messageData)

	// Send the message to the broadcast channel
    broadcast <- string(message)
	}
}

// Function to broadcast a message to all connected clients
func broadcastMessage(messageType int, messageData []byte) {
    mu.Lock()
    defer mu.Unlock()
    for client := range clients {
        if err := client.WriteMessage(messageType, messageData); err != nil {
            log.Println("Failed to write message to client:", err)
            client.Close()
            delete(clients, client)
        }
    }
}

// Start the WebSocket server
func StartServer() {
	// Create a new ServeMux and apply the CORS middleware
    mux := http.NewServeMux()

	// Handle incoming WebSocket connections, publish messages to Redis and distribute messages to all connected clients
	mux.HandleFunc("/ws", handleConnections)

	// Handle user signup
	mux.HandleFunc("/signup", auth.HandleSignup)
	// Handle user login
	mux.HandleFunc("/login", auth.HandleLogin)
	
	handler := middleware.CORS(mux)
	//Start goroutine, using multiple threads to listen for messages on the broadcast channel, each client has its own goroutine
	go func() {
		for {
			//For as long as the server is running, listen for messages on the broadcast channel and publish them to Redis
			msg := <-broadcast
			// Publish to Redis channel, chat-room-1 which currently does nothing else but in the future could be used to 
			// distribute messages to multiple servers or for analytics if the prototype is scaled up
			redis.PublishMessage("chat-room-1", msg)

		}
	}()
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}