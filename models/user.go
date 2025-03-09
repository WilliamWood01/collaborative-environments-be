package models

// Strcuture for users in the database and JSON responses
type User struct {
	ID       string `json:"id,omitempty" bson:"_id,omitempty"`
	UserID   string `json:"user_id" bson:"user_id"`
	Password string `json:"password" bson:"password"`
}
