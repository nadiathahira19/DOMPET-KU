package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token subject")
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}