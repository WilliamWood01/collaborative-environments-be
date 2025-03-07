package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"chat-app-server/models" // Adjust the import path as needed
	"chat-app-server/mongo"  // Adjust the import path as needed

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// HandleSignup handles user signup
func HandleSignup(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling signup request")

	// Decode the request body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Println("Invalid request payload:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Println("Decoded user:", user)

	// Check if the userID already exists
	_, err := mongo.FindUserByUsername(user.UserID)
	if err == nil {
		log.Println("UserID already exists:", user.UserID)
		http.Error(w, "UserID already exists", http.StatusConflict)
		return
	}

	if err := mongo.SaveUserToDB(user); err != nil {
		log.Println("Failed to save user to DB:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleLogin handles user login
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling login request", r.Body)
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Println("Invalid request payload:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Println("Decoded user:", user)

	foundUser, err := mongo.FindUserByUsername(user.UserID)
	if err != nil {
		log.Println("User not found:", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	log.Println("Found user:", foundUser)
	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(user.Password)); err != nil {
		log.Println("Invalid password:", err)
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		UserID: foundUser.UserID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(models.JwtKey)
	if err != nil {
		log.Println("Failed to generate token:", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Return the token as JSON
	response := map[string]string{
		"token": tokenString,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
