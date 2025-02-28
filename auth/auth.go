package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"chat-app-server/models" // Adjust the import path as needed
	"chat-app-server/mongo"  // Adjust the import path as needed

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

    w.WriteHeader(http.StatusOK)
}
