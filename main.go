// main.go
package main

import (
	"auth-system/database"
	"auth-system/handlers"
	"auth-system/middleware"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	database.ConnectDB()

	// Run migrations
	database.Migrate()

	// Create Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	r.Use(cors.New(config))

	// Public routes
	r.POST("/api/register", handlers.Register)
	r.POST("/api/login", handlers.Login)

	// Protected routes
	auth := r.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/profile", handlers.GetProfile)

		// Event routes
		auth.GET("/events", handlers.GetEvents)
		auth.POST("/events", handlers.CreateEvent)
		auth.POST("/events/filter", handlers.FilterEvents)

		// Event-specific routes
		event := auth.Group("/events/:id")
		{
			event.GET("", handlers.GetEventByID)
			event.PUT("", handlers.UpdateEvent)
			event.DELETE("", handlers.DeleteEvent)
			event.POST("/participate", handlers.ParticipateEvent)
			event.DELETE("/participate", handlers.CancelParticipation)

			// Comments for specific event
			event.GET("/comments", handlers.GetComments)
			event.POST("/comments", handlers.CreateComment)
		}

		// Comment routes (individual comments)
		auth.PUT("/comments/:commentId", handlers.UpdateComment)
		auth.DELETE("/comments/:commentId", handlers.DeleteComment)
		auth.POST("/comments/:commentId/vote", handlers.VoteComment)

		// User-specific events
		auth.GET("/user/events", handlers.GetEventsByUser)
		auth.GET("/user/participated", handlers.GetParticipatedEvents)

		// Notification routes
		auth.GET("/notifications", handlers.GetNotifications)
		auth.PUT("/notifications/:id/read", handlers.MarkNotificationAsRead)
		auth.POST("/notifications/mark-all-read", handlers.MarkAllNotificationsAsRead)

		// User dashboard routes - ОДИН раз!
		// Используем RoleMiddleware для проверки роли
		userGroup := auth.Group("")
		userGroup.Use(middleware.RoleMiddleware("user", "admin"))
		{
			userGroup.GET("/user/dashboard", handlers.GetUserDashboard)
		}

		// Admin routes
		adminGroup := auth.Group("")
		adminGroup.Use(middleware.RoleMiddleware("admin"))
		{
			adminGroup.GET("/admin/dashboard", handlers.GetAdminDashboard)
			adminGroup.GET("/admin/events", handlers.AdminGetEvents)
			adminGroup.PUT("/admin/events/:eventId/verify", handlers.AdminVerifyEvent)
			adminGroup.PUT("/admin/events/:eventId/reject", handlers.AdminRejectEvent)
			adminGroup.DELETE("/admin/events/:eventId", handlers.AdminDeleteEvent)
			adminGroup.GET("/admin/users", handlers.AdminGetUsers)
			adminGroup.PUT("/admin/users/:userId/block", handlers.AdminBlockUser)
			adminGroup.PUT("/admin/users/:userId/unblock", handlers.AdminUnblockUser)
			adminGroup.DELETE("/admin/comments/:commentId", handlers.AdminDeleteComment)
			adminGroup.GET("/admin/statistics", handlers.AdminGetStatistics)
		}
	}

	log.Println("Server started on :8080")
	r.Run(":8080")
}
