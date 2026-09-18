package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required,oneof=income expense"`
}

type UpdateCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

// GET /categories?type=expense
func listCategoriesHandler(c *gin.Context) {
	userID := c.GetString("userID")
	typeFilter := c.Query("type")

	query := "SELECT id, name, type, created_at FROM categories WHERE user_id = ?"
	args := []interface{}{userID}

	if typeFilter != "" {
		query += " AND type = ?"
		args = append(args, typeFilter)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch categories")
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.CreatedAt); err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to read categories")
			return
		}
		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// POST /categories
func createCategoryHandler(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		"INSERT INTO categories (id, user_id, name, type, created_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, req.Name, req.Type, createdAt,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create category")
		return
	}

	c.JSON(http.StatusCreated, Category{ID: id, Name: req.Name, Type: req.Type, CreatedAt: createdAt})
}

// PUT /categories/:id
func updateCategoryHandler(c *gin.Context) {
	userID := c.GetString("userID")
	catID := c.Param("id")

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// pastikan kategori ini milik user yang login
	var ownerID string
	err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", catID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "category not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}
	if ownerID != userID {
		respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this category")
		return
	}

	_, err = db.Exec("UPDATE categories SET name = ? WHERE id = ?", req.Name, catID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update category")
		return
	}

	var cat Category
	db.QueryRow("SELECT id, name, type, created_at FROM categories WHERE id = ?", catID).
		Scan(&cat.ID, &cat.Name, &cat.Type, &cat.CreatedAt)

	c.JSON(http.StatusOK, cat)
}

// DELETE /categories/:id
func deleteCategoryHandler(c *gin.Context) {
	userID := c.GetString("userID")
	catID := c.Param("id")

	var ownerID string
	err := db.QueryRow("SELECT user_id FROM categories WHERE id = ?", catID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "category not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}
	if ownerID != userID {
		respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this category")
		return
	}

	_, err = db.Exec("DELETE FROM categories WHERE id = ?", catID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete category")
		return
	}

	c.Status(http.StatusNoContent)
}