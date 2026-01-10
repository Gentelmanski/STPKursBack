package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to User Dashboard",
		"data":    "User-specific content here",
	})
}

func GetAdminDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Admin Dashboard",
		"data":    "Admin-specific content here",
	})
}
