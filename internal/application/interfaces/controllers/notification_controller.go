package controllers

import (
	"net/http"
	"strconv"

	"auth-system/internal/application/interfaces"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService interfaces.NotificationService
}

func NewNotificationController(notificationService interfaces.NotificationService) *NotificationController {
	return &NotificationController{notificationService: notificationService}
}

func (c *NotificationController) GetNotifications(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	notifications, err := c.notificationService.GetNotifications(ctx.Request.Context(), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, notifications)
}

func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	notificationID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.notificationService.MarkAsRead(ctx.Request.Context(), uint(notificationID), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	err := c.notificationService.MarkAllAsRead(ctx.Request.Context(), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	count, err := c.notificationService.GetUnreadCount(ctx.Request.Context(), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"unread_count": count})
}
