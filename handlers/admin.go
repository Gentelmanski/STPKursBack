package handlers

import (
	"net/http"
	"strconv"
	"time"

	"auth-system/database"
	"auth-system/models"

	"github.com/gin-gonic/gin"
)

// AdminGetEvents - получение всех мероприятий для администратора
func AdminGetEvents(c *gin.Context) {
	var events []models.Event

	if err := database.DB.Preload("Creator").
		Preload("Tags").
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// AdminGetUsers - получение всех пользователей для администратора
func AdminGetUsers(c *gin.Context) {
	var users []models.User

	if err := database.DB.Select("id, username, email, role, created_at, is_blocked, last_online").
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// AdminGetStatistics - получение статистики для администратора
func AdminGetStatistics(c *gin.Context) {
	var stats struct {
		TotalUsers         int64 `json:"total_users"`
		TotalEvents        int64 `json:"total_events"`
		ActiveEvents       int64 `json:"active_events"`
		VerifiedEvents     int64 `json:"verified_events"`
		TotalComments      int64 `json:"total_comments"`
		TodayRegistrations int64 `json:"today_registrations"`
		OnlineUsers        int64 `json:"online_users"`
	}

	// Общее количество пользователей
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// Общее количество мероприятий
	database.DB.Model(&models.Event{}).Count(&stats.TotalEvents)

	// Активные мероприятия
	database.DB.Model(&models.Event{}).Where("is_active = ?", true).Count(&stats.ActiveEvents)

	// Верифицированные мероприятия
	database.DB.Model(&models.Event{}).Where("is_verified = ?", true).Count(&stats.VerifiedEvents)

	// Общее количество комментариев
	database.DB.Model(&models.Comment{}).Where("is_deleted = ?", false).Count(&stats.TotalComments)

	// Регистрации сегодня
	today := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&models.User{}).Where("created_at >= ?", today).Count(&stats.TodayRegistrations)

	// Пользователи онлайн (последние 15 минут)
	fifteenMinutesAgo := time.Now().Add(-15 * time.Minute)
	database.DB.Model(&models.User{}).Where("last_online >= ?", fifteenMinutesAgo).Count(&stats.OnlineUsers)

	// Получаем топ мероприятий по участникам
	var topEvents []struct {
		EventID      uint   `json:"event_id"`
		Title        string `json:"title"`
		Participants int64  `json:"participants"`
	}

	database.DB.Raw(`
        SELECT e.id as event_id, e.title, COUNT(ep.user_id) as participants
        FROM events e
        LEFT JOIN event_participants ep ON e.id = ep.event_id AND ep.status = 'going'
        WHERE e.is_active = true
        GROUP BY e.id, e.title
        ORDER BY participants DESC
        LIMIT 10
    `).Scan(&topEvents)

	// Получаем мероприятия для верификации
	var pendingEvents []models.Event
	database.DB.Preload("Creator").
		Where("is_verified = ? AND is_active = ?", false, true).
		Order("created_at DESC").
		Find(&pendingEvents)

	c.JSON(http.StatusOK, gin.H{
		"stats":          stats,
		"top_events":     topEvents,
		"pending_events": pendingEvents,
	})
}

// AdminVerifyEvent - верификация мероприятия администратором
func AdminVerifyEvent(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("eventId"))
	adminID, _ := c.Get("user_id")

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	event.IsVerified = true
	database.DB.Save(&event)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "verify_event",
		TargetID:    uint(eventID),
		TargetType:  "event",
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	// Создаем уведомление для создателя мероприятия
	notification := models.Notification{
		UserID:    event.CreatorID,
		Message:   "Ваше мероприятие \"" + event.Title + "\" было верифицировано администратором",
		Type:      "event_verified",
		Read:      false,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Event verified"})
}

// AdminRejectEvent - отклонение мероприятия администратором
func AdminRejectEvent(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("eventId"))
	adminID, _ := c.Get("user_id")

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	event.IsActive = false
	database.DB.Save(&event)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "reject_event",
		TargetID:    uint(eventID),
		TargetType:  "event",
		Reason:      req.Reason,
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	// Создаем уведомление для создателя мероприятия
	notification := models.Notification{
		UserID:    event.CreatorID,
		Message:   "Ваше мероприятие \"" + event.Title + "\" было отклонено администратором. Причина: " + req.Reason,
		Type:      "event_rejected",
		Read:      false,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Event rejected"})
}

// AdminDeleteEvent - удаление мероприятия администратором
func AdminDeleteEvent(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("eventId"))
	adminID, _ := c.Get("user_id")

	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	event.IsActive = false
	database.DB.Save(&event)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "delete_event",
		TargetID:    uint(eventID),
		TargetType:  "event",
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	// Создаем уведомление для создателя мероприятия
	notification := models.Notification{
		UserID:    event.CreatorID,
		Message:   "Ваше мероприятие \"" + event.Title + "\" было удалено администратором",
		Type:      "event_deleted",
		Read:      false,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted"})
}

// AdminBlockUser - блокировка пользователя администратором
func AdminBlockUser(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("userId"))
	adminID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsBlocked = true
	database.DB.Save(&user)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "block_user",
		TargetID:    uint(userID),
		TargetType:  "user",
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	// Создаем уведомление для пользователя
	notification := models.Notification{
		UserID:    uint(userID),
		Message:   "Ваш аккаунт был заблокирован администратором",
		Type:      "system",
		Read:      false,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}

// AdminUnblockUser - разблокировка пользователя администратором
func AdminUnblockUser(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("userId"))
	adminID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsBlocked = false
	database.DB.Save(&user)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "unblock_user",
		TargetID:    uint(userID),
		TargetType:  "user",
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked"})
}

// AdminDeleteComment - удаление комментария администратором
func AdminDeleteComment(c *gin.Context) {
	commentID, _ := strconv.Atoi(c.Param("commentId"))
	adminID, _ := c.Get("user_id")

	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	comment.IsDeleted = true
	database.DB.Save(&comment)

	// Создаем запись о действии администратора
	action := models.AdminAction{
		AdminID:     adminID.(uint),
		ActionType:  "delete_comment",
		TargetID:    uint(commentID),
		TargetType:  "comment",
		PerformedAt: time.Now(),
	}
	database.DB.Create(&action)

	// Создаем уведомление для автора комментария
	notification := models.Notification{
		UserID:    comment.UserID,
		Message:   "Ваш комментарий был удален администратором",
		Type:      "system",
		Read:      false,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}
