package main

import (
	"chat-app-server/chat"  // Import the chat package
	"chat-app-server/mongo" // Import the mongo package
	"chat-app-server/redis" // Import the redis package
	"fmt"
)

func main() {
	// Initialize Redis
	redis.SetupRedis()

	// Initialize MongoDB
	mongo.SetupMongoDB()

	// Start the WebSocket server
	chat.StartServer()

	fmt.Println("Chat application is running...")
}