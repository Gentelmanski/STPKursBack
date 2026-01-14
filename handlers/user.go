package handlers

import (
	"net/http"

	"auth-system/database"
	"auth-system/models"

	"github.com/gin-gonic/gin"
)

func GetUserDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Получаем статистику пользователя
	var stats struct {
		CreatedEvents      int64 `json:"created_events"`
		ParticipatedEvents int64 `json:"participated_events"`
		Comments           int64 `json:"comments"`
		Rating             int64 `json:"rating"`
	}

	// Количество созданных мероприятий
	database.DB.Model(&models.Event{}).
		Where("creator_id = ? AND is_active = ?", userID, true).
		Count(&stats.CreatedEvents)

	// Количество мероприятий, в которых участвует пользователь
	database.DB.Model(&models.EventParticipant{}).
		Where("user_id = ? AND status = 'going'", userID).
		Count(&stats.ParticipatedEvents)

	// Количество комментариев пользователя
	database.DB.Model(&models.Comment{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&stats.Comments)

	// Рейтинг (можно вычислить как сумму голосов за комментарии)
	var rating struct {
		TotalScore int64
	}
	database.DB.Raw(`
        SELECT COALESCE(SUM(score), 0) as total_score 
        FROM comments 
        WHERE user_id = ? AND is_deleted = false
    `, userID).Scan(&rating)

	stats.Rating = rating.TotalScore

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to User Dashboard",
		"stats":   stats,
	})
}

func GetAdminDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to Admin Dashboard",
		"data":    "Admin-specific content here",
	})
}
