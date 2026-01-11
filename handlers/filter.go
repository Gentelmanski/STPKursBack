package handlers

import (
	"auth-system/database"
	"auth-system/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type FilterRequest struct {
	Type []string  `json:"type"`
	Date time.Time `json:"date"`
}

// Структура для получения координат из базы данных
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func FilterEvents(c *gin.Context) {
	var req FilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var events []models.Event

	query := database.DB.Model(&models.Event{}).
		Preload("Creator").
		Preload("Tags").
		Where("is_active = ?", true)

	// Фильтрация по типу
	if len(req.Type) > 0 {
		query = query.Where("type IN ?", req.Type)
	}

	// Фильтрация по дате
	if !req.Date.IsZero() {
		startOfDay := req.Date.Truncate(24 * time.Hour)
		endOfDay := startOfDay.Add(24 * time.Hour)
		query = query.Where("event_date BETWEEN ? AND ?", startOfDay, endOfDay)
	}

	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем координаты и получаем количество участников
	for i := range events {
		var coords Coordinates
		database.DB.Raw("SELECT ST_Y(location::geometry) as latitude, ST_X(location::geometry) as longitude FROM events WHERE id = ?", events[i].ID).
			Scan(&coords)
		events[i].Latitude = coords.Latitude
		events[i].Longitude = coords.Longitude

		var participantsCount int64
		database.DB.Model(&models.EventParticipant{}).
			Where("event_id = ? AND status = 'going'", events[i].ID).
			Count(&participantsCount)
		events[i].ParticipantsCount = int(participantsCount)
	}

	c.JSON(http.StatusOK, events)
}
