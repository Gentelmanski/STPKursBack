package main

import (
	"auth-system/database"
	"auth-system/handlers"
	"auth-system/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	database.ConnectDB()

	// Create Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// Public routes
	r.POST("/api/register", handlers.Register)
	r.POST("/api/login", handlers.Login)

	// Protected routes
	auth := r.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/profile", handlers.GetProfile)

		// User routes
		user := auth.Group("/user")
		user.Use(middleware.RoleMiddleware("user", "admin"))
		{
			user.GET("/dashboard", handlers.GetUserDashboard)
		}

		// Admin routes
		admin := auth.Group("/admin")
		admin.Use(middleware.RoleMiddleware("admin"))
		{
			admin.GET("/dashboard", handlers.GetAdminDashboard)
		}
	}

	r.Run(":8080")
}
