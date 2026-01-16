package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	database "github.com/Johnhi19/TreeSpotter_backend/db"
	"github.com/Johnhi19/TreeSpotter_backend/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ----------------------
// Register
// ----------------------
func Register(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Ensure DB is connected
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DATABASE_ISSUE", "error": "database not initialized"})
		return
	}

	// Check if username exists
	var exists string
	err := database.DB.QueryRow("SELECT username FROM users WHERE username = ?", user.Username).Scan(&exists)
	if err != sql.ErrNoRows && err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DATABASE_ISSUE", "error": "Database error when checking username"})
		return
	}
	if exists != "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "USERNAME_TAKEN", "error": "Username already taken"})
		return
	}

	// Check if email exists
	err = database.DB.QueryRow("SELECT email FROM users WHERE email = ?", user.Email).Scan(&exists)
	if err != sql.ErrNoRows && err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DATABASE_ISSUE", "error": "Database error when checking email"})
		return
	}
	if exists != "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "EMAIL_TAKEN", "error": "Email already registered"})
		return
	}

	// Hash password
	hashed, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	// Insert into DB
	_, err = database.DB.Exec(
		"INSERT INTO users (username, password, email) VALUES (?, ?, ?)",
		user.Username,
		string(hashed),
		user.Email,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DATABASE_ISSUE", "error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered"})
}

// ----------------------
// Login
// ----------------------
func Login(c *gin.Context) {
	var user models.User
	var stored models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Ensure DB is connected
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "DATABASE_ISSUE", "error": "database not initialized"})
		return
	}

	// Get user by username
	err := database.DB.QueryRow(
		"SELECT ID, username, password FROM users WHERE username = ?",
		user.Username,
	).Scan(&stored.ID, &stored.Username, &stored.Password)

	if err == sql.ErrNoRows || err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "error": "Invalid username or password"})
		return
	}

	// Compare password
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(user.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "error": "Invalid username or password"})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": stored.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // token expires after a day
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "UNKNOWN_ERROR", "error": "Token creation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
