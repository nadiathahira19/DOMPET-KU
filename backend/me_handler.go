package main

import "github.com/gin-gonic/gin"

func meHandler(c *gin.Context) {
	userID := c.GetString("userID")

	var user User
	err := db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		respondError(c, 404, "NOT_FOUND", "user not found")
		return
	}

	c.JSON(200, user)
}