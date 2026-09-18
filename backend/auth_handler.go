package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("190105_rahasia-dompetku-2026-190105")

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Error: ErrorDetail{Code: code, Message: message}})
}

func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	var exists string
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&exists)
	if err == nil {
		respondError(c, http.StatusConflict, "EMAIL_TAKEN", "Email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
		return
	}

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = db.Exec(
		"INSERT INTO users (id, name, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		id, req.Name, req.Email, string(hash), createdAt,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user")
		return
	}

	c.JSON(http.StatusCreated, User{
		ID:        id,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: createdAt,
	})
}

func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	var id, passwordHash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE email = ?", req.Email).Scan(&id, &passwordHash)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	expiresIn := 3600 // 1 hour in seconds
	claims := jwt.MapClaims{
		"sub": id,
		"exp": time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": signedToken,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
	})
}