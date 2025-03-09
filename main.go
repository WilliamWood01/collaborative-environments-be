package main

import (
	"chat-app-server/chat"  // Import the chat package
	"chat-app-server/mongo" // Import the mongo package
	"chat-app-server/redis" // Import the redis package
	"fmt"
)

//Function to start the chat application, can be run in the terminal with the command "go run main.go"
func main() {
	// Initialize Redis
	// Just comment this out if you don't have Redis installed as it does not really do anything at the moment
	redis.SetupRedis()

	// Initialize MongoDB
	mongo.SetupMongoDB()

	// Start the WebSocket server
	chat.StartServer()

	fmt.Println("Chat application is running...")
}
