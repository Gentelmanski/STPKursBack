package handlers

import (
	"auth-system/database"
	"auth-system/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var notifications []models.Notification
	if err := database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func MarkNotificationAsRead(c *gin.Context) {
	notificationID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	var notification models.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	notification.Read = true
	database.DB.Save(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func MarkAllNotificationsAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true)

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}
