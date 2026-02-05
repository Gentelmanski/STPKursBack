package controllers

import (
	"net/http"
	"strconv"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	adminService interfaces.AdminService
}

func NewAdminController(adminService interfaces.AdminService) *AdminController {
	return &AdminController{adminService: adminService}
}

func (c *AdminController) GetAdminDashboard(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Admin Dashboard",
		"data":    "Admin-specific content here",
	})
}

func (c *AdminController) GetAllEvents(ctx *gin.Context) {
	events, err := c.adminService.GetAllEvents(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}

func (c *AdminController) VerifyEvent(ctx *gin.Context) {
	eventID, err := strconv.ParseUint(ctx.Param("eventId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.VerifyEvent(ctx.Request.Context(), uint(eventID), adminID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Event verified"})
}

func (c *AdminController) RejectEvent(ctx *gin.Context) {
	eventID, err := strconv.ParseUint(ctx.Param("eventId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var req dto.RejectEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.RejectEvent(ctx.Request.Context(), uint(eventID), adminID.(uint), req.Reason)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Event rejected"})
}

func (c *AdminController) DeleteEvent(ctx *gin.Context) {
	eventID, err := strconv.ParseUint(ctx.Param("eventId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.DeleteEvent(ctx.Request.Context(), uint(eventID), adminID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Event deleted"})
}

func (c *AdminController) GetAllUsers(ctx *gin.Context) {
	users, err := c.adminService.GetAllUsers(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, users)
}

func (c *AdminController) BlockUser(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("userId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.BlockUser(ctx.Request.Context(), uint(userID), adminID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}

func (c *AdminController) UnblockUser(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("userId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.UnblockUser(ctx.Request.Context(), uint(userID), adminID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User unblocked"})
}

func (c *AdminController) DeleteComment(ctx *gin.Context) {
	commentID, err := strconv.ParseUint(ctx.Param("commentId"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	adminID, _ := ctx.Get("user_id")
	err = c.adminService.DeleteComment(ctx.Request.Context(), uint(commentID), adminID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}

func (c *AdminController) GetStatistics(ctx *gin.Context) {
	stats, err := c.adminService.GetStatistics(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

func (c *AdminController) GetPendingEvents(ctx *gin.Context) {
	events, err := c.adminService.GetPendingEvents(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}
