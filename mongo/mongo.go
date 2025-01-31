package mongo

import (
	"chat-app-server/models" // Import the models package
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var chatCollection *mongo.Collection

// Setup MongoDB connection
func SetupMongoDB() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	mongoClient = client
	chatCollection = mongoClient.Database("chat-app-db").Collection("messages")
	fmt.Println("Connected to MongoDB!")
}

// Save a message to MongoDB
func SaveMessageToDB(message models.Message) {
	_, err := chatCollection.InsertOne(context.Background(), message)
	if err != nil {
		log.Fatalf("Failed to save message: %v", err)
	}
	fmt.Println("Message saved to MongoDB!")
}

// Get all messages from MongoDB
func GetAllMessagesFromDB() ([]models.Message, error) {
    var messages []models.Message
    cursor, err := chatCollection.Find(context.Background(), bson.D{})
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve messages: %v", err)
    }
    defer cursor.Close(context.Background())

    for cursor.Next(context.Background()) {
        var message models.Message
        if err = cursor.Decode(&message); err != nil {
            return nil, fmt.Errorf("failed to decode message: %v", err)
        }
        messages = append(messages, message)
    }

    if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
    }

    return messages, nil
}