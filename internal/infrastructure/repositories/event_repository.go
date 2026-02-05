package repositories

import (
	"context"
	"fmt"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) interfaces.EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *entities.Event) error {
	// Используем raw SQL для вставки с PostGIS
	query := `
		INSERT INTO events (
			title, description, event_date, location, type, 
			max_participants, price, address, is_verified, 
			is_active, creator_id, created_at, updated_at
		) VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	return r.db.WithContext(ctx).Raw(query,
		event.Title,
		event.Description,
		event.EventDate,
		event.Longitude, // ST_MakePoint принимает longitude first
		event.Latitude,
		event.Type,
		event.MaxParticipants,
		event.Price,
		event.Address,
		event.IsVerified,
		event.IsActive,
		event.CreatorID,
		event.CreatedAt,
		event.UpdatedAt,
	).Scan(&event.ID).Error
}

func (r *EventRepository) FindByID(ctx context.Context, id uint) (*entities.Event, error) {
	var event entities.Event

	// Загружаем базовую информацию о событии
	err := r.db.WithContext(ctx).
		Table("events").
		Select("events.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON events.creator_id = users.id").
		Where("events.id = ?", id).
		First(&event).Error

	if err != nil {
		return nil, err
	}

	// Ручной расчет количества участников
	var participantsCount int64
	r.db.WithContext(ctx).Table("event_participants").
		Where("event_id = ? AND status = 'going'", id).
		Count(&participantsCount)
	event.ParticipantsCount = int(participantsCount)

	// Получаем координаты из PostGIS
	err = r.db.WithContext(ctx).Raw(`
		SELECT 
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude 
		FROM events WHERE id = ?`, id).Scan(&event).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	// Загружаем теги
	var tags []entities.Tag
	err = r.db.WithContext(ctx).
		Table("tags").
		Select("tags.*").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("event_tags.event_id = ?", id).
		Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	event.Tags = tags

	// Загружаем медиа
	var media []entities.EventMedia
	err = r.db.WithContext(ctx).
		Table("event_media").
		Where("event_id = ?", id).
		Order("order_index").
		Find(&media).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	event.Media = media

	return &event, nil
}

func (r *EventRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]entities.Event, error) {
	var events []entities.Event

	// Базовый запрос
	query := r.db.WithContext(ctx).
		Table("events").
		Select("events.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON events.creator_id = users.id").
		Where("events.is_active = ?", true)

	// Фильтры
	if types, ok := filter["type"]; ok {
		if typeSlice, ok := types.([]string); ok && len(typeSlice) > 0 {
			query = query.Where("events.type IN ?", typeSlice)
		}
	}
	if date, ok := filter["date"]; ok {
		if dateTime, ok := date.(time.Time); ok && !dateTime.IsZero() {
			startOfDay := dateTime.Truncate(24 * time.Hour)
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("events.event_date BETWEEN ? AND ?", startOfDay, endOfDay)
		}
	}

	// Получаем события
	err := query.Order("events.created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	// Для каждого события получаем дополнительные данные
	for i := range events {
		// Количество участников
		var count int64
		r.db.WithContext(ctx).Table("event_participants").
			Where("event_id = ? AND status = 'going'", events[i].ID).
			Count(&count)
		events[i].ParticipantsCount = int(count)

		// Координаты
		err = r.db.WithContext(ctx).Raw(`
			SELECT 
				ST_Y(location::geometry) as latitude,
				ST_X(location::geometry) as longitude 
			FROM events WHERE id = ?`, events[i].ID).Scan(&events[i]).Error
		if err != nil {
			return nil, fmt.Errorf("failed to get coordinates for event %d: %w", events[i].ID, err)
		}

		// Теги
		var tags []entities.Tag
		err = r.db.WithContext(ctx).
			Table("tags").
			Select("tags.*").
			Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
			Where("event_tags.event_id = ?", events[i].ID).
			Find(&tags).Error
		if err == nil {
			events[i].Tags = tags
		}

		// Медиа
		var media []entities.EventMedia
		err = r.db.WithContext(ctx).
			Table("event_media").
			Where("event_id = ?", events[i].ID).
			Order("order_index").
			Find(&media).Error
		if err == nil {
			events[i].Media = media
		}
	}

	return events, nil
}

func (r *EventRepository) GetEventTags(ctx context.Context, eventID uint) ([]entities.Tag, error) {
	var tags []entities.Tag

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.*").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("event_tags.event_id = ?", eventID).
		Find(&tags).Error

	return tags, err
}

func (r *EventRepository) GetEventParticipants(ctx context.Context, eventID uint) ([]entities.EventParticipant, error) {
	var participants []entities.EventParticipant

	err := r.db.WithContext(ctx).
		Table("event_participants").
		Select("event_participants.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON event_participants.user_id = users.id").
		Where("event_participants.event_id = ?", eventID).
		Find(&participants).Error

	return participants, err
}

func (r *EventRepository) Update(ctx context.Context, event *entities.Event) error {
	// Обновляем только негеографические поля
	// Для обновления location нужно использовать отдельный запрос с PostGIS
	updates := map[string]interface{}{
		"title":            event.Title,
		"description":      event.Description,
		"event_date":       event.EventDate,
		"type":             event.Type,
		"max_participants": event.MaxParticipants,
		"price":            event.Price,
		"address":          event.Address,
		"is_verified":      event.IsVerified,
		"is_active":        event.IsActive,
		"updated_at":       time.Now(),
	}

	// Если нужно обновить координаты
	if event.Latitude != 0 && event.Longitude != 0 {
		// Используем raw SQL для обновления location
		err := r.db.WithContext(ctx).Exec(`
			UPDATE events 
			SET location = ST_SetSRID(ST_MakePoint(?, ?), 4326)
			WHERE id = ?`,
			event.Longitude, event.Latitude, event.ID).Error
		if err != nil {
			return err
		}
	}

	return r.db.WithContext(ctx).Model(&entities.Event{}).
		Where("id = ?", event.ID).
		Updates(updates).Error
}

func (r *EventRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&entities.Event{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

func (r *EventRepository) GetByCreator(ctx context.Context, creatorID uint) ([]entities.Event, error) {
	var events []entities.Event
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Where("creator_id = ? AND is_active = ?", creatorID, true).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *EventRepository) GetParticipatedEvents(ctx context.Context, userID uint) ([]entities.Event, error) {
	var events []entities.Event

	// Получаем мероприятия, в которых участвует пользователь
	err := r.db.WithContext(ctx).
		Joins("JOIN event_participants ep ON events.id = ep.event_id").
		Preload("Creator").
		Preload("Tags").
		Where("ep.user_id = ? AND ep.status = 'going' AND events.is_active = ?", userID, true).
		Order("ep.joined_at DESC").
		Find(&events).Error

	return events, err
}

func (r *EventRepository) VerifyEvent(ctx context.Context, eventID uint) error {
	return r.db.WithContext(ctx).
		Model(&entities.Event{}).
		Where("id = ?", eventID).
		Update("is_verified", true).Error
}

func (r *EventRepository) RejectEvent(ctx context.Context, eventID uint, reason string) error {
	return r.db.WithContext(ctx).
		Model(&entities.Event{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"is_active":   false,
			"is_verified": false,
		}).Error
}

func (r *EventRepository) GetPendingEvents(ctx context.Context) ([]entities.Event, error) {
	var events []entities.Event
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Where("is_verified = ? AND is_active = ?", false, true).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *EventRepository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	// Этот метод уже не нужен, так как статистика разбита на методы в AdminRepository
	// Оставляем пустую реализацию для совместимости
	return make(map[string]interface{}), nil
}

func (r *EventRepository) AddParticipant(ctx context.Context, eventID, userID uint) error {
	participant := &entities.EventParticipant{
		EventID:  eventID,
		UserID:   userID,
		Status:   "going",
		JoinedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(participant).Error
}

func (r *EventRepository) RemoveParticipant(ctx context.Context, eventID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Delete(&entities.EventParticipant{}).Error
}

func (r *EventRepository) GetParticipantCount(ctx context.Context, eventID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.EventParticipant{}).
		Where("event_id = ? AND status = 'going'", eventID).
		Count(&count).Error
	return count, err
}

func (r *EventRepository) IsParticipant(ctx context.Context, eventID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.EventParticipant{}).
		Where("event_id = ? AND user_id = ? AND status = 'going'", eventID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *EventRepository) AddTags(ctx context.Context, eventID uint, tags []string) error {
	// Начинаем транзакцию
	tx := r.db.WithContext(ctx).Begin()

	for _, tagName := range tags {
		var tag entities.Tag
		// Ищем или создаем тег
		if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
			// Тег не найден, создаем новый
			tag = entities.Tag{
				Name:      tagName,
				Slug:      slug.Make(tagName),
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&tag).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		// Создаем связь события с тегом
		eventTag := entities.EventTag{
			EventID: eventID,
			TagID:   tag.ID,
		}
		if err := tx.Create(&eventTag).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *EventRepository) GetTopEvents(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// Этот метод теперь находится в AdminRepository
	// Оставляем пустую реализацию для совместимости
	return nil, nil
}
