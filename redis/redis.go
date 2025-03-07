package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client
var ctx = context.Background()

// Setup Redis connection
func SetupRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server address
	})
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis!")
}

// Publish a message to Redis
func PublishMessage(channel string, message string) {
	err := redisClient.Publish(ctx, channel, message).Err()
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}
}
