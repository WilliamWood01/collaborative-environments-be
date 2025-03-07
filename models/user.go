package models

type User struct {
	ID       string `json:"id,omitempty" bson:"_id,omitempty"`
	UserID   string `json:"user_id" bson:"user_id"`
	Password string `json:"password" bson:"password"`
}
