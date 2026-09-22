package main

import "github.com/gin-gonic/gin"

func main() {
	if err := initDB(); err != nil {
		panic(err)
	}

	r := gin.Default()

	r.POST("/auth/register", registerHandler)
	r.POST("/auth/login", loginHandler)

	protected := r.Group("/")
	protected.Use(authMiddleware())
	protected.GET("/me", meHandler)

	protected.GET("/categories", listCategoriesHandler)
	protected.POST("/categories", createCategoryHandler)
	protected.PUT("/categories/:id", updateCategoryHandler)
	protected.DELETE("/categories/:id", deleteCategoryHandler)
	protected.GET("/transactions", listTransactionsHandler)
	protected.POST("/transactions", createTransactionHandler)
	protected.GET("/transactions/:id", getTransactionHandler)
	protected.PUT("/transactions/:id", updateTransactionHandler)
	protected.DELETE("/transactions/:id", deleteTransactionHandler)
	protected.GET("/summary", summaryHandler)
	protected.GET("/summary/by-category", summaryByCategoryHandler)

	r.Run(":8080")
}