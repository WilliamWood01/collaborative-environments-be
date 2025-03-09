package models

import "time"

// Strcuture for messages in the database and JSON responses
type Message struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	UserID    string    `json:"user_id" bson:"user_id"`
	RoomID    string    `json:"room_id" bson:"room_id"`
	Text      string    `json:"text" bson:"text"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	FileID    string    `json:"file_id,omitempty" bson:"file_id,omitempty"`
	FileName  string    `json:"file_name,omitempty" bson:"file_name,omitempty"`
    FileType  string    `json:"file_type,omitempty" bson:"file_type,omitempty"`
}
