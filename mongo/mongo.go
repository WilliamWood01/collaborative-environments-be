package mongo

import (
	"chat-app-server/models" // Import the models package
	"context"
	"fmt"
	"io"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var mongoClient *mongo.Client
var chatCollection *mongo.Collection
var userCollection *mongo.Collection

// Setup MongoDB connection
func SetupMongoDB() {
	//Set up the MongoDB client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	mongoClient = client
	//Initialize each collection in the database
	chatCollection = mongoClient.Database("chat-app-db").Collection("messages")
	userCollection = mongoClient.Database("chat-app-db").Collection("users")
	fmt.Println("Connected to MongoDB!")
}

// Save a message to MongoDB
func SaveMessageToDB(message models.Message) {
	//Insert the message into the collection
	_, err := chatCollection.InsertOne(context.Background(), message)
	if err != nil {
		log.Fatalf("Failed to save message: %v", err)
	}
	fmt.Println("Message saved to MongoDB!")
}

// Get all stored messages from MongoDB
func GetAllMessagesFromDB() ([]models.Message, error) {
	var messages []models.Message
	cursor, err := chatCollection.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %v", err)
	}
	defer cursor.Close(context.Background())

	// Iterate through the cursor and decode each message
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

// Save a user to MongoDB
func SaveUserToDB(user models.User) error {
	log.Println("Saving user to DB", user)
	// Hash the password before saving it to the database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	user.Password = string(hashedPassword)

	// Insert the user into the collection
	_, err = userCollection.InsertOne(context.Background(), user)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	return nil
}

// Find a user by username
func FindUserByUsername(userID string) (models.User, error) {
	log.Println("Finding user by username")
	var user models.User
	// Find the user by username and decode it
	err := userCollection.FindOne(context.Background(), bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		return user, fmt.Errorf("user not found: %v", err)
	}
	return user, nil
}

// Function to save a file to GridFS
func SaveFileToGridFS(fileData []byte, fileName string, fileType string) (string, error) {
    bucket, err := gridfs.NewBucket(mongoClient.Database("chat-app-db"))
    if err != nil {
        return "", err
    }

    uploadStream, err := bucket.OpenUploadStream(fileName, options.GridFSUpload().SetMetadata(bson.M{"contentType": fileType}))
    if err != nil {
        return "", err
    }
    defer uploadStream.Close()

    _, err = uploadStream.Write(fileData)
    if err != nil {
        return "", err
    }

    fileID := uploadStream.FileID.(primitive.ObjectID).Hex() // Convert ObjectID to string
    return fileID, nil
}

// Function to get a file from GridFS
func GetFileFromGridFS(fileID string) ([]byte, string, string, error) {
    bucket, err := gridfs.NewBucket(mongoClient.Database("chat-app-db"))
    if err != nil {
        return nil, "", "", err
    }

	// Convert the file ID string to an ObjectID
    objectID, err := primitive.ObjectIDFromHex(fileID)
    if err != nil {
        return nil, "", "", err
    }

    downloadStream, err := bucket.OpenDownloadStream(objectID)
    if err != nil {
        return nil, "", "", err
    }
    defer downloadStream.Close()

    fileData, err := io.ReadAll(downloadStream)
    if err != nil {
        return nil, "", "", err
    }

	// Assign file name and content type from the file metadata
    fileName := downloadStream.GetFile().Name
    var metadata bson.M
    if err := bson.Unmarshal(downloadStream.GetFile().Metadata, &metadata); err != nil {
        return nil, "", "", err
    }
    fileType, ok := metadata["contentType"].(string)
    if !ok {
        fileType = "application/octet-stream" // Default to binary stream if content type is not found
    }

    return fileData, fileName, fileType, nil
}