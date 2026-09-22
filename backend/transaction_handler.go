package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Transaction struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Amount     int                  `json:"amount"`
	Note       string               `json:"note"`
	Category   TransactionCategory  `json:"category"`
	OccurredAt string               `json:"occurred_at"`
	CreatedAt  string               `json:"created_at"`
}

type CreateTransactionRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=income expense"`
	Amount     int    `json:"amount" binding:"required,gt=0"`
	Note       string `json:"note"`
	OccurredAt string `json:"occurred_at" binding:"required"`
}

type UpdateTransactionRequest struct {
	CategoryID *string `json:"category_id"`
	Amount     *int    `json:"amount"`
	Note       *string `json:"note"`
	OccurredAt *string `json:"occurred_at"`
}

// GET /transactions
func listTransactionsHandler(c *gin.Context) {
	userID := c.GetString("userID")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := `SELECT t.id, t.type, t.amount, t.note, t.category_id, c.name, t.occurred_at, t.created_at
	          FROM transactions t
	          JOIN categories c ON t.category_id = c.id
	          WHERE t.user_id = ?`
	countQuery := `SELECT COUNT(*) FROM transactions t WHERE t.user_id = ?`
	args := []interface{}{userID}
	countArgs := []interface{}{userID}

	if typeFilter := c.Query("type"); typeFilter != "" {
		query += " AND t.type = ?"
		countQuery += " AND t.type = ?"
		args = append(args, typeFilter)
		countArgs = append(countArgs, typeFilter)
	}
	if catFilter := c.Query("category_id"); catFilter != "" {
		query += " AND t.category_id = ?"
		countQuery += " AND t.category_id = ?"
		args = append(args, catFilter)
		countArgs = append(countArgs, catFilter)
	}
	if from := c.Query("from"); from != "" {
		query += " AND t.occurred_at >= ?"
		countQuery += " AND t.occurred_at >= ?"
		args = append(args, from)
		countArgs = append(countArgs, from)
	}
	if to := c.Query("to"); to != "" {
		query += " AND t.occurred_at <= ?"
		countQuery += " AND t.occurred_at <= ?"
		args = append(args, to+"T23:59:59Z")
		countArgs = append(countArgs, to+"T23:59:59Z")
	}

	var total int
	if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count transactions")
		return
	}

	query += " ORDER BY t.occurred_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch transactions")
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Note, &t.Category.ID, &t.Category.Name, &t.OccurredAt, &t.CreatedAt); err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to read transactions")
			return
		}
		transactions = append(transactions, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": transactions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GET /transactions/:id
func getTransactionHandler(c *gin.Context) {
	userID := c.GetString("userID")
	txID := c.Param("id")

	var t Transaction
	err := db.QueryRow(`SELECT t.id, t.type, t.amount, t.note, t.category_id, c.name, t.occurred_at, t.created_at
		FROM transactions t JOIN categories c ON t.category_id = c.id
		WHERE t.id = ? AND t.user_id = ?`, txID, userID).
		Scan(&t.ID, &t.Type, &t.Amount, &t.Note, &t.Category.ID, &t.Category.Name, &t.OccurredAt, &t.CreatedAt)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "transaction not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}

	c.JSON(http.StatusOK, t)
}

// POST /transactions
func createTransactionHandler(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// pastikan kategori ada dan milik user ini
	var catName, catOwner string
	err := db.QueryRow("SELECT name, user_id FROM categories WHERE id = ?", req.CategoryID).Scan(&catName, &catOwner)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "category not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}
	if catOwner != userID {
		respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this category")
		return
	}

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = db.Exec(
		`INSERT INTO transactions (id, user_id, category_id, type, amount, note, occurred_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, req.CategoryID, req.Type, req.Amount, req.Note, req.OccurredAt, createdAt,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create transaction")
		return
	}

	c.JSON(http.StatusCreated, Transaction{
		ID:         id,
		Type:       req.Type,
		Amount:     req.Amount,
		Note:       req.Note,
		Category:   TransactionCategory{ID: req.CategoryID, Name: catName},
		OccurredAt: req.OccurredAt,
		CreatedAt:  createdAt,
	})
}

// PUT /transactions/:id
func updateTransactionHandler(c *gin.Context) {
	userID := c.GetString("userID")
	txID := c.Param("id")

	var ownerID string
	err := db.QueryRow("SELECT user_id FROM transactions WHERE id = ?", txID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "transaction not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}
	if ownerID != userID {
		respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this transaction")
		return
	}

	var req UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if req.CategoryID != nil {
		db.Exec("UPDATE transactions SET category_id = ? WHERE id = ?", *req.CategoryID, txID)
	}
	if req.Amount != nil {
		db.Exec("UPDATE transactions SET amount = ? WHERE id = ?", *req.Amount, txID)
	}
	if req.Note != nil {
		db.Exec("UPDATE transactions SET note = ? WHERE id = ?", *req.Note, txID)
	}
	if req.OccurredAt != nil {
		db.Exec("UPDATE transactions SET occurred_at = ? WHERE id = ?", *req.OccurredAt, txID)
	}

	var t Transaction
	db.QueryRow(`SELECT t.id, t.type, t.amount, t.note, t.category_id, c.name, t.occurred_at, t.created_at
		FROM transactions t JOIN categories c ON t.category_id = c.id WHERE t.id = ?`, txID).
		Scan(&t.ID, &t.Type, &t.Amount, &t.Note, &t.Category.ID, &t.Category.Name, &t.OccurredAt, &t.CreatedAt)

	c.JSON(http.StatusOK, t)
}

// DELETE /transactions/:id
func deleteTransactionHandler(c *gin.Context) {
	userID := c.GetString("userID")
	txID := c.Param("id")

	var ownerID string
	err := db.QueryRow("SELECT user_id FROM transactions WHERE id = ?", txID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "transaction not found")
		return
	} else if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "database error")
		return
	}
	if ownerID != userID {
		respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this transaction")
		return
	}

	db.Exec("DELETE FROM transactions WHERE id = ?", txID)
	c.Status(http.StatusNoContent)
}