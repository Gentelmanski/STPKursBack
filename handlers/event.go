package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"auth-system/database"
	"auth-system/models"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type CreateEventRequest struct {
	Title           string    `json:"title" binding:"required"`
	Description     string    `json:"description" binding:"required"`
	EventDate       time.Time `json:"event_date" binding:"required"`
	Latitude        float64   `json:"latitude" binding:"required"`
	Longitude       float64   `json:"longitude" binding:"required"`
	Type            string    `json:"type" binding:"required,oneof=concert exhibition meetup workshop sport festival other"`
	MaxParticipants *int      `json:"max_participants"`
	Price           float64   `json:"price"`
	Tags            []string  `json:"tags"`
	Address         string    `json:"address"`
}

type UpdateEventRequest struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	EventDate       time.Time `json:"event_date"`
	Type            string    `json:"type" oneof:"concert,exhibition,meetup,workshop,sport,festival,other"`
	MaxParticipants *int      `json:"max_participants"`
	Price           float64   `json:"price"`
}

func GetEvents(c *gin.Context) {
	var events []models.Event

	query := database.DB.Model(&models.Event{}).
		Preload("Creator").
		Preload("Tags").
		Where("is_active = ?", true)

	// Фильтрация по типу
	if types := c.QueryArray("type"); len(types) > 0 {
		query = query.Where("type IN ?", types)
	}

	// Фильтрация по дате
	if date := c.Query("date"); date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err == nil {
			startOfDay := parsedDate
			endOfDay := parsedDate.Add(24 * time.Hour)
			query = query.Where("event_date BETWEEN ? AND ?", startOfDay, endOfDay)
		}
	}

	// Поиск по тегам
	if tags := c.Query("tags"); tags != "" {
		tagList := strings.Split(tags, ",")
		query = query.Joins("JOIN event_tags ON events.id = event_tags.event_id").
			Joins("JOIN tags ON event_tags.tag_id = tags.id").
			Where("tags.name IN ?", tagList).
			Group("events.id")
	}

	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем PostGIS geometry в latitude/longitude и получаем количество участников
	for i := range events {
		type Coordinates struct {
			Latitude  float64
			Longitude float64
		}

		var coords Coordinates
		database.DB.Raw("SELECT ST_Y(location::geometry) as latitude, ST_X(location::geometry) as longitude FROM events WHERE id = ?", events[i].ID).
			Scan(&coords)

		events[i].Latitude = coords.Latitude
		events[i].Longitude = coords.Longitude

		// Подсчет участников
		var participantsCount int64
		database.DB.Model(&models.EventParticipant{}).
			Where("event_id = ? AND status = 'going'", events[i].ID).
			Count(&participantsCount)
		events[i].ParticipantsCount = int(participantsCount)
	}

	c.JSON(http.StatusOK, events)
}

func GetEventByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var event models.Event
	if err := database.DB.Preload("Creator").
		Preload("Tags").
		Preload("Media").
		Preload("Comments", "parent_id IS NULL AND is_deleted = false").
		Preload("Comments.User").
		Preload("Comments.Replies", "is_deleted = false").
		Preload("Comments.Replies.User").
		First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if !event.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Преобразуем PostGIS geometry в latitude/longitude
	type Coordinates struct {
		Latitude  float64
		Longitude float64
	}

	var coords Coordinates
	database.DB.Raw("SELECT ST_Y(location::geometry) as latitude, ST_X(location::geometry) as longitude FROM events WHERE id = ?", event.ID).
		Scan(&coords)

	event.Latitude = coords.Latitude
	event.Longitude = coords.Longitude

	// Подсчет участников
	var participantsCount int64
	database.DB.Model(&models.EventParticipant{}).
		Where("event_id = ? AND status = 'going'", event.ID).
		Count(&participantsCount)
	event.ParticipantsCount = int(participantsCount)

	// Получаем участников
	var participants []models.EventParticipant
	database.DB.Preload("User").Where("event_id = ?", event.ID).Find(&participants)
	event.Participants = participants

	c.JSON(http.StatusOK, event)
}

func CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// Создаем PostGIS POINT из координат
	location := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)

	event := models.Event{
		Title:           req.Title,
		Description:     req.Description,
		EventDate:       req.EventDate,
		Location:        location,
		Type:            req.Type,
		MaxParticipants: req.MaxParticipants,
		Price:           req.Price,
		Address:         req.Address,
		CreatorID:       userID.(uint),
		IsVerified:      false,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Добавляем теги
	for _, tagName := range req.Tags {
		var tag models.Tag
		if err := database.DB.Where("name = ?", tagName).First(&tag).Error; err != nil {
			tag = models.Tag{
				Name: tagName,
				Slug: slug.Make(tagName),
			}
			database.DB.Create(&tag)
		}
		database.DB.Create(&models.EventTag{
			EventID: event.ID,
			TagID:   tag.ID,
		})
	}

	// Создаем уведомление для администраторов
	var admins []models.User
	database.DB.Where("role = ?", "admin").Find(&admins)

	for _, admin := range admins {
		notification := models.Notification{
			UserID:    admin.ID,
			Message:   fmt.Sprintf("Новое мероприятие создано: %s", event.Title),
			Type:      "event_created",
			Read:      false,
			CreatedAt: time.Now(),
		}
		database.DB.Create(&notification)
	}

	event.Latitude = req.Latitude
	event.Longitude = req.Longitude

	c.JSON(http.StatusCreated, event)
}

func UpdateEvent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	var event models.Event
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Проверяем права
	if event.CreatorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this event"})
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if !req.EventDate.IsZero() {
		updates["event_date"] = req.EventDate
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.MaxParticipants != nil {
		updates["max_participants"] = *req.MaxParticipants
	}
	if req.Price >= 0 {
		updates["price"] = req.Price
	}
	updates["updated_at"] = time.Now()

	if err := database.DB.Model(&event).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

func DeleteEvent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	var event models.Event
	if err := database.DB.First(&event, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Проверяем права
	if event.CreatorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this event"})
		return
	}

	event.IsActive = false
	event.UpdatedAt = time.Now()

	if err := database.DB.Save(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
}

func ParticipateEvent(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	// Проверяем существование мероприятия
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if !event.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event is not active"})
		return
	}

	// Проверяем максимальное количество участников
	if event.MaxParticipants != nil {
		var count int64
		database.DB.Model(&models.EventParticipant{}).
			Where("event_id = ? AND status = 'going'", eventID).
			Count(&count)

		if int(count) >= *event.MaxParticipants {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Event is full"})
			return
		}
	}

	// Проверяем, существует ли уже запись о участии
	var existingParticipant models.EventParticipant
	result := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).
		First(&existingParticipant)

	if result.Error == nil {
		// Запись уже существует - обновляем статус
		existingParticipant.Status = "going"
		existingParticipant.JoinedAt = time.Now()

		if err := database.DB.Save(&existingParticipant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Записи нет - создаем новую
		participant := models.EventParticipant{
			EventID:  uint(eventID),
			UserID:   userID.(uint),
			Status:   "going",
			JoinedAt: time.Now(),
		}

		if err := database.DB.Create(&participant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Создаем уведомление для создателя мероприятия
	if event.CreatorID != userID.(uint) {
		notification := models.Notification{
			UserID:    event.CreatorID,
			Message:   fmt.Sprintf("Новый участник присоединился к вашему мероприятию: %s", event.Title),
			Type:      "participation",
			Read:      false,
			CreatedAt: time.Now(),
		}
		database.DB.Create(&notification)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined event"})
}

func CancelParticipation(c *gin.Context) {
	eventID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")

	// Вместо удаления записи, меняем статус на 'declined'
	var participant models.EventParticipant
	result := database.DB.Where("event_id = ? AND user_id = ?", eventID, userID).
		First(&participant)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Participation not found"})
		return
	}

	participant.Status = "declined"

	if err := database.DB.Save(&participant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully cancelled participation"})
}

func GetEventsByUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var events []models.Event

	// Загружаем мероприятия, созданные именно этим пользователем
	if err := database.DB.
		Preload("Creator").
		Preload("Tags").
		Where("creator_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем PostGIS geometry в latitude/longitude
	for i := range events {
		type Coordinates struct {
			Latitude  float64
			Longitude float64
		}

		var coords Coordinates
		database.DB.Raw(`
            SELECT 
                ST_Y(ST_Transform(location::geometry, 4326)) as latitude,
                ST_X(ST_Transform(location::geometry, 4326)) as longitude 
            FROM events WHERE id = ?
        `, events[i].ID).Scan(&coords)

		events[i].Latitude = coords.Latitude
		events[i].Longitude = coords.Longitude

		// Подсчет участников
		var participantsCount int64
		database.DB.Model(&models.EventParticipant{}).
			Where("event_id = ? AND status = 'going'", events[i].ID).
			Count(&participantsCount)
		events[i].ParticipantsCount = int(participantsCount)
	}

	c.JSON(http.StatusOK, events)
}

func GetParticipatedEvents(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var participations []models.EventParticipant
	if err := database.DB.Where("user_id = ? AND status = 'going'", userID).
		Preload("Event", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).
				Preload("Creator").
				Preload("Tags")
		}).
		Order("joined_at DESC").
		Find(&participations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Преобразуем координаты для каждого мероприятия
	for i := range participations {
		// Проверяем, что Event был загружен (ID не равен 0)
		if participations[i].Event.ID != 0 {
			type Coordinates struct {
				Latitude  float64
				Longitude float64
			}

			var coords Coordinates
			database.DB.Raw(`
				SELECT 
					ST_Y(ST_Transform(location::geometry, 4326)) as latitude,
					ST_X(ST_Transform(location::geometry, 4326)) as longitude 
				FROM events WHERE id = ?
			`, participations[i].Event.ID).Scan(&coords)

			participations[i].Event.Latitude = coords.Latitude
			participations[i].Event.Longitude = coords.Longitude

			// Подсчет участников
			var participantsCount int64
			database.DB.Model(&models.EventParticipant{}).
				Where("event_id = ? AND status = 'going'", participations[i].Event.ID).
				Count(&participantsCount)
			participations[i].Event.ParticipantsCount = int(participantsCount)
		}
	}

	c.JSON(http.StatusOK, participations)
}
