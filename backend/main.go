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

	r.Run(":8080")
}