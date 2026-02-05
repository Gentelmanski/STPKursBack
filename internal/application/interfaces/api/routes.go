package api

import (
	"auth-system/internal/application/interfaces/controllers"
	"auth-system/internal/infrastructure/http/middlewares"
	"auth-system/internal/pkg/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	router *gin.Engine,
	ctrls *controllers.Controllers,
	jwtUtil utils.JWTUtil,
) *gin.Engine {
	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	router.Use(cors.New(config))

	// Public routes
	api := router.Group("/api")
	{
		// Auth routes
		api.POST("/register", ctrls.Auth.Register)
		api.POST("/login", ctrls.Auth.Login)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middlewares.AuthMiddleware(jwtUtil))
	{
		protected.GET("/profile", ctrls.Auth.GetProfile)

		// Event routes
		eventRoutes := protected.Group("/events")
		{
			eventRoutes.GET("", ctrls.Event.GetEvents)
			eventRoutes.POST("", ctrls.Event.CreateEvent)
			eventRoutes.POST("/filter", ctrls.Event.FilterEvents)

			// Event-specific routes
			eventRoutes.GET("/:id", ctrls.Event.GetEventByID)
			eventRoutes.PUT("/:id", ctrls.Event.UpdateEvent)
			eventRoutes.DELETE("/:id", ctrls.Event.DeleteEvent)
			eventRoutes.POST("/:id/participate", ctrls.Event.Participate)
			eventRoutes.DELETE("/:id/participate", ctrls.Event.CancelParticipation)

			// Comments for specific event
			eventRoutes.GET("/:id/comments", ctrls.Comment.GetComments)
			eventRoutes.POST("/:id/comments", ctrls.Comment.CreateComment)
		}

		// Comment routes (individual comments)
		protected.PUT("/comments/:commentId", ctrls.Comment.UpdateComment)
		protected.DELETE("/comments/:commentId", ctrls.Comment.DeleteComment)
		protected.POST("/comments/:commentId/vote", ctrls.Comment.VoteComment)

		// User-specific events
		protected.GET("/user/events", ctrls.Event.GetUserEvents)
		protected.GET("/user/participated", ctrls.Event.GetParticipatedEvents)

		// Notification routes
		protected.GET("/notifications", ctrls.Notification.GetNotifications)
		protected.PUT("/notifications/:id/read", ctrls.Notification.MarkAsRead)
		protected.POST("/notifications/mark-all-read", ctrls.Notification.MarkAllAsRead)

		// Admin routes
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middlewares.RoleMiddleware("admin"))
		{
			adminRoutes.GET("/dashboard", ctrls.Admin.GetAdminDashboard)
			adminRoutes.GET("/events", ctrls.Admin.GetAllEvents)
			adminRoutes.PUT("/events/:eventId/verify", ctrls.Admin.VerifyEvent)
			adminRoutes.PUT("/events/:eventId/reject", ctrls.Admin.RejectEvent)
			adminRoutes.DELETE("/events/:eventId", ctrls.Admin.DeleteEvent)
			adminRoutes.GET("/users", ctrls.Admin.GetAllUsers)
			adminRoutes.PUT("/users/:userId/block", ctrls.Admin.BlockUser)
			adminRoutes.PUT("/users/:userId/unblock", ctrls.Admin.UnblockUser)
			adminRoutes.DELETE("/comments/:commentId", ctrls.Admin.DeleteComment)
			adminRoutes.GET("/statistics", ctrls.Admin.GetStatistics)
		}
	}

	return router
}
