package chat

import (
	"chat-app-server/auth" // Import the auth package
	"chat-app-server/middleware"
	"chat-app-server/models" // Import the models package
	"chat-app-server/mongo"  // Import the mongo package
	"chat-app-server/redis"  // Import the redis package
	"fmt"

	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/petermattis/goid"
)

// Create a map to store connected clients, a map is a collection of key-value pairs, in this case, the key is a WebSocket connection and the value is a boolean
// Proected by a mutex when used
var clients = make(map[*websocket.Conn]bool)

// Creates a broadcast channel to distribute messages to all connected clients, a channel is a communication mechanism that allows one goroutine to send values to another goroutine
var broadcast = make(chan string)

// Create a mutex to synchronize access to the clients map
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
	
	// Get the goroutine ID to prove that each client has its own goroutine, 
	// therefore handling multiple clients concurrently via multi-threading 
	goroutineID := goid.Get()
	log.Printf("New WebSocket connection established: %s (goroutine ID: %d)", conn.RemoteAddr().String(), goroutineID)

	// Defer is a function that is executed when the surrounding function returns, ie. when the client disconnects
	defer func() {
        log.Printf("WebSocket connection closed: %s (goroutine ID: %d)", conn.RemoteAddr().String(), goroutineID)
        conn.Close()
    }()

	// Register the client while the mutual exclusion locking mechanism is in place
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	// Send stored messages to the newly connected client
	messages, err := mongo.GetAllMessagesFromDB()
	if err != nil {
		log.Println("Failed to retrieve messages:", err)
	} else {
		// Write all messages saved in the database to the client by looping through the retrieved messages
		for _, msg := range messages {
			messageData, err := json.Marshal(map[string]interface{}{
				//Using the message struct to create a JSON object
				"user_id":   msg.UserID,
				"text":      msg.Text,
				"timestamp": msg.Timestamp,
				"file_id":   msg.FileID,
				"file_name": msg.FileName,
                "file_type": msg.FileType,
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
		// Read the message from the client
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}

		var incomingMessage struct {
			UserID   string `json:"user_id"`
			Text     string `json:"text"`
			FileID   string `json:"file_id"`
			FileData []byte `json:"file_data"`
			FileName string `json:"file_name"`
            FileType string `json:"file_type"`
		}

		if err := json.Unmarshal(message, &incomingMessage); err != nil {
			log.Println("Failed to unmarshal message:", err)
			continue
		}

		// If the incoming message contains file data
		if len(incomingMessage.FileData) > 0 {
			// Save the file to GridFS and get the file ID
			fileID, err := mongo.SaveFileToGridFS(incomingMessage.FileData, incomingMessage.FileName, incomingMessage.FileType) // Pass file data, name, and type
			if err != nil {
				log.Println("Failed to save file:", err)
				continue
			}

			log.Println("File saved to GridFS with ID:", fileID)
			incomingMessage.FileID = fileID // Assign the generated file ID
		}

		msg := models.Message{
			UserID:    incomingMessage.UserID,
			RoomID:    "chat-room-1",
			Text:      incomingMessage.Text,
			Timestamp: time.Now(),
			FileID:    incomingMessage.FileID, // Use the generated file ID
			FileName:  incomingMessage.FileName,
            FileType:  incomingMessage.FileType,
		}
		mongo.SaveMessageToDB(msg)

		// Marshal the message to JSON before broadcasting
		messageData, err := json.Marshal(map[string]interface{}{
			"user_id":   msg.UserID,
			"text":      msg.Text,
			"timestamp": msg.Timestamp,
			"file_id":   msg.FileID,
			"file_name": msg.FileName,
            "file_type": msg.FileType,
		})
		if err != nil {
			log.Println("Failed to marshal message:", err)
			continue
		}

		broadcastMessage(websocket.TextMessage, messageData)
	}
}

// Function to broadcast a message to all connected clients
func broadcastMessage(messageType int, messageData []byte) {
	//Lock while broadcasting to clients
	mu.Lock()
	//Unlock after broadcasting
	defer mu.Unlock()
	for client := range clients {
		if err := client.WriteMessage(messageType, messageData); err != nil {
			log.Println("Failed to write message to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// Handle file download requests
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    fileID := vars["fileID"]

    fileData, fileName, fileType, err := mongo.GetFileFromGridFS(fileID)
    if err != nil {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }

	// Return the file to the client
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
    w.Header().Set("Content-Type", fileType)
    w.Write(fileData)
}

// Start the WebSocket server
func StartServer() {
	// Create a new mux router
	router := mux.NewRouter()

	// Handle user signup
	router.HandleFunc("/signup", auth.HandleSignup)
	// Handle user login
	router.HandleFunc("/login", auth.HandleLogin)
	// Handle incoming WebSocket connections, publish messages to Redis and distribute messages to all connected clients
	router.HandleFunc("/ws", middleware.VerifyJWT(http.HandlerFunc(handleConnections)).ServeHTTP)
	// Handle file download requests
	router.HandleFunc("/files/{fileID}", middleware.VerifyJWT(http.HandlerFunc(handleFileDownload)).ServeHTTP)

	handler := middleware.CORS(router)
	// Start goroutine, using multiple threads to listen for messages on the broadcast channel, each client has its own goroutine
	go func() {
		for {
			// For as long as the server is running, listen for messages on the broadcast channel and publish them to Redis
			msg := <-broadcast
			// Publish to Redis channel, chat-room-1 which currently does nothing else but in the future could be used to
			// distribute messages to multiple servers or for analytics if the prototype is scaled up
			// Comment out if not using redis
			redis.PublishMessage("chat-room-1", msg)

		}
	}()
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
