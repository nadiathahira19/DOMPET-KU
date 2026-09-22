package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /summary?month=2026-09
func summaryHandler(c *gin.Context) {
	userID := c.GetString("userID")
	month := c.Query("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}

	var totalIncome, totalExpense int

	err := db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM transactions
		 WHERE user_id = ? AND type = 'income' AND strftime('%Y-%m', occurred_at) = ?`,
		userID, month,
	).Scan(&totalIncome)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate income")
		return
	}

	err = db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM transactions
		 WHERE user_id = ? AND type = 'expense' AND strftime('%Y-%m', occurred_at) = ?`,
		userID, month,
	).Scan(&totalExpense)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate expense")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"month":         month,
		"total_income":  totalIncome,
		"total_expense": totalExpense,
		"balance":       totalIncome - totalExpense,
	})
}

type CategoryBreakdown struct {
	Category string `json:"category"`
	Total    int    `json:"total"`
}

// GET /summary/by-category?month=2026-09&type=expense
func summaryByCategoryHandler(c *gin.Context) {
	userID := c.GetString("userID")
	month := c.Query("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	typeFilter := c.DefaultQuery("type", "expense")

	rows, err := db.Query(
		`SELECT c.name, COALESCE(SUM(t.amount), 0) as total
		 FROM transactions t
		 JOIN categories c ON t.category_id = c.id
		 WHERE t.user_id = ? AND t.type = ? AND strftime('%Y-%m', t.occurred_at) = ?
		 GROUP BY c.id, c.name
		 ORDER BY total DESC`,
		userID, typeFilter, month,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate breakdown")
		return
	}
	defer rows.Close()

	breakdown := []CategoryBreakdown{}
	for rows.Next() {
		var b CategoryBreakdown
		if err := rows.Scan(&b.Category, &b.Total); err != nil {
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to read breakdown")
			return
		}
		breakdown = append(breakdown, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"month": month,
		"type":  typeFilter,
		"data":  breakdown,
	})
}