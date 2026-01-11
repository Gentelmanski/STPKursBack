package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"auth-system/database"
	"auth-system/models"

	"github.com/gin-gonic/gin"
)

func GetComments(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("id")) // Изменено с :eventId на :id

	var comments []models.Comment
	if err := database.DB.Preload("User").
		Preload("Replies", "is_deleted = ?", false).
		Preload("Replies.User").
		Where("event_id = ? AND parent_id IS NULL AND is_deleted = ?", eventID, false).
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func CreateComment(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("id")) // Изменено с :eventId на :id
	userID, _ := c.Get("user_id")

	var req struct {
		Content  string `json:"content" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем существование мероприятия
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	comment := models.Comment{
		Content:   req.Content,
		EventID:   uint(eventID),
		UserID:    userID.(uint),
		ParentID:  req.ParentID,
		Score:     0,
		IsDeleted: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Загружаем созданный комментарий с пользователем
	database.DB.Preload("User").First(&comment, comment.ID)

	// Создаем уведомление для создателя мероприятия
	if event.CreatorID != userID.(uint) {
		notification := models.Notification{
			UserID:    event.CreatorID,
			Message:   fmt.Sprintf("Новый комментарий к вашему мероприятию: %s", event.Title),
			Type:      "comment_added",
			Read:      false,
			CreatedAt: time.Now(),
		}
		database.DB.Create(&notification)
	}

	c.JSON(http.StatusCreated, comment)
}

func UpdateComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentId")) // Изменено с :id на :commentId
	userID, _ := c.Get("user_id")

	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this comment"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment.Content = req.Content
	comment.UpdatedAt = time.Now()

	if err := database.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
}

func DeleteComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentId")) // Изменено с :id на :commentId
	userID, _ := c.Get("user_id")

	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this comment"})
		return
	}

	comment.IsDeleted = true
	comment.UpdatedAt = time.Now()

	if err := database.DB.Save(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func VoteComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentId")) // Изменено с :commentId на :commentId
	userID, _ := c.Get("user_id")

	var req struct {
		VoteType string `json:"vote_type" binding:"required,oneof=upvote downvote"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем существование комментария
	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Проверяем, не голосовал ли уже пользователь
	var existingVote models.CommentVote
	if err := database.DB.Where("user_id = ? AND comment_id = ?", userID, commentID).
		First(&existingVote).Error; err == nil {
		// Если голос уже есть, обновляем его
		existingVote.VoteType = req.VoteType
		existingVote.VotedAt = time.Now()
		database.DB.Save(&existingVote)
	} else {
		// Создаем новый голос
		vote := models.CommentVote{
			UserID:    userID.(uint),
			CommentID: uint(commentID),
			VoteType:  req.VoteType,
			VotedAt:   time.Now(),
		}
		database.DB.Create(&vote)
	}

	// Пересчитываем счет комментария
	var upvotes, downvotes int64
	database.DB.Model(&models.CommentVote{}).
		Where("comment_id = ? AND vote_type = ?", commentID, "upvote").
		Count(&upvotes)
	database.DB.Model(&models.CommentVote{}).
		Where("comment_id = ? AND vote_type = ?", commentID, "downvote").
		Count(&downvotes)

	comment.Score = int(upvotes - downvotes)
	database.DB.Save(&comment)

	c.JSON(http.StatusOK, gin.H{"score": comment.Score})
}
